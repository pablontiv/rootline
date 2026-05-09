import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { createRunner } from "../runner.js";

/**
 * Parameters for the rootline-describe tool.
 */
interface DescribeToolParams {
  /** Directory or file path to describe schema for (required) */
  path: string;
  /** Optional field name or dot path to extract from schema (e.g., "Estado", "schema.Estado") */
  field?: string;
  /** Filter schema fields by domain (e.g., "governance", "metadata") */
  byDomain?: string;
}

/**
 * Register the rootline-describe Pi tool.
 *
 * This tool shows the effective (merged) .stem schema that applies to a
 * directory or file, including inherited rules and local overrides.
 *
 * @param pi - Pi ExtensionAPI instance
 */
export default function (pi: ExtensionAPI) {
  const run = createRunner(pi);

  pi.tool("rootline-describe", {
    description:
      "Show the effective merged .stem schema for a directory or file, including inherited rules.",
    parameters: {
      path: {
        type: "string",
        description: "Directory or file path to describe schema for",
        required: true,
      },
      field: {
        type: "string",
        description:
          "Optional field name to extract from the schema (e.g., 'estado', 'tipo')",
      },
      byDomain: {
        type: "string",
        description:
          "Filter schema fields by domain (e.g., 'governance', 'metadata')",
      },
    },
    execute: async (params: DescribeToolParams, ctx) => {
      const args = ["describe", params.path, "--output", "json"];

      if (params.byDomain) {
        args.push("--by-domain", params.byDomain);
      }

      const result = await run(args, { cwd: ctx.cwd });

      if (!result.ok) {
        return `Error: ${result.error.message}`;
      }

      // If field extraction is requested, extract it from the schema
      if (params.field && typeof result.data === "object" && result.data !== null) {
        const data = result.data as Record<string, any>;
        if (data.schema && typeof data.schema === "object") {
          const schema = data.schema as Record<string, any>;
          if (params.field in schema) {
            return JSON.stringify(
              { field: params.field, definition: schema[params.field] },
              null,
              2
            );
          } else {
            return `Field '${params.field}' not found in schema`;
          }
        }
      }

      return JSON.stringify(result.data, null, 2);
    },
  });
}
