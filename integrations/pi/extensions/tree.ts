import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { createRunner, type RunnerResult } from "../runner";

/**
 * Tool interface for rootline-tree.
 *
 * Calls `rootline tree <path> --output json [--where <expr>]`.
 */
interface TreeToolInput {
  /** Directory path to scan (required) */
  path: string;
  /** Filter expression for records (e.g. "estado == 'Pending'") */
  where?: string;
  /** Maximum depth to display (optional) */
  depth?: number;
}

/**
 * JSON output structure for tree (matches TreeResult from Go).
 */
interface TreeResult {
  version: number;
  kind: string;
  root: TreeNode;
}

/**
 * Represents a node in the directory tree.
 */
interface TreeNode {
  name: string;
  path: string;
  children?: TreeNode[];
  completed: number;
  total: number;
  is_leaf?: boolean;
  estado?: string;
}

/**
 * Registers the rootline-tree tool.
 * @param pi ExtensionAPI instance
 */
export default function (pi: ExtensionAPI) {
  const run = createRunner(pi);

  pi.registerTool({
    name: "rootline-tree",
    description:
      "Display hierarchical view of documents with completion counts. " +
      "Shows directory structure with metadata derived from frontmatter fields.",
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
            "Filter expression for records (e.g. 'estado == \"Completed\"')",
        },
        depth: {
          type: "number",
          description: "Maximum depth to display (optional)",
        },
      },
      required: ["path"],
    },
    execute: async (input: unknown) => {
      const { path, where, depth } = input as TreeToolInput;

      // Build command args
      const args = ["tree", path, "--output", "json"];

      if (where) {
        args.push("--where", where);
      }

      // Execute rootline command
      const result = await run<TreeResult>(args);

      if (!result.ok) {
        return {
          error: result.error.message,
          details: result.error.stderr || undefined,
          code: result.error.code,
        };
      }

      // If depth is specified, truncate the tree
      const tree = depth ? truncateTree(result.data.root, depth) : result.data.root;

      return {
        root: tree,
        total: tree.total,
        completed: tree.completed,
      };
    },
  });
}

/**
 * Truncates a tree to a maximum depth.
 * @param node The tree node to truncate
 * @param depth Maximum depth (0 = just root, 1 = root + immediate children, etc.)
 * @param currentDepth Current depth (used internally)
 * @returns Truncated tree
 */
function truncateTree(node: TreeNode, depth: number, currentDepth = 0): TreeNode {
  if (currentDepth >= depth || !node.children) {
    return {
      ...node,
      children: undefined,
    };
  }

  return {
    ...node,
    children: node.children.map((child) =>
      truncateTree(child, depth, currentDepth + 1)
    ),
  };
}
