import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

/**
 * Options for rootline runner execution.
 */
export interface RunnerOptions {
  /** Working directory for the rootline process */
  cwd?: string;
  /** Timeout in milliseconds (default: 10_000) */
  timeout?: number;
  /** Abort signal for cancellation */
  signal?: AbortSignal;
}

/**
 * Structured error type returned by runner on failure.
 */
export interface RootlineError {
  /** Error code indicating failure category */
  code: "exec_failed" | "json_parse_failed" | "timeout" | "not_found";
  /** Human-readable error message */
  message: string;
  /** Exit code (if exec_failed) */
  exitCode?: number;
  /** stderr output from rootline (if available) */
  stderr?: string;
}

/**
 * Result type: discriminated union of success or failure.
 */
export type RunnerResult<T> =
  | { ok: true; data: T }
  | { ok: false; error: RootlineError };

/**
 * Creates a rootline CLI runner that executes rootline commands via pi.exec.
 *
 * The runner:
 * - Executes rootline with specified arguments
 * - Parses JSON output (--output json is recommended)
 * - Handles timeouts, non-zero exit codes, and invalid JSON
 * - Returns structured errors with exit codes and stderr
 *
 * @param pi ExtensionAPI instance from Pi extension context
 * @returns Async function to execute rootline commands
 *
 * @example
 * const run = createRunner(pi);
 * const result = await run(["query", "--output", "json"], { cwd: "/path/to/repo" });
 * if (result.ok) {
 *   console.log(result.data);
 * } else {
 *   console.error(result.error.message);
 * }
 */
export function createRunner(pi: ExtensionAPI) {
  return async function run<T = unknown>(
    args: string[],
    opts: RunnerOptions = {}
  ): Promise<RunnerResult<T>> {
    const { cwd, timeout = 10_000, signal } = opts;

    try {
      // Check for abort signal before execution
      if (signal?.aborted) {
        return {
          ok: false,
          error: {
            code: "exec_failed",
            message: "Operation aborted before execution",
          },
        };
      }

      const { stdout, stderr, code } = await pi.exec("rootline", args, {
        cwd,
        timeout,
      });

      // Handle non-zero exit code
      if (code !== 0) {
        return {
          ok: false,
          error: {
            code: "exec_failed",
            message: stderr || "rootline command failed",
            exitCode: code,
            stderr: stderr || undefined,
          },
        };
      }

      // Parse JSON output
      try {
        const data = JSON.parse(stdout) as T;
        return { ok: true, data };
      } catch (_err) {
        return {
          ok: false,
          error: {
            code: "json_parse_failed",
            message: "rootline output is not valid JSON",
            stderr: stdout,
          },
        };
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);

      // Detect timeout errors
      if (
        msg.includes("timeout") ||
        msg.includes("ETIMEDOUT") ||
        msg.includes("Timeout")
      ) {
        return {
          ok: false,
          error: {
            code: "timeout",
            message: `rootline command timed out after ${timeout}ms`,
          },
        };
      }

      // Detect not found errors
      if (msg.includes("ENOENT") || msg.includes("not found")) {
        return {
          ok: false,
          error: {
            code: "not_found",
            message: "rootline binary not found",
          },
        };
      }

      // Generic execution error
      return {
        ok: false,
        error: {
          code: "exec_failed",
          message: msg,
        },
      };
    }
  };
}
