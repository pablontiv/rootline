import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { createRunner, type RunnerResult } from "../runner";

/**
 * Tool interface for rootline-validate.
 *
 * Calls `rootline validate --all <path> --output json [--where <expr>]`.
 */
interface ValidateToolInput {
  /** Directory path to validate (required) */
  path: string;
  /** Validate all files in scope (default: true) */
  all?: boolean;
  /** Filter expression for records (e.g. "estado == 'Pending'") */
  where?: string;
}

/**
 * JSON output structure for validation (matches BatchValidationResult from Go).
 */
interface ValidationResult {
  version: number;
  kind: string;
  results: Array<{
    version: number;
    kind: string;
    path: string;
    valid: boolean;
    errors: Array<{
      path: string;
      field: string;
      rule: string;
      message: string;
      severity: string;
    }>;
    warnings: Array<{
      path: string;
      field: string;
      rule: string;
      message: string;
      severity: string;
    }>;
  }>;
  drift_warnings: unknown[];
  summary: {
    total: number;
    valid: number;
    invalid: number;
    errors_count: number;
    warnings_count: number;
    drift_warnings_count: number;
  };
}

/**
 * Registers the rootline-validate tool.
 * @param pi ExtensionAPI instance
 */
export default function (pi: ExtensionAPI) {
  const run = createRunner(pi);

  pi.registerTool({
    name: "rootline-validate",
    description:
      "Validate documents in a directory against their .stem schema rules. " +
      "Returns validation results with errors and warnings preserved for model inspection.",
    input_schema: {
      type: "object" as const,
      properties: {
        path: {
          type: "string",
          description: "Directory path to validate",
        },
        all: {
          type: "boolean",
          description: "Validate all files in scope (default: true)",
        },
        where: {
          type: "string",
          description:
            "Filter expression for records (e.g. 'estado == \"Pending\"')",
        },
      },
      required: ["path"],
    },
    execute: async (input: unknown) => {
      const { path, all = true, where } = input as ValidateToolInput;

      // Build command args
      const args = ["validate"];
      if (all) {
        args.push("--all");
      }
      args.push(path);
      args.push("--output", "json");

      if (where) {
        args.push("--where", where);
      }

      // Execute rootline command
      const result = await run<ValidationResult>(args);

      if (!result.ok) {
        return {
          error: result.error.message,
          details: result.error.stderr || undefined,
          code: result.error.code,
        };
      }

      // Return validation results with summary
      return {
        valid: result.data.summary.invalid === 0,
        summary: result.data.summary,
        results: result.data.results,
        drift_warnings: result.data.drift_warnings,
      };
    },
  });
}
