import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { createRunner, type RootlineError } from "../runner";

/**
 * Rootline project state for status widget.
 */
type ProjectState = "no_rootline" | "binary_only" | "stem_governed" | "error";

/**
 * Tool input for status check.
 */
interface StatusToolInput {
  /** Directory path to check (optional, defaults to cwd) */
  path?: string;
}

/**
 * Validation summary from rootline validate output.
 */
interface ValidationSummary {
  total?: number;
  valid?: number;
  invalid?: number;
  errors_count?: number;
  warnings_count?: number;
}

/**
 * Validation result from rootline validate.
 */
interface ValidateResult {
  summary?: ValidationSummary;
}

/**
 * Describe result from rootline.
 */
interface DescribeResult {
  applies?: string[];
}

/**
 * Status check result returned by the widget.
 */
interface StatusCheckResult {
  /** Project state: no_rootline, binary_only, stem_governed, or error */
  state: ProjectState;
  /** Human-readable single-line status */
  status_line: string;
  /** Version string if binary is available */
  version?: string;
  /** Number of .stem files if directory is governed */
  stem_count?: number;
  /** Number of valid documents */
  valid?: number;
  /** Number of validation errors */
  errors?: number;
  /** Number of validation warnings */
  warnings?: number;
}

/**
 * Registers the rootline-status tool for project health status widget.
 *
 * Fast health check for display in UI status bars or health dashboards.
 * Combines lightweight state detection with validation summary for a
 * concise, single-line status representation.
 *
 * States:
 * - no_rootline: binary not found in PATH
 * - binary_only: binary exists but no .stem files in target directory
 * - stem_governed: binary + .stem files present, with validation summary
 * - error: unexpected failure during check
 *
 * Timeout: 5s (safe for periodic polling in UI).
 *
 * Returns:
 * - state: one of the above
 * - status_line: human-readable one-liner suitable for status display
 * - version, stem_count, valid, errors, warnings: metadata for rich display
 *
 * @param pi ExtensionAPI instance
 */
export default function (pi: ExtensionAPI) {
  const run = createRunner(pi);

  pi.registerTool({
    name: "rootline-status",
    description:
      "Get Rootline project health status for display in status widgets or dashboards. " +
      "Returns state (no_rootline, binary_only, stem_governed, or error), " +
      "a human-readable status line, version, stem count, and validation summary. " +
      "Fast operation (<3s typical, 5s timeout).",
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
      const { path } = input as StatusToolInput;
      const checkPath = path || process.cwd();

      // Initialize result
      const result: StatusCheckResult = {
        state: "no_rootline",
        status_line: "",
      };

      // Step 1: Check if rootline binary is available
      const versionResult = await run<string>(["--version"], {
        timeout: 5000,
      });

      if (!versionResult.ok) {
        // Binary not found
        result.state = "no_rootline";
        result.status_line = "✗ Rootline: binary not found";
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
        result.status_line = "⚠ Rootline: no schema found (.stem files missing)";
        return result;
      }

      // Directory is governed, count .stem files
      const stemFiles = describeResult.data.applies || [];
      result.stem_count = stemFiles.length;

      // Step 3: Run validation to get summary
      const validateArgs = ["validate", "--all", checkPath, "--output", "json"];
      const validateResult = await run<ValidateResult>(validateArgs, {
        cwd: checkPath,
        timeout: 5000,
      });

      // Extract validation summary
      let validCount = 0;
      let errorCount = 0;
      let warningCount = 0;

      if (validateResult.ok && validateResult.data.summary) {
        const summary = validateResult.data.summary;
        validCount = summary.valid ?? 0;
        errorCount = summary.errors_count ?? 0;
        warningCount = summary.warnings_count ?? 0;
      }

      result.valid = validCount;
      result.errors = errorCount;
      result.warnings = warningCount;

      // Build status line based on validation results
      if (errorCount > 0) {
        result.state = "stem_governed";
        const warnPart = warningCount > 0 ? `, ${warningCount} warn` : "";
        result.status_line =
          `✓ Rootline: governed | ${result.stem_count} stems | ` +
          `${validCount} valid, ${errorCount} errors${warnPart}`;
      } else {
        result.state = "stem_governed";
        const warnPart = warningCount > 0 ? ` | ${warningCount} warn` : "";
        result.status_line =
          `✓ Rootline: governed | ${result.stem_count} stems | ` +
          `${validCount} valid${warnPart}`;
      }

      return result;
    },
  });
}
