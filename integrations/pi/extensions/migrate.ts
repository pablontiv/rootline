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
 * Tool input for rootline-migrate.
 */
interface MigrateToolInput {
  /** Target directory path to migrate (required) */
  path: string;
  /** Dry run mode: preview changes without writing (optional, default: true for safety) */
  dry_run?: boolean;
  /** Explicit confirmation for mutations in non-interactive mode (optional) */
  confirmed?: boolean;
  /** Whether Pi is running interactively (optional, default: false) */
  interactive?: boolean;
  /** Field rename in format "old_field=new_field" (mutually exclusive with split, scaffold) */
  rename?: string;
  /** Split flat .stem to hierarchical per-level files (mutually exclusive with rename, scaffold) */
  split?: boolean;
  /** Scaffold missing required section fields (mutually exclusive with rename, split) */
  scaffold?: boolean;
}

/**
 * Structured result returned by rootline-migrate tool.
 */
interface RootlineMigrateResult {
  dry_run: boolean;
  path: string;
  operation?: string;
  changes_detected: boolean;
  results?: Record<string, unknown>;
  summary?: Record<string, unknown>;
  validation?: ValidationGuardrailResult;
  error?: string;
}

/**
 * Registers the rootline-migrate tool for schema migration operations.
 *
 * This tool:
 * 1. Validates the target path (prevents path traversal)
 * 2. Enforces mutual exclusion of rename/split/scaffold
 * 3. Defaults to dry_run=true for safety
 * 4. Checks if confirmation is required for non-dry-run operations
 * 5. Calls `rootline migrate` with appropriate flags
 * 6. Validates affected roadmap/docs after mutation
 * 7. Returns structured result with validation details
 *
 * Security: Uses guardrails to prevent:
 * - Path traversal attacks (validateTargetPath)
 * - Unconfirmed mutations (requiresConfirmation)
 * - Invalid schema mutations (validateAfterWrite)
 *
 * @param pi ExtensionAPI instance
 */
