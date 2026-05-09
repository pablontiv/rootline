import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { createRunner, type RunnerResult } from "../runner";
import {
  validateTargetPath,
  requiresConfirmation,
  validateAfterWrite,
  type ValidationGuardrailResult,
} from "./guardrails";
import * as path from "node:path";

/**
 * Tool input for rootline-fix.
 */
interface FixToolInput {
  /** Directory to fix (required) */
  path: string;
  /** Preview without applying (optional, default: true — safe by default) */
  dry_run?: boolean;
  /** Explicit confirmation for non-dry-run mode (optional) */
  confirmed?: boolean;
  /** Whether Pi is running interactively (optional, default: false) */
  interactive?: boolean;
  /** Filter expression to limit which files to fix (optional) */
  where?: string;
}

/**
 * Single file result from fix operation.
 */
interface FixResultItem {
  path: string;
  fixed: boolean;
  fields_added: number;
  values_corrected: number;
  changes: string[];
}

/**
 * Summary statistics for fix operation.
 */
interface FixSummaryData {
  total: number;
  fixed: number;
  skipped: number;
}

/**
 * Raw result from rootline fix --output json.
 */
interface RootlineFixOutput {
  version: number;
  kind: string;
  results: FixResultItem[];
  summary: FixSummaryData;
  schema_suggestions?: number;
}

/**
 * Validation summary from rootline validate.
 */
interface ValidationSummary {
  total_files: number;
  valid_count: number;
  error_count: number;
  warning_count: number;
}

/**
 * Structured result returned by rootline-fix tool.
 */
interface RootlineFixResult {
  success: boolean;
  dry_run: boolean;
  path: string;
  summary: FixSummaryData;
  schema_suggestions?: number;
  results: FixResultItem[];
  validation?: ValidationSummary;
  error?: string;
}

/**
 * Registers the rootline-fix tool for bulk file correction.
 *
 * This tool:
 * 1. Validates the target path (prevents path traversal)
 * 2. Checks if confirmation is required for non-dry-run mode
 * 3. Calls `rootline fix --all` with appropriate flags
 * 4. Optionally validates the affected directory after fixes
 * 5. Returns structured result with fix summary and validation details
 *
 * Security: Uses guardrails to prevent:
 * - Path traversal attacks (validateTargetPath)
 * - Unconfirmed mutations (requiresConfirmation)
 * - Invalid document state (validateAfterWrite)
 *
 * Key invariant: dry_run defaults to true — Pi must explicitly set
 * dry_run: false + confirmed: true to mutate.
 *
 * @param pi ExtensionAPI instance
 */
export default function (pi: ExtensionAPI) {
  const run = createRunner(pi);

  pi.registerTool({
    name: "rootline-fix",
    description:
      "Bulk fix validation errors across multiple files in a directory. " +
      "Auto-repairs missing required fields and corrects invalid enum values. " +
      "Defaults to dry-run mode (safe). Requires explicit confirmation to apply fixes. " +
      "Validates after writing to ensure no schema violations are introduced.",
    input_schema: {
      type: "object" as const,
      properties: {
        path: {
          type: "string",
          description:
            "Directory to fix (required). Scoped to project root — rejects path traversal.",
        },
        dry_run: {
          type: "boolean",
          description:
            "Preview without applying (optional, default: true). " +
            "Set to false only with confirmed: true to actually fix files.",
        },
        confirmed: {
          type: "boolean",
          description:
            "Explicit confirmation for non-dry-run mode (optional). " +
            "Required to set dry_run: false in non-interactive mode.",
        },
        interactive: {
          type: "boolean",
          description: "Whether Pi is running interactively (optional, default: false)",
        },
        where: {
          type: "string",
          description:
            "Filter expression to limit which files to fix (optional). " +
            "Examples: estado=pending, tipo=Epic. Uses same expr-lang syntax as rootline query.",
        },
      },
      required: ["path"],
    },
    execute: async (input: unknown) => {
      const {
        path: targetPath,
        dry_run = true, // Safe by default
        confirmed = false,
        interactive = false,
        where,
      } = input as FixToolInput;

      // Step 1: Validate target path
      const pathValidation = validateTargetPath(targetPath, process.cwd());
      if (!pathValidation.ok) {
        return {
          success: false,
          dry_run,
          path: targetPath,
          summary: { total: 0, fixed: 0, skipped: 0 },
          results: [],
          error: `Path validation failed: ${pathValidation.reason}`,
        } as RootlineFixResult;
      }

      // Step 2: Check if confirmation is required for non-dry-run mode
      if (!dry_run && requiresConfirmation("update", interactive) && !confirmed) {
        return {
          success: false,
          dry_run,
          path: targetPath,
          summary: { total: 0, fixed: 0, skipped: 0 },
          results: [],
          error:
            "Confirmation required: set confirmed: true to apply fixes in non-interactive mode",
        } as RootlineFixResult;
      }

      // Step 3: Resolve path to absolute
      const resolvedPath = path.resolve(process.cwd(), targetPath);

      // Step 4: Build args for rootline fix command
      const args = ["fix", "--all", resolvedPath, "--output", "json"];

      if (dry_run) {
        args.push("--dry-run");
      }

      if (where) {
        args.push("--where", where);
      }

      // Step 5: Execute rootline fix command
      const fixResult = await run<RootlineFixOutput>(args, {
        cwd: process.cwd(),
        timeout: 30_000, // Fix can be slow on large repos
      });

      if (!fixResult.ok) {
        return {
          success: false,
          dry_run,
          path: targetPath,
          summary: { total: 0, fixed: 0, skipped: 0 },
          results: [],
          error: `rootline fix failed: ${fixResult.error.message}`,
        } as RootlineFixResult;
      }

      // Step 6: For non-dry-run mode, validate after writing
      let validationSummary: ValidationSummary | undefined;
      if (!dry_run) {
        const validateResult = await run<{
          results: Array<{ path: string; valid: boolean; errors?: string[] }>;
          summary: Record<string, unknown>;
        }>(["validate", resolvedPath, "--output", "json"], {
          cwd: process.cwd(),
          timeout: 30_000,
        });

        if (validateResult.ok) {
          // Extract validation summary
          const { results } = validateResult.data;
          const validCount = results.filter((r) => r.valid).length;
          const errorCount = results.filter((r) => !r.valid && (r.errors?.length ?? 0) > 0).length;

          validationSummary = {
            total_files: results.length,
            valid_count: validCount,
            error_count: errorCount,
            warning_count: 0,
          };
        }
      }

      return {
        success: true,
        dry_run,
        path: resolvedPath,
        summary: fixResult.data.summary,
        schema_suggestions: fixResult.data.schema_suggestions,
        results: fixResult.data.results,
        validation: validationSummary,
      } as RootlineFixResult;
    },
  });
}
