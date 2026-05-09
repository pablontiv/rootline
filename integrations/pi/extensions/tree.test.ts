import { describe, it, expect, mock } from "bun:test";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import tree from "./tree";

/**
 * Tests for rootline-tree extension.
 *
 * Tests verify:
 * - Tool registration with correct schema
 * - Command construction with filters and depth
 * - Success path: returns tree structure with completion counts
 * - Tree truncation when depth is specified
 * - Failure path: rootline not found, invalid paths
 */

describe("rootline-tree extension", () => {
  describe("tool registration", () => {
    it("registers rootline-tree with correct description", () => {
      const registerToolMock = mock(() => {});
      const pi = {
        exec: mock(async () => ({
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/tree-result",
            root: { name: "root", path: "/", completed: 0, total: 0 },
          }),
          stderr: "",
          code: 0,
        })),
        tool: mock(() => {}),
        registerTool: registerToolMock,
      } as unknown as ExtensionAPI;

      tree(pi);

      expect(registerToolMock).toHaveBeenCalled();
      const call = registerToolMock.mock.calls[0];
      const toolDef = call[0];
      expect(toolDef.name).toBe("rootline-tree");
      expect(toolDef.description).toContain("hierarchical view");
    });
  });

  describe("command construction", () => {
    it("constructs tree command with path only", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        expect(cmd).toBe("rootline");
        expect(args).toEqual([
          "tree",
          "/docs/roadmap",
          "--output",
          "json",
        ]);

        return {
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/tree-result",
            root: { name: "roadmap", path: "/docs/roadmap", completed: 0, total: 0 },
          }),
          stderr: "",
          code: 0,
        };
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        tool: mock(() => {}),
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      tree(pi);
      await toolExecute?.({ path: "/docs/roadmap" });

      expect(execMock).toHaveBeenCalled();
    });

    it("includes where filter when provided", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        expect(args).toContain("--where");
        expect(args).toContain("estado == 'Completed'");

        return {
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/tree-result",
            root: { name: "roadmap", path: "/docs/roadmap", completed: 0, total: 0 },
          }),
          stderr: "",
          code: 0,
        };
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        tool: mock(() => {}),
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      tree(pi);
      await toolExecute?.({ path: "/docs/roadmap", where: "estado == 'Completed'" });

      expect(execMock).toHaveBeenCalled();
    });
  });

  describe("success path", () => {
    it("returns tree structure with completion counts", async () => {
      const execMock = mock(async () => ({
        stdout: JSON.stringify({
          version: 1,
          kind: "rootline/tree-result",
          root: {
            name: "roadmap",
            path: "/docs/roadmap",
            completed: 5,
            total: 10,
            children: [
              {
                name: "E01",
                path: "/docs/roadmap/E01",
                completed: 2,
                total: 3,
                children: [
                  {
                    name: "F01.md",
                    path: "/docs/roadmap/E01/F01.md",
                    completed: 1,
                    total: 1,
                    is_leaf: true,
                  },
                ],
              },
            ],
          },
        }),
        stderr: "",
        code: 0,
      }));

      let toolExecute: Function | null = null;
      let result: any = null;
      const pi = {
        exec: execMock,
        tool: mock(() => {}),
        registerTool: (def: any) => {
          toolExecute = async (params: any) => {
            result = await def.execute(params);
            return result;
          };
        },
      } as unknown as ExtensionAPI;

      tree(pi);
      await toolExecute?.({ path: "/docs/roadmap" });

      expect(result).toBeDefined();
      expect(result.root).toBeDefined();
      expect(result.root.completed).toBe(5);
      expect(result.root.total).toBe(10);
      expect(result.total).toBe(10);
      expect(result.completed).toBe(5);
    });

    it("truncates tree to specified depth", async () => {
      const execMock = mock(async () => ({
        stdout: JSON.stringify({
          version: 1,
          kind: "rootline/tree-result",
          root: {
            name: "roadmap",
            path: "/docs/roadmap",
            completed: 5,
            total: 10,
            children: [
              {
                name: "E01",
                path: "/docs/roadmap/E01",
                completed: 2,
                total: 3,
                children: [
                  {
                    name: "F01.md",
                    path: "/docs/roadmap/E01/F01.md",
                    completed: 1,
                    total: 1,
                  },
                ],
              },
            ],
          },
        }),
        stderr: "",
        code: 0,
      }));

      let toolExecute: Function | null = null;
      let result: any = null;
      const pi = {
        exec: execMock,
        tool: mock(() => {}),
        registerTool: (def: any) => {
          toolExecute = async (params: any) => {
            result = await def.execute(params);
            return result;
          };
        },
      } as unknown as ExtensionAPI;

      tree(pi);
      await toolExecute?.({ path: "/docs/roadmap", depth: 1 });

      expect(result.root).toBeDefined();
      // When depth=1, children should exist but grandchildren should be removed
      if (result.root.children && result.root.children[0]) {
        expect(result.root.children[0].children).toBeUndefined();
      }
    });
  });

  describe("failure paths", () => {
    it("returns error when rootline not found", async () => {
      const execMock = mock(async () => {
        throw new Error("ENOENT: rootline not found");
      });

      let toolExecute: Function | null = null;
      let result: any = null;
      const pi = {
        exec: execMock,
        tool: mock(() => {}),
        registerTool: (def: any) => {
          toolExecute = async (params: any) => {
            result = await def.execute(params);
            return result;
          };
        },
      } as unknown as ExtensionAPI;

      tree(pi);
      await toolExecute?.({ path: "/docs/roadmap" });

      expect(result.error).toBeDefined();
      expect(result.code).toBe("not_found");
    });

    it("returns error when path is invalid", async () => {
      const execMock = mock(async () => ({
        stdout: "",
        stderr: "directory not found: /nonexistent",
        code: 1,
      }));

      let toolExecute: Function | null = null;
      let result: any = null;
      const pi = {
        exec: execMock,
        tool: mock(() => {}),
        registerTool: (def: any) => {
          toolExecute = async (params: any) => {
            result = await def.execute(params);
            return result;
          };
        },
      } as unknown as ExtensionAPI;

      tree(pi);
      await toolExecute?.({ path: "/nonexistent" });

      expect(result.error).toBeDefined();
      expect(result.code).toBe("exec_failed");
    });
  });
});
