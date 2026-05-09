import { describe, it, expect, mock } from "bun:test";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import query from "./query";

/**
 * Tests for rootline-query extension.
 *
 * Tests verify:
 * - Tool registration with correct name and parameters
 * - Command construction with various parameter combinations
 * - Success path: valid JSON output parsing
 * - Failure path: rootline not found, non-zero exit codes
 */

const mockContext = {
  cwd: "/home/shared/rootline",
};

describe("rootline-query extension", () => {
  describe("tool registration", () => {
    it("registers rootline-query tool with correct description", () => {
      const toolMock = mock(() => {});
      const pi = {
        exec: mock(async () => ({
          stdout: JSON.stringify({ rows: [] }),
          stderr: "",
          code: 0,
        })),
        tool: toolMock,
        registerTool: mock(() => {}),
      } as unknown as ExtensionAPI;

      query(pi);

      expect(toolMock).toHaveBeenCalled();
      const call = toolMock.mock.calls[0];
      expect(call[0]).toBe("rootline-query");
      expect(call[1].description).toContain("Query Rootline records");
      expect(call[1].parameters.path.required).toBe(true);
    });
  });

  describe("command construction", () => {
    it("constructs query with path only", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        expect(cmd).toBe("rootline");
        expect(args).toEqual([
          "query",
          "--from",
          "/docs/roadmap",
          "--output",
          "json",
        ]);
        return {
          stdout: JSON.stringify({ rows: [], total: 0 }),
          stderr: "",
          code: 0,
        };
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        tool: (_name: string, def: any) => {
          toolExecute = def.execute;
        },
        registerTool: mock(() => {}),
      } as unknown as ExtensionAPI;

      query(pi);
      await toolExecute?.({ path: "/docs/roadmap" }, mockContext);

      expect(execMock).toHaveBeenCalled();
    });

    it("constructs query with all optional parameters", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        expect(cmd).toBe("rootline");
        expect(args).toContain("query");
        expect(args).toContain("--from");
        expect(args).toContain("/docs/roadmap");
        expect(args).toContain("--where");
        expect(args).toContain("estado == 'Pending'");
        expect(args).toContain("--select");
        expect(args).toContain("path,estado,title");
        expect(args).toContain("--limit");
        expect(args).toContain("10");
        expect(args).toContain("--sort");
        expect(args).toContain("prioridad:asc");

        return {
          stdout: JSON.stringify({ rows: [], total: 0 }),
          stderr: "",
          code: 0,
        };
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        tool: (_name: string, def: any) => {
          toolExecute = def.execute;
        },
        registerTool: mock(() => {}),
      } as unknown as ExtensionAPI;

      query(pi);
      await toolExecute?.(
        {
          path: "/docs/roadmap",
          where: "estado == 'Pending'",
          select: "path,estado,title",
          limit: 10,
          sort: "prioridad:asc",
        },
        mockContext
      );

      expect(execMock).toHaveBeenCalled();
    });

    it("omits optional parameters when not provided", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        expect(args).not.toContain("--where");
        expect(args).not.toContain("--select");
        expect(args).not.toContain("--limit");
        expect(args).not.toContain("--sort");

        return {
          stdout: JSON.stringify({ rows: [], total: 0 }),
          stderr: "",
          code: 0,
        };
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        tool: (_name: string, def: any) => {
          toolExecute = def.execute;
        },
        registerTool: mock(() => {}),
      } as unknown as ExtensionAPI;

      query(pi);
      await toolExecute?.({ path: "/docs/roadmap" }, mockContext);

      expect(execMock).toHaveBeenCalled();
    });

    it("skips limit when limit is 0 or negative", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        expect(args).not.toContain("--limit");
        return {
          stdout: JSON.stringify({ rows: [], total: 0 }),
          stderr: "",
          code: 0,
        };
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        tool: (_name: string, def: any) => {
          toolExecute = def.execute;
        },
        registerTool: mock(() => {}),
      } as unknown as ExtensionAPI;

      query(pi);
      await toolExecute?.({ path: "/docs/roadmap", limit: 0 }, mockContext);

      expect(execMock).toHaveBeenCalled();
    });
  });

  describe("success path", () => {
    it("parses JSON response and returns formatted output", async () => {
      const execMock = mock(async () => ({
        stdout: JSON.stringify({
          version: 1,
          kind: "rootline/query-result",
          rows: [
            { path: "docs/E01.md", estado: "In Progress" },
            { path: "docs/E02.md", estado: "Pending" },
          ],
          total: 2,
        }),
        stderr: "",
        code: 0,
      }));

      let toolExecute: Function | null = null;
      let result: string = "";
      const pi = {
        exec: execMock,
        tool: (_name: string, def: any) => {
          toolExecute = async (params: any, ctx: any) => {
            result = await def.execute(params, ctx);
            return result;
          };
        },
        registerTool: mock(() => {}),
      } as unknown as ExtensionAPI;

      query(pi);
      await toolExecute?.({ path: "/docs/roadmap" }, mockContext);

      expect(result).toContain("query-result");
      expect(result).toContain("E01.md");
    });
  });

  describe("failure paths", () => {
    it("returns error message when rootline not found", async () => {
      const execMock = mock(async () => {
        throw new Error("ENOENT: rootline not found");
      });

      let toolExecute: Function | null = null;
      let result: string = "";
      const pi = {
        exec: execMock,
        tool: (_name: string, def: any) => {
          toolExecute = async (params: any, ctx: any) => {
            result = await def.execute(params, ctx);
            return result;
          };
        },
        registerTool: mock(() => {}),
      } as unknown as ExtensionAPI;

      query(pi);
      await toolExecute?.({ path: "/docs/roadmap" }, mockContext);

      expect(result).toContain("Error:");
      expect(result).toContain("not found");
    });

    it("returns error message when rootline exits with non-zero code", async () => {
      const execMock = mock(async () => ({
        stdout: "",
        stderr: "directory not found: /invalid/path",
        code: 1,
      }));

      let toolExecute: Function | null = null;
      let result: string = "";
      const pi = {
        exec: execMock,
        tool: (_name: string, def: any) => {
          toolExecute = async (params: any, ctx: any) => {
            result = await def.execute(params, ctx);
            return result;
          };
        },
        registerTool: mock(() => {}),
      } as unknown as ExtensionAPI;

      query(pi);
      await toolExecute?.({ path: "/invalid/path" }, mockContext);

      expect(result).toContain("Error:");
      expect(result).toContain("not found");
    });

    it("returns error for timeout", async () => {
      const execMock = mock(async () => {
        throw new Error("ETIMEDOUT");
      });

      let toolExecute: Function | null = null;
      let result: string = "";
      const pi = {
        exec: execMock,
        tool: (_name: string, def: any) => {
          toolExecute = async (params: any, ctx: any) => {
            result = await def.execute(params, ctx);
            return result;
          };
        },
        registerTool: mock(() => {}),
      } as unknown as ExtensionAPI;

      query(pi);
      await toolExecute?.({ path: "/docs/roadmap" }, mockContext);

      expect(result).toContain("Error:");
    });
  });
});
