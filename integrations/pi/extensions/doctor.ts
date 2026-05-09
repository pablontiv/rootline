import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { createRunner, type RunnerResult } from "../runner";

/**
 * Tool interface for rootline-doctor.
 *
 * Runs a health check on the rootline setup:
 * - Checks if rootline binary is in PATH and returns its version
 * - Checks if the given directory is governed (has .stem files)
 * - Reports what tools are available
 */
interface DoctorToolInput {
  /** Directory path to check (optional, defaults to cwd) */
  path?: string;
}

/**
 * JSON output structure for describe (matches rootline describe output).
 */
interface DescribeResult {
  version: number;
  kind: string;
  path: string;
  applies: string[];
  scope?: {
    match?: string;
  };
  schema?: Record<string, unknown>;
  validate?: unknown[];
  derive?: Record<string, unknown>;
  aggregate?: Record<string, unknown>;
  links?: unknown;
  layers?: string[];
  provenance?: Record<string, string>;
}

/**
 * Health check result returned by doctor.
 */
interface DoctorResult {
  /** Whether rootline binary is available */
  binary_available: boolean;
  /** Version string if binary is available */
  version?: string;
  /** Whether the directory is governed (has .stem files) */
  is_governed: boolean;
  /** Absolute path to .stem files if directory is governed */
  stem_files?: string[];
  /** Available tools in this directory */
  available_tools: string[];
  /** Diagnostic messages */
  diagnostics: string[];
}

/**
 * Registers the rootline-doctor tool.
 * @param pi ExtensionAPI instance
 */
export default function (pi: ExtensionAPI) {
  const run = createRunner(pi);

  pi.registerTool({
    name: "rootline-doctor",
    description:
      "Run a health check on the rootline setup. Checks if rootline binary is available, " +
      "if the directory is governed (has .stem files), and what tools are available. " +
      "Provides diagnostic information about the rootline environment.",
    input_schema: {
      type: "object" as const,
      properties: {
        path: {
          type: "string",
          description:
            "Directory path to check for governance (optional, defaults to current working directory)",
        },
      },
      required: [],
    },
    execute: async (input: unknown) => {
      const { path } = input as DoctorToolInput;
      const checkPath = path || process.cwd();
      const diagnostics: string[] = [];

      // Initialize result
      const result: DoctorResult = {
        binary_available: false,
        is_governed: false,
        available_tools: [],
        diagnostics: [],
      };

      // Step 1: Check if rootline binary is available
      const versionResult = await run<string>(["--version"]);
      if (versionResult.ok) {
        result.binary_available = true;
        // Extract version from output (format: "rootline version v0.9.100-115-g02db2b4")
        const versionStr = versionResult.data.trim();
        result.version = versionStr;
        diagnostics.push(`✓ rootline binary available: ${versionStr}`);
      } else {
        result.binary_available = false;
        diagnostics.push(
          `✗ rootline binary not found: ${versionResult.error.message}`
        );
      }

      // Step 2: Check if directory is governed
      if (result.binary_available) {
        const describeResult = await run<DescribeResult>(
          ["describe", checkPath, "--output", "json"],
          { cwd: checkPath }
        );

        if (describeResult.ok) {
          result.is_governed = true;
          result.stem_files = describeResult.data.applies || [];
          diagnostics.push(
            `✓ Directory is governed with ${result.stem_files.length} .stem file(s)`
          );

          // Step 3: Report available tools based on schema
          const schema = describeResult.data.schema;
          if (schema && typeof schema === "object") {
            const fields = Object.keys(schema);
            result.available_tools = fields;
            diagnostics.push(
              `✓ Available schema fields: ${fields.join(", ")}`
            );
          } else {
            diagnostics.push("✓ Schema available but no fields defined");
          }
        } else {
          result.is_governed = false;
          diagnostics.push(
            `✗ Directory is not governed: ${describeResult.error.message}`
          );
        }
      }

      result.diagnostics = diagnostics;

      return result;
    },
  });
}
