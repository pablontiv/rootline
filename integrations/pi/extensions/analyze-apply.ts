import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { createRunner, type RunnerResult } from "../runner";
import {
  validateTargetPath,
  requiresConfirmation,
  type ValidationGuardrailResult,
} from "./guardrails";
import * as path from "node:path";

/**
 * Analyze report result from rootline.
 */
interface AnalyzeReport {
  version: number;
  kind: string;
  path: string;
  incremental?: boolean;
  categories?: Array<{
    id: string;
    name: string;
    inference_count: number;
    inferences: Array<{
      type: string;
      source?: string;
      field?: string;
      message: string;
      requires_agent: boolean;
    }>;
  }>;
  summary?: Record<string, unknown>;
}

/**
 * Schema apply result from rootline.
 */
interface SchemaApplyResult {
  version: number;
  kind: string;
  dry_run: boolean;
  path?: string;
  results?: Array<{
    proposal_type: string;
    source: string;
    field?: string;
    action: string;
    status: string;
    affected_stem_file?: string;
  }>;
  summary?: {
    total_proposals_in_report: number;
    applied_count: number;
    skipped_count: number;
    requires_agent_count: number;
  };
  validation?: {
    valid: boolean;
    schema_conflicts?: number;
    stem_health_warnings?: string[];
  };
}

/**
 * Repair apply result from rootline.
 */
interface RepairApplyResult {
  version: number;
  kind: string;
  dry_run: boolean;
  report_source?: string;
  path?: string;
  results?: Array<{
    path: string;
    status: string;
    changes?: Array<{
      field: string;
      old_value?: unknown;
      new_value?: unknown;
      reason: string;
    }>;
  }>;
  summary?: {
    total_files: number;
    would_repair?: number;
    skipped?: number;
    total_fields_changed: number;
    fields_added?: number;
    values_corrected?: number;
  };
  schema_surface_count?: number;
  repair_surface_count?: number;
  validation_summary?: {
    valid: boolean;
    total_files: number;
    error_count: number;
    warning_count: number;
  };
}

/**
 * Validate result from rootline.
 */
interface ValidateResult {
  version: number;
  kind: string;
  results: Array<{
    path: string;
    valid: boolean;
    errors: unknown[];
  }>;
  summary: {
    total: number;
    valid: number;
    invalid: number;
  };
}

/**
 * Tool input for rootline-analyze.
 */
interface AnalyzeToolInput {
  path: string;
  incremental?: boolean;
  threshold?: number;
}

/**
 * Tool input for rootline-apply.
 */
interface ApplyToolInput {
  report_path: string;
  mode: "schema" | "repair" | "both";
  dry_run?: boolean;
  confirmed?: boolean;
  interactive?: boolean;
}

/**
 * Result of analyze operation.
 */
interface RootlineAnalyzeResult {
  analyzed: boolean;
  path: string;
  report?: AnalyzeReport;
  error?: string;
}

/**
 * Result of apply operation.
 */
interface RootlineApplyResult {
  applied: boolean;
  dry_run: boolean;
  mode: string;
  schema_result?: SchemaApplyResult;
  repair_result?: RepairApplyResult;
  validation?: ValidationGuardrailResult;
  error?: string;
}

/**
 * Registers the rootline-analyze tool for analyzing documents and inferring schema patterns.
 *
 * This tool:
 * 1. Validates the target path
 * 2. Calls `rootline analyze` with appropriate flags
 * 3. Returns the full analyze report for inspection
 *
 * No confirmation required (read-only operation).
 *
 * @param pi ExtensionAPI instance
 */
function registerAnalyzeTool(pi: ExtensionAPI) {
  const run = createRunner(pi);

  pi.registerTool({
    name: "rootline-analyze",
    description:
      "Analyze documents and infer schema patterns. Returns a structured analyze report with inferred fields, enums, coverage gaps, and governance findings. Read-only operation, no confirmation required.",
    input_schema: {
      type: "object" as const,
      properties: {
        path: {
          type: "string",
          description: "Directory to analyze (required)",
        },
        incremental: {
          type: "boolean",
          description:
            "Report only inferences not covered by existing .stem files (optional, default: false)",
        },
        threshold: {
          type: "number",
          description:
            "Section pattern detection threshold (0.0-1.0, optional, default: 0.6)",
        },
      },
      required: ["path"],
    },
    execute: async (input: unknown) => {
      const {
        path: targetPath,
        incremental = false,
        threshold,
      } = input as AnalyzeToolInput;

      // Step 1: Validate target path
      const pathValidation = validateTargetPath(targetPath, process.cwd());
      if (!pathValidation.ok) {
        return {
          analyzed: false,
          path: targetPath,
          error: `Path validation failed: ${pathValidation.reason}`,
        } as RootlineAnalyzeResult;
      }

      // Step 2: Resolve path to absolute
      const resolvedPath = path.resolve(process.cwd(), targetPath);

      // Step 3: Build args for rootline analyze command
      const args = ["analyze", resolvedPath, "--output", "json"];

      if (incremental) {
        args.push("--incremental");
      }

      if (threshold !== undefined) {
        args.push("--threshold", String(threshold));
      }

      // Step 4: Execute rootline analyze command
      const result = await run<AnalyzeReport>(args, {
        cwd: process.cwd(),
        timeout: 60_000, // 60s timeout for analysis
      });

      if (!result.ok) {
        return {
          analyzed: false,
          path: targetPath,
          error: `Failed to analyze: ${result.error.message}`,
        } as RootlineAnalyzeResult;
      }

      // Step 5: Return the report
      return {
        analyzed: true,
        path: resolvedPath,
        report: result.data,
      } as RootlineAnalyzeResult;
    },
  });
}

