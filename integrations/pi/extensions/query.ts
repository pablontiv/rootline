import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { createRunner } from "../runner.js";

/**
 * Parameters for the rootline-query tool.
 */
interface QueryToolParams {
  /** Directory or file path to query from (required) */
  path: string;
  /** Filter expression using expr syntax (e.g., "estado == 'Pending'") */
  where?: string;
  /** Comma-separated field names to include in results (e.g., "path,estado,title") */
  select?: string;
  /** Maximum number of results to return */
  limit?: number;
  /** Sort specification (e.g., "prioridad:asc,impact_score:desc") */
  sort?: string;
  /** Return count instead of records */
  count?: boolean;
}

/**
 * Register the rootline-query Pi tool.
 *
 * This tool queries Rootline records by frontmatter fields and returns
 * versioned JSON results. It supports filtering, sorting, field projection,
 * and result limiting.
 *
 * @param pi - Pi ExtensionAPI instance
 */
export default function (pi: ExtensionAPI) {
  const run = createRunner(pi);

  pi.tool("rootline-query", {
    description:
      "Query Rootline records by frontmatter fields. Returns versioned JSON with rows of matching records.",
    parameters: {
      path: {
        type: "string",
        description: "Directory or file path to query from",
        required: true,
      },
      where: {
        type: "string",
        description:
          'Filter expression (e.g., "estado == \'Pending\'" or "tipo in [\'epic\', \'feature\']")',
      },
      select: {
        type: "string",
        description:
          "Comma-separated fields for compact output (e.g., 'path,estado,title')",
      },
      limit: {
        type: "number",
        description: "Maximum number of results to return (0 = unlimited)",
      },
      sort: {
        type: "string",
        description:
          'Sort by fields (e.g., "prioridad:asc,impact_score:desc")',
      },
      count: {
        type: "boolean",
        description: "Return count instead of records",
      },
    },
    execute: async (params: QueryToolParams, ctx) => {
      const args = ["query", "--from", params.path, "--output", "json"];

      if (params.where) {
        args.push("--where", params.where);
      }
      if (params.select) {
        args.push("--select", params.select);
      }
      if (params.limit !== undefined && params.limit > 0) {
        args.push("--limit", String(params.limit));
      }
      if (params.sort) {
        args.push("--sort", params.sort);
      }
      if (params.count) {
        args.push("--count");
      }

      const result = await run(args, { cwd: ctx.cwd });

      if (!result.ok) {
        return `Error: ${result.error.message}`;
      }

      return JSON.stringify(result.data, null, 2);
    },
  });
}
