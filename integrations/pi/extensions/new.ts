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
 * Tool input for rootline-new.
 */
interface NewToolInput {
  /** Target file path to create (required) */
  path: string;
  /** Overwrite existing file (optional, default: false) */
  force?: boolean;
  /** Preview generated content without writing (optional, default: false) */
  dry_run?: boolean;
  /** Explicit confirmation for mutations in non-interactive mode (optional) */
  confirmed?: boolean;
  /** Whether Pi is running interactively (optional, default: false) */
  interactive?: boolean;
}

/**
 * Result of file creation.
 */
interface NewResult {
  version: number;
  kind: string;
  path?: string;
  content_preview?: string;
}

/**
 * Structured result returned by rootline-new tool.
 */
interface RootlineNewResult {
  created: boolean;
  path: string;
  validation?: ValidationGuardrailResult;
  content_preview?: string;
  error?: string;
}

/**
 * Registers the rootline-new tool for creating governed records.
 *
 * This tool:
 * 1. Validates the target path (prevents path traversal)
 * 2. Checks if confirmation is required
 * 3. Calls `rootline new` with appropriate flags
 * 4. Validates the created file
 * 5. Returns structured result with validation details
 *
 * Security: Uses guardrails to prevent:
 * - Path traversal attacks (validateTargetPath)
 * - Unconfirmed mutations (requiresConfirmation)
 * - Invalid document creation (validateAfterWrite)
 *
 * @param pi ExtensionAPI instance
 */
export default function (pi: ExtensionAPI) {
  const run = createRunner(pi);

  pi.registerTool({
    name: "rootline-new",
    description:
      "Create a new governed record file with frontmatter scaffolded from effective .stem schema. " +
      "Returns validation details and content preview. Enforces path safety and requires explicit confirmation in non-interactive mode.",
    input_schema: {
      type: "object" as const,
      properties: {
        path: {
          type: "string",
          description: "Target file path to create (required)",
        },
        force: {
          type: "boolean",
          description: "Overwrite existing file (optional, default: false)",
        },
        dry_run: {
          type: "boolean",
          description:
            "Preview generated content without writing (optional, default: false)",
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
      required: ["path"],
    },
    execute: async (input: unknown) => {
      const {
        path: targetPath,
        force = false,
        dry_run = false,
        confirmed = false,
        interactive = false,
      } = input as NewToolInput;

      // Step 1: Validate target path
      const pathValidation = validateTargetPath(targetPath, process.cwd());
      if (!pathValidation.ok) {
        return {
          created: false,
          path: targetPath,
          error: `Path validation failed: ${pathValidation.reason}`,
        } as RootlineNewResult;
      }

      // Step 2: Check if confirmation is required
      if (requiresConfirmation("create", interactive) && !confirmed && !force) {
        return {
          created: false,
          path: targetPath,
          error:
            "Confirmation required: set confirmed: true to create this file in non-interactive mode",
        } as RootlineNewResult;
      }

      // Step 3: Resolve path to absolute
      const resolvedPath = path.resolve(process.cwd(), targetPath);

      // Step 4: Build args for rootline new command
      const args = ["new", resolvedPath];

      if (force) {
        args.push("--force");
      }

      if (dry_run) {
        args.push("--dry-run");
      }

      // Note: rootline new doesn't support --output json yet, so we don't add it
      // The command outputs "Created <path>" on success or plain text content on dry-run

      // Step 5: Execute rootline new command
      // Note: rootline new doesn't support --output json, so it outputs plain text
      // The runner will fail with json_parse_failed if we try to parse as JSON
      // We need to call exec directly for plain text output
      const execResult = await pi.exec("rootline", args, {
        cwd: process.cwd(),
        timeout: 10_000,
      });

      if (execResult.code !== 0) {
        return {
          created: false,
          path: targetPath,
          error: execResult.stderr || "rootline new command failed",
        } as RootlineNewResult;
      }

      // Step 6: Handle dry-run case (returns content preview)
      if (dry_run) {
        return {
          created: false,
          path: targetPath,
          content_preview: execResult.stdout,
        } as RootlineNewResult;
      }

      // Step 7: For successful creation, validate the written file
      const validation = await validateAfterWrite(
        resolvedPath,
        run,
        process.cwd()
      );

      return {
        created: true,
        path: resolvedPath,
        validation,
      } as RootlineNewResult;
    },
  });
}
