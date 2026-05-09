import { describe, it, expect, mock } from "bun:test";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import stats from "./stats";

/**
 * Tests for rootline-stats extension.
 *
 * Tests verify:
 * - Tool registration with correct schema
 * - Command construction with filters
 * - Success path: returns aggregate statistics with percentages
 * - Failure path: rootline not found, invalid filters
 */

describe("rootline-stats extension", () => {
  describe("tool registration", () => {
    it("registers rootline-stats with correct description", () => {
      const registerToolMock = mock(() => {});
      const pi = {
        exec: mock(async () => ({
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/stats-result",
            by_estado: {},
            by_tipo: {},
            total: 0,
          }),
          stderr: "",
          code: 0,
        })),
        tool: mock(() => {}),
        registerTool: registerToolMock,
      } as unknown as ExtensionAPI;

      stats(pi);

      expect(registerToolMock).toHaveBeenCalled();
      const call = registerToolMock.mock.calls[0];
      const toolDef = call[0];
      expect(toolDef.name).toBe("rootline-stats");
      expect(toolDef.description).toContain("aggregate statistics");
    });
  });

  describe("command construction", () => {
    it("constructs stats command with path only", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        expect(cmd).toBe("rootline");
        expect(args).toEqual([
          "stats",
          "/docs/roadmap",
          "--output",
          "json",
        ]);

        return {
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/stats-result",
            by_estado: {},
            by_tipo: {},
            total: 0,
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

      stats(pi);
      await toolExecute?.({ path: "/docs/roadmap" });

      expect(execMock).toHaveBeenCalled();
    });

    it("includes where filter when provided", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        expect(args).toContain("--where");
        expect(args).toContain("tipo == 'epic'");

        return {
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/stats-result",
            by_estado: {},
            by_tipo: {},
            total: 0,
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

      stats(pi);
      await toolExecute?.({ path: "/docs/roadmap", where: "tipo == 'epic'" });

      expect(execMock).toHaveBeenCalled();
    });
  });

  describe("success path", () => {
    it("returns statistics with counts and percentages", async () => {
      const execMock = mock(async () => ({
        stdout: JSON.stringify({
          version: 1,
          kind: "rootline/stats-result",
          by_estado: {
            "In Progress": 5,
            Completed: 3,
            Pending: 2,
          },
          by_tipo: {
            epic: 2,
            feature: 5,
            story: 3,
          },
          total: 10,
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

      stats(pi);
      await toolExecute?.({ path: "/docs/roadmap" });

      expect(result).toBeDefined();
      expect(result.total).toBe(10);
      expect(result.by_estado).toBeDefined();
      expect(result.by_tipo).toBeDefined();
      expect(result.by_estado_pct).toBeDefined();
      expect(result.by_tipo_pct).toBeDefined();
      expect(result.by_estado_pct["In Progress"]).toBe("50.0%");
      expect(result.by_estado_pct["Completed"]).toBe("30.0%");
      expect(result.by_estado_pct["Pending"]).toBe("20.0%");
    });

    it("handles zero total gracefully", async () => {
      const execMock = mock(async () => ({
        stdout: JSON.stringify({
          version: 1,
          kind: "rootline/stats-result",
          by_estado: {},
          by_tipo: {},
          total: 0,
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

      stats(pi);
      await toolExecute?.({ path: "/docs/roadmap" });

      expect(result.total).toBe(0);
      expect(result.by_estado_pct).toBeDefined();
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

      stats(pi);
      await toolExecute?.({ path: "/docs/roadmap" });

      expect(result.error).toBeDefined();
      expect(result.code).toBe("not_found");
    });

    it("returns error when filter expression is invalid", async () => {
      const execMock = mock(async () => ({
        stdout: "",
        stderr: "invalid filter expression",
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

      stats(pi);
      await toolExecute?.({ path: "/docs/roadmap", where: "invalid" });

      expect(result.error).toBeDefined();
      expect(result.code).toBe("exec_failed");
    });
  });
});
