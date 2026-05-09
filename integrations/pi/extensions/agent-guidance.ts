import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { createRunner } from "../runner";

/**
 * Injects compact Rootline guidance before agent starts.
 *
 * This extension:
 * 1. Detects project state (no_rootline, binary_only, stem_governed)
 * 2. Registers a guidance tool that returns state-specific instructions
 * 3. The guidance is designed to be called by Pi's system prompt or skill before agent execution
 *
 * Project states:
 * - no_rootline: Rootline binary not found. No guidance injected.
 * - binary_only: Binary exists but no .stem files. Lightweight init guidance.
 * - stem_governed: Binary + .stem files present. Full Rootline usage guidance.
 *
 * Guidance is compact (<10 lines) and non-blocking. Safe to call at agent start.
 *
 * @param pi ExtensionAPI instance
 */
export default function (pi: ExtensionAPI) {
  const run = createRunner(pi);

  pi.registerTool({
    name: "rootline-agent-guidance",
    description:
      "Inject compact Rootline guidance before agent runs. " +
      "Detects project state and returns state-specific instructions for using Rootline tools. " +
      "Safe to call at agent start; returns empty guidance for non-Rootline projects.",
    input_schema: {
      type: "object" as const,
      properties: {
        path: {
          type: "string",
          description:
            "Directory path to check for Rootline governance (optional, defaults to current working directory)",
        },
      },
      required: [],
    },
    execute: async (input: unknown) => {
      const { path } = input as { path?: string };
      const checkPath = path || process.cwd();

      // Try to detect project state with a fast check
      const versionResult = await run<string>(["--version"], {
        timeout: 5000,
      });

      // no_rootline: binary not found, skip guidance
      if (!versionResult.ok) {
        return {
          state: "no_rootline",
          guidance: "", // No guidance for non-Rootline projects
        };
      }

      // Binary exists, check if directory is governed
      const describeResult = await run<{ applies?: string[] }>(
        ["describe", checkPath, "--output", "json"],
        { cwd: checkPath, timeout: 5000 }
      );

      // binary_only: binary exists but no .stem files
      if (!describeResult.ok) {
        return {
          state: "binary_only",
          guidance:
            "Rootline is available but no .stem files found in this directory. " +
            "Initialize schema governance with: `rootline init --directory .`",
        };
      }

      // stem_governed: binary + .stem files present
      const stemCount = describeResult.data.applies?.length || 0;
      return {
        state: "stem_governed",
        stem_count: stemCount,
        guidance: `## Rootline Context\n` +
          `This project uses Rootline for schema-governed documentation (${stemCount} .stem file(s) active).\n` +
          `- Query records: use \`rootline-query\` tool (not grep/read)\n` +
          `- Validate: use \`rootline-validate\` tool\n` +
          `- Check schema: use \`rootline-describe\` tool\n` +
          `- Explore structure: use \`rootline-tree\` tool`,
      };
    },
  });
}