/**
 * Registers the rootline-apply tool for applying schema and repair proposals.
 *
 * This tool:
 * 1. Validates the report path
 * 2. Checks if confirmation is required (when not dry-run)
 * 3. Calls `rootline schema apply` and/or `rootline repair apply` as specified
 * 4. Runs post-apply validation
 * 5. Returns structured result with both apply results and validation state
 *
 * Confirmation required in non-interactive mode when not in dry-run.
 *
 * @param pi ExtensionAPI instance
 */
function registerApplyTool(pi: ExtensionAPI) {
  const run = createRunner(pi);

  pi.registerTool({
    name: "rootline-apply",
    description:
      "Apply schema and/or repair proposals from an analyze report. Supports dry-run preview and explicit confirmation. Returns structured result with applied proposals and post-apply validation.",
    input_schema: {
      type: "object" as const,
      properties: {
        report_path: {
          type: "string",
          description: "Path to the analyze report JSON file (required)",
        },
        mode: {
          type: "string",
          description:
            "Which apply to run: 'schema' | 'repair' | 'both' (required)",
        },
        dry_run: {
          type: "boolean",
          description:
            "Preview changes without writing (optional, default: true)",
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
      },
      required: ["report_path", "mode"],
    },
    execute: async (input: unknown) => {
      const {
        report_path: reportPath,
        mode,
        dry_run = true,
        confirmed = false,
        interactive = false,
      } = input as ApplyToolInput;

      // Step 1: Validate report path
      const pathValidation = validateTargetPath(reportPath, process.cwd());
      if (!pathValidation.ok) {
        return {
          applied: false,
          dry_run,
          mode,
          error: `Path validation failed: ${pathValidation.reason}`,
        } as RootlineApplyResult;
      }

      // Step 2: Validate mode
      if (!["schema", "repair", "both"].includes(mode)) {
        return {
          applied: false,
          dry_run,
          mode,
          error: `Invalid mode: ${mode}. Must be 'schema', 'repair', or 'both'`,
        } as RootlineApplyResult;
      }

      // Step 3: Check if confirmation is required (only for non-dry-run)
      if (!dry_run && requiresConfirmation("update", interactive) && !confirmed) {
        return {
          applied: false,
          dry_run,
          mode,
          error:
            "Confirmation required: set confirmed: true to apply changes in non-interactive mode",
        } as RootlineApplyResult;
      }

      // Step 4: Resolve report path to absolute
      const resolvedReportPath = path.resolve(process.cwd(), reportPath);

      const result: RootlineApplyResult = {
        applied: false,
        dry_run,
        mode,
      };

      // Step 5: Apply schema proposals if requested
      if (mode === "schema" || mode === "both") {
        const schemaArgs = ["schema", "apply", "--report", resolvedReportPath, "--output", "json"];

        if (dry_run) {
          schemaArgs.push("--dry-run");
        }

        const schemaResult = await run<SchemaApplyResult>(schemaArgs, {
          cwd: process.cwd(),
          timeout: 30_000, // 30s timeout per apply command
        });

        if (!schemaResult.ok) {
          return {
            ...result,
            error: `Schema apply failed: ${schemaResult.error.message}`,
          };
        }

        result.schema_result = schemaResult.data;
      }

      // Step 6: Apply repair proposals if requested
      if (mode === "repair" || mode === "both") {
        const repairArgs = [
          "repair",
          "apply",
          "--report",
          resolvedReportPath,
          "--output",
          "json",
        ];

        if (dry_run) {
          repairArgs.push("--dry-run");
        }

        const repairResult = await run<RepairApplyResult>(repairArgs, {
          cwd: process.cwd(),
          timeout: 30_000, // 30s timeout per apply command
        });

        if (!repairResult.ok) {
          return {
            ...result,
            error: `Repair apply failed: ${repairResult.error.message}`,
          };
        }

        result.repair_result = repairResult.data;
      }

      // Step 7: Run post-apply validation (only if not dry-run)
      if (!dry_run) {
        // Validate all files in the project after applying changes
        const validateResult = await run<ValidateResult>(
          ["validate", "--all", "--output", "json"],
          {
            cwd: process.cwd(),
            timeout: 30_000,
          }
        );

        if (validateResult.ok) {
          result.validation = {
            ok: true,
            summary: validateResult.data.summary,
          };
        } else {
          result.validation = {
            ok: false,
            errors: [
              {
                message: validateResult.error.message,
                code: validateResult.error.code,
              },
            ],
          };
        }
      }

      // Step 8: Return success
      result.applied = true;
      return result;
    },
  });
}

/**
 * Registers both rootline-analyze and rootline-apply tools.
 *
 * @param pi ExtensionAPI instance
 */
export default function (pi: ExtensionAPI) {
  registerAnalyzeTool(pi);
  registerApplyTool(pi);
}