export default function (pi: ExtensionAPI) {
  const run = createRunner(pi);

  pi.registerTool({
    name: "rootline-migrate",
    description:
      "Perform schema migration operations: rename fields, split flat .stem to hierarchical, or scaffold missing fields. " +
      "Defaults to dry_run=true for safety. Exposes diff/plan before mutation and validates affected roadmap after. " +
      "Requires explicit confirmation in non-interactive mode for non-dry-run operations.",
    input_schema: {
      type: "object" as const,
      properties: {
        path: {
          type: "string",
          description: "Directory to migrate (required)",
        },
        dry_run: {
          type: "boolean",
          description:
            "Preview changes without writing (optional, default: true for safety)",
        },
        confirmed: {
          type: "boolean",
          description:
            "Explicit confirmation for mutations in non-interactive mode (optional)",
        },
        interactive: {
          type: "boolean",
          description:
            "Whether Pi is running interactively (optional, default: false)",
        },
        rename: {
          type: "string",
          description:
            'Field rename in format "old_field=new_field" (mutually exclusive with split, scaffold)',
        },
        split: {
          type: "boolean",
          description:
            "Split flat .stem to hierarchical per-level files (mutually exclusive with rename, scaffold)",
        },
        scaffold: {
          type: "boolean",
          description:
            "Scaffold missing required section fields (mutually exclusive with rename, split)",
        },
      },
      required: ["path"],
    },
    execute: async (input: unknown, ctx) => {
      const {
        path: targetPath,
        dry_run = true, // Default to safe mode
        confirmed = false,
        interactive = false,
        rename,
        split = false,
        scaffold = false,
      } = input as MigrateToolInput;

      try {
        // Step 1: Validate target path
        const pathValidation = validateTargetPath(targetPath, ctx.cwd);
        if (!pathValidation.ok) {
          return {
            dry_run,
            path: targetPath,
            changes_detected: false,
            error: `Path validation failed: ${pathValidation.reason}`,
          } as RootlineMigrateResult;
        }

        // Step 2: Enforce mutual exclusion of operations
        const operationCount = [rename ? 1 : 0, split ? 1 : 0, scaffold ? 1 : 0].reduce((a, b) => a + b, 0);
        if (operationCount > 1) {
          return {
            dry_run,
            path: targetPath,
            changes_detected: false,
            error: "Exactly one of --rename, --split, or --scaffold must be specified (mutually exclusive)",
          } as RootlineMigrateResult;
        }

        if (operationCount === 0) {
          return {
            dry_run,
            path: targetPath,
            changes_detected: false,
            error: "One of --rename, --split, or --scaffold must be specified",
          } as RootlineMigrateResult;
        }

        // Step 3: Parse and validate rename format if provided
        let operation = "migrate";
        if (rename) {
          const parts = rename.split("=");
          if (parts.length !== 2 || !parts[0] || !parts[1]) {
            return {
              dry_run,
              path: targetPath,
              changes_detected: false,
              error: 'Invalid --rename format; expected "old_field=new_field"',
            } as RootlineMigrateResult;
          }
          operation = "rename";
        } else if (split) {
          operation = "split";
        } else if (scaffold) {
          operation = "scaffold";
        }

        // Step 4: Check if confirmation is required (for non-dry-run mutations)
        if (!dry_run && requiresConfirmation("update", interactive)) {
          if (!confirmed) {
            return {
              dry_run: true, // Report as dry-run since we're not executing
              path: targetPath,
              changes_detected: false,
              error:
                "Confirmation required: set confirmed: true to proceed with this migration in non-interactive mode",
            } as RootlineMigrateResult;
          }
        }

        // Step 5: Resolve path to absolute
        const resolvedPath = path.resolve(ctx.cwd, targetPath);

        // Step 6: Build args for rootline migrate command
        const args = ["migrate", resolvedPath, "--output", "json"];

        if (dry_run) {
          args.push("--dry-run");
        }

        if (rename) {
          args.push("--rename", rename);
        }

        if (split) {
          args.push("--split");
        }

        if (scaffold) {
          args.push("--scaffold");
        }

        // Step 7: Execute rootline migrate command
        const migrateResult = await run<{
          version: number;
          kind: string;
          path: string;
          operation?: string;
          dry_run: boolean;
          results?: unknown[];
          summary?: Record<string, unknown>;
          breaking_changes_detected?: boolean;
        }>(args, { cwd: ctx.cwd, timeout: 30_000 });

        if (!migrateResult.ok) {
          return {
            dry_run,
            path: targetPath,
            changes_detected: false,
            error: migrateResult.error.message,
          } as RootlineMigrateResult;
        }

        const migrateData = migrateResult.data;

        // Step 8: Check if changes were detected
        const changesDetected =
          migrateData.summary && Object.values(migrateData.summary).some(v => v && v !== 0 && v !== false);

        // Step 9: For non-dry-run with actual changes, validate affected scope
        let validation: ValidationGuardrailResult | undefined;
        if (!dry_run && changesDetected) {
          validation = await validateAfterWrite(
            resolvedPath,
            run,
            ctx.cwd
          );

          // Fail if validation returned errors
          if (!validation.ok) {
            return {
              dry_run: false,
              path: targetPath,
              operation,
              changes_detected: changesDetected,
              results: migrateData.results,
              summary: migrateData.summary,
              validation,
              error: "Post-migration validation failed",
            } as RootlineMigrateResult;
          }
        }

        // Step 10: Build response
        return {
          dry_run,
          path: targetPath,
          operation,
          changes_detected: changesDetected,
          results: migrateData.results,
          summary: migrateData.summary,
          validation,
        } as RootlineMigrateResult;
      } catch (err) {
        const msg = err instanceof Error ? err.message : String(err);
        return {
          dry_run: true,
          path: targetPath,
          changes_detected: false,
          error: `Unexpected error: ${msg}`,
        } as RootlineMigrateResult;
      }
    },
  });
}
