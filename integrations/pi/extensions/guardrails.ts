import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { createRunner, type RunnerResult } from "../runner";
import * as path from "node:path";

/**
 * Result of path validation guardrail.
 */
export type GuardrailResult = { ok: true } | { ok: false; reason: string };

/**
 * Result of post-mutation validation guardrail.
 */
export type ValidationGuardrailResult =
  | { ok: true; summary: Record<string, unknown> }
  | { ok: false; errors: unknown[] };

/**
 * Mutation operation type.
 */
export type MutationOperation = "create" | "update" | "delete";

/**
 * Function type for executing rootline commands.
 */
type RunnerFn = <T>(
  args: string[],
  opts?: { cwd?: string; timeout?: number }
) => Promise<RunnerResult<T>>;

/**
 * Validates that the target path is within the cwd and rejects absolute paths
 * outside the project root.
 *
 * Security guardrail: prevents path traversal attacks like ../../../etc/passwd
 *
 * @param targetPath The path to write to
 * @param cwd The current working directory (project root)
 * @returns GuardrailResult indicating if path is safe
 */
export function validateTargetPath(
  targetPath: string,
  cwd: string
): GuardrailResult {
  // Reject absolute paths that are outside cwd
  if (path.isAbsolute(targetPath)) {
    const resolved = path.resolve(targetPath);
    const resolvedCwd = path.resolve(cwd);
    if (!resolved.startsWith(resolvedCwd)) {
      return {
        ok: false,
        reason: `Path ${targetPath} is outside project root ${cwd}`,
      };
    }
    return { ok: true };
  }

  // Resolve relative path against cwd
  const resolvedPath = path.resolve(cwd, targetPath);
  const resolvedCwd = path.resolve(cwd);

  // Check if resolved path is within cwd
  if (!resolvedPath.startsWith(resolvedCwd)) {
    return {
      ok: false,
      reason: `Path ${targetPath} escapes project root via ../ traversal`,
    };
  }

  return { ok: true };
}

/**
 * Determines if a mutation operation requires user confirmation.
 *
 * Rules:
 * - In non-interactive mode: all mutations require explicit confirmation (--force or confirmed: true)
 * - In interactive mode: confirmation is handled by Pi agent asking the user
 *
 * @param operation The mutation operation (create, update, delete)
 * @param isInteractive Whether Pi is in interactive mode
 * @returns true if confirmation is required before executing the mutation
 */
export function requiresConfirmation(
  operation: MutationOperation,
  isInteractive: boolean
): boolean {
  // Non-interactive mode: all mutations require confirmation
  if (!isInteractive) {
    return true;
  }

  // Interactive mode: Pi agent handles user interaction
  // This function would return true if we want Pi to always ask,
  // or false if we allow auto-execution in interactive mode
  // Current implementation: in interactive mode, return false because Pi handles confirmation
  return false;
}

/**
 * Validates a written file by running `rootline validate` on it.
 *
 * Post-mutation validation ensures that writes don't produce invalid documents.
 *
 * @param filePath Path to the file that was written
 * @param run Runner function to execute rootline commands
 * @param cwd Current working directory for resolving relative paths
 * @returns ValidationGuardrailResult with validation summary or errors
 */
export async function validateAfterWrite(
  filePath: string,
  run: RunnerFn,
  cwd: string = process.cwd()
): Promise<ValidationGuardrailResult> {
  // Determine the directory to validate (parent of the file)
  const fileAbsolute = path.resolve(cwd, filePath);
  const validateDir = path.dirname(fileAbsolute);

  // Execute: rootline validate <dir> --output json
  const result = await run<{
    version: number;
    kind: string;
    results: Array<{ path: string; valid: boolean; errors: unknown[] }>;
    summary: Record<string, unknown>;
  }>(["validate", validateDir, "--output", "json"], { cwd });

  if (!result.ok) {
    return {
      ok: false,
      errors: [
        {
          message: result.error.message,
          code: result.error.code,
          stderr: result.error.stderr,
        },
      ],
    };
  }

  // Check if validation passed
  const { results, summary } = result.data;

  // Find validation result for the target file
  const fileResult = results.find((r) => r.path.includes(path.basename(filePath)));

  if (fileResult && !fileResult.valid) {
    return {
      ok: false,
      errors: fileResult.errors || [
        { message: `Validation failed for ${filePath}` },
      ],
    };
  }

  // All validations passed
  return {
    ok: true,
    summary,
  };
}

/**
 * Guardrails module for safe mutation operations.
 *
 * Exports:
 * - validateTargetPath: Check path safety (no ../escape)
 * - requiresConfirmation: Determine if confirmation is needed
 * - validateAfterWrite: Run post-mutation validation
 *
 * Used by rootline-new and rootline-set tools to enforce safety constraints.
 *
 * @param pi ExtensionAPI instance
 */
export default function (pi: ExtensionAPI) {
  // This function is called for registration if used as an extension
  // The actual guardrails are exported as functions for use by mutation tools
}
