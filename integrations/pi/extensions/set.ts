import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { createRunner, type RunnerResult } from "../runner.js";
import {
  validateTargetPath,
  requiresConfirmation,
  validateAfterWrite,
} from "./guardrails.js";
import * as path from "node:path";

/**
 * Parameters for the rootline-set tool.
 *
 * Tool allows updating frontmatter fields or document sections with guardrails
 * for path safety, confirmation requirements, and post-mutation validation.
 */
interface SetToolParams {
  /** Target file to update (required) */
  path: string;
  /** Array of field assignments: { field: "fieldname", value: "value" } */
  fields: Array<{ field: string; value: string }>;
  /** Preview changes without writing (default: false) */
  dry_run?: boolean;
  /** User explicitly confirms the mutation (required if not interactive) */
  confirmed?: boolean;
  /** Whether Pi is running interactively (default: false) */
  interactive?: boolean;
}

/**
 * Result structure for set operations.
 */
interface SetResult {
  updated: boolean;
  path: string;
  fields_set?: string[];
  dry_run_preview?: string[];
  validation?: {
    ok: boolean;
    summary?: Record<string, unknown>;
    errors?: unknown[];
  };
  error?: string;
}

/**
 * Register the rootline-set Pi tool.
 *
 * This tool updates frontmatter fields in a document with guardrails:
 * 1. Path validation: ensures target is within project
 * 2. Confirmation: requires explicit confirmation in non-interactive mode
 * 3. Post-validation: validates the file after mutation
 *
 * @param pi - Pi ExtensionAPI instance
 */
export default function (pi: ExtensionAPI) {
  const run = createRunner(pi);

  pi.registerTool({
    name: "rootline-set",
    description:
      "Update frontmatter fields in a Markdown file with path safety guardrails " +
      "and post-mutation validation. Requires confirmation in non-interactive mode.",
    input_schema: {
      type: "object" as const,
      properties: {
        path: {
          type: "string",
          description: "Path to the file to update",
        },
        fields: {
          type: "array",
          description: "Array of field assignments to apply",
          items: {
            type: "object",
            properties: {
              field: {
                type: "string",
                description: "Field name to set",
              },
              value: {
                type: "string",
                description: "Value to assign",
              },
            },
            required: ["field", "value"],
          } as any,
        },
        dry_run: {
          type: "boolean",
          description: "Preview changes without writing (default: false)",
        },
        confirmed: {
          type: "boolean",
          description:
            "Explicitly confirm the mutation (required if not interactive)",
        },
        interactive: {
          type: "boolean",
          description: "Whether Pi is running interactively (default: false)",
        },
      },
      required: ["path", "fields"],
    },
    execute: async (input: unknown, ctx): Promise<SetResult> => {
      const {
        path: targetPath,
        fields,
        dry_run = false,
        confirmed = false,
        interactive = false,
      } = input as SetToolParams;

      try {
        // Step 1: Validate target path safety (no path traversal)
        const pathCheck = validateTargetPath(targetPath, ctx.cwd);
        if (!pathCheck.ok) {
          return {
            updated: false,
            path: targetPath,
            error: pathCheck.reason,
          };
        }

        // Step 2: Check if confirmation is required
        if (requiresConfirmation("update", interactive)) {
          if (!confirmed) {
            return {
              updated: false,
              path: targetPath,
              error:
                "Confirmation required: set confirmed=true to proceed with this mutation in non-interactive mode",
            };
          }
        }

        // Step 3: Resolve the full path
        const resolvedPath = path.resolve(ctx.cwd, targetPath);

        // Step 4: Build rootline set command arguments
        const args = ["set", resolvedPath];

        // Add field=value arguments
        const fieldStrs: string[] = [];
        for (const f of fields) {
          const fieldArg = `${f.field}=${f.value}`;
          args.push(fieldArg);
          fieldStrs.push(fieldArg);
        }

        // Add --dry-run flag if requested
        if (dry_run) {
          args.push("--dry-run");
        }

        // Step 5: Execute rootline set command
        // Note: rootline set does not support --output json, so we handle text output
        let setResult: { ok: boolean; stdout?: string; stderr?: string; code?: number; error?: any };
        try {
          // Execute the set command without JSON parsing expectation
          const execResult = await pi.exec("rootline", args, { cwd: ctx.cwd, timeout: 10_000 });

          if (execResult.code !== 0) {
            setResult = {
              ok: false,
              stderr: execResult.stderr,
              code: execResult.code,
            };
          } else {
            setResult = {
              ok: true,
              stdout: execResult.stdout,
            };
          }
        } catch (err) {
          const msg = err instanceof Error ? err.message : String(err);
          return {
            updated: false,
            path: targetPath,
            error: `Failed to execute rootline set: ${msg}`,
          };
        }

        // Step 6: Handle execution failure
        if (!setResult.ok) {
          return {
            updated: false,
            path: targetPath,
            error: setResult.stderr || "rootline set command failed",
          };
        }

        // Step 7: If dry-run, return preview
        if (dry_run) {
          const preview = (setResult.stdout || "")
            .split("\n")
            .filter((line) => line.trim());
          return {
            updated: false,
            path: targetPath,
            dry_run_preview: preview,
          };
        }

        // Step 8: After successful write, validate the file
        const validationResult = await validateAfterWrite(
          resolvedPath,
          run,
          ctx.cwd
        );

        // Step 9: Build response
        if (validationResult.ok) {
          return {
            updated: true,
            path: targetPath,
            fields_set: fieldStrs,
            validation: {
              ok: true,
              summary: validationResult.summary,
            },
          };
        } else {
          return {
            updated: false,
            path: targetPath,
            error: "Post-mutation validation failed",
            validation: {
              ok: false,
              errors: validationResult.errors,
            },
          };
        }
      } catch (err) {
        const msg = err instanceof Error ? err.message : String(err);
        return {
          updated: false,
          path: targetPath,
          error: `Unexpected error: ${msg}`,
        };
      }
    },
  });
}
