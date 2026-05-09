import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { createRunner, type RunnerResult } from "../runner";

/**
 * Tool interface for rootline-stats.
 *
 * Calls `rootline stats <path> --output json [--where <expr>]`.
 */
interface StatsToolInput {
  /** Directory path to scan (required) */
  path: string;
  /** Filter expression for records (e.g. "estado == 'Pending'") */
  where?: string;
  /** Aggregate by a specific field name (optional) */
  by?: string;
}

/**
 * JSON output structure for stats (matches StatsResult from Go).
 */
interface StatsResult {
  version: number;
  kind: string;
  by_estado: Record<string, number>;
  by_tipo: Record<string, number>;
  total: number;
}

/**
 * Registers the rootline-stats tool.
 * @param pi ExtensionAPI instance
 */
export default function (pi: ExtensionAPI) {
  const run = createRunner(pi);

  pi.registerTool({
    name: "rootline-stats",
    description:
      "Show aggregate statistics for documents: counts by type, state, and other fields. " +
      "Returns summary counts derived from frontmatter analysis.",
    input_schema: {
      type: "object" as const,
      properties: {
        path: {
          type: "string",
          description: "Directory path to scan",
        },
        where: {
          type: "string",
          description:
            "Filter expression for records (e.g. 'estado == \"In Progress\"')",
        },
        by: {
          type: "string",
          description:
            "Aggregate by specific field name (optional, not yet supported in rootline stats)",
        },
      },
      required: ["path"],
    },
    execute: async (input: unknown) => {
      const { path, where, by } = input as StatsToolInput;

      // Build command args
      const args = ["stats", path, "--output", "json"];

      if (where) {
        args.push("--where", where);
      }

      // Note: 'by' parameter is included in schema for future expansion
      // but not yet supported by rootline stats CLI

      // Execute rootline command
      const result = await run<StatsResult>(args);

      if (!result.ok) {
        return {
          error: result.error.message,
          details: result.error.stderr || undefined,
          code: result.error.code,
        };
      }

      // Format results with percentages
      return {
        total: result.data.total,
        by_estado: result.data.by_estado,
        by_tipo: result.data.by_tipo,
        by_estado_pct: formatPercentages(result.data.by_estado, result.data.total),
        by_tipo_pct: formatPercentages(result.data.by_tipo, result.data.total),
      };
    },
  });
}

/**
 * Formats field counts as percentages.
 * @param counts Map of field values to counts
 * @param total Total number of records
 * @returns Map of field values to percentage strings
 */
function formatPercentages(
  counts: Record<string, number>,
  total: number
): Record<string, string> {
  const pct: Record<string, string> = {};
  for (const [key, count] of Object.entries(counts)) {
    if (total === 0) {
      pct[key] = "0%";
    } else {
      pct[key] = `${((count / total) * 100).toFixed(1)}%`;
    }
  }
  return pct;
}
