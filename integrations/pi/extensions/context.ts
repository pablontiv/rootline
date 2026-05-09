import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { createRunner, type RootlineError } from "../runner";

/**
 * Rootline project state detection results.
 */
type ProjectState = "no_rootline" | "binary_only" | "stem_governed";

/**
 * Tool input for context detection.
 */
interface ContextToolInput {
  /** Directory path to check (optional, defaults to cwd) */
  path?: string;
}

/**
 * Detected project context with state and version info.
 */
interface RootlineContext {
  /** Project state: no_rootline, binary_only, or stem_governed */
  state: ProjectState;
  /** Version string if binary is available */
  version?: string;
  /** Number of .stem files if directory is governed */
  stem_count?: number;
}

/**
 * Describe result from rootline (subset needed for context detection).
 */
interface DescribeResult {
  applies?: string[];
}

/**
 * Registers the rootline-context tool for lightweight project state detection.
 *
 * Used for runtime context injection (e.g., in before_agent_start hooks).
 * Distinguishes three states:
 * - no_rootline: binary not found in PATH
 * - binary_only: binary exists but no .stem files in target directory
 * - stem_governed: binary + .stem files present
 *
 * Fast operation: timeout 5s, no heavy directory scanning.
 *
 * @param pi ExtensionAPI instance
 */
export default function (pi: ExtensionAPI) {
  const run = createRunner(pi);

  pi.registerTool({
    name: "rootline-context",
    description:
      "Detect Rootline project state from current or specified directory. " +
      "Returns project state (no_rootline, binary_only, or stem_governed), " +
      "version if binary exists, and stem count if governed. " +
      "Lightweight and fast (<5s timeout).",
    input_schema: {
      type: "object" as const,
      properties: {
        path: {
          type: "string",
          description:
            "Directory path to check (optional, defaults to current working directory)",
        },
      },
      required: [],
    },
    execute: async (input: unknown) => {
      const { path } = input as ContextToolInput;
      const checkPath = path || process.cwd();

      // Initialize result
      const result: RootlineContext = {
        state: "no_rootline",
      };

      // Step 1: Check if rootline binary is available
      const versionResult = await run<string>(["--version"], {
        timeout: 5000,
      });

      if (!versionResult.ok) {
        // Binary not found
        if (
          versionResult.error.code === "not_found" ||
          versionResult.error.code === "exec_failed"
        ) {
          result.state = "no_rootline";
          return result;
        }
        // Other errors (timeout, etc.) also count as no_rootline
        result.state = "no_rootline";
        return result;
      }

      // Binary exists, extract version
      result.version = versionResult.data.trim();

      // Step 2: Check if directory is governed (has .stem files)
      const describeResult = await run<DescribeResult>(
        ["describe", checkPath, "--output", "json"],
        { cwd: checkPath, timeout: 5000 }
      );

      if (!describeResult.ok) {
        // Binary exists but no .stem files
        result.state = "binary_only";
        return result;
      }

      // Directory is governed, count .stem files
      const stemFiles = describeResult.data.applies || [];
      result.state = "stem_governed";
      result.stem_count = stemFiles.length;

      return result;
    },
  });
}
