import { describe, it, expect, mock } from "bun:test";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import describeExt from "./describe";

/**
 * Tests for rootline-describe extension.
 *
 * Tests verify:
 * - Tool registration with correct schema
 * - Command construction with filters and field extraction
 * - Success path: returns merged schema
 * - Field extraction from schema
 * - Failure path: rootline not found, invalid paths
 */

describe("rootline-describe extension", () => {
  describe("tool registration", () => {
    it("registers rootline-describe with correct description", () => {
      const toolMock = mock(() => {});
      const pi = {
        exec: mock(async () => ({
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/describe-result",
            path: "/docs/roadmap",
            schema: {},
          }),
          stderr: "",
          code: 0,
        })),
        tool: toolMock,
        registerTool: mock(() => {}),
      } as unknown as ExtensionAPI;

      describeExt(pi);

      expect(toolMock).toHaveBeenCalled();
      const call = toolMock.mock.calls[0];
      expect(call[0]).toBe("rootline-describe");
      expect(call[1].description).toContain("merged .stem schema");
    });
  });

  describe("command construction", () => {
    it("constructs describe command with path only", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        expect(cmd).toBe("rootline");
        expect(args).toEqual([
          "describe",
          "/docs/roadmap",
          "--output",
          "json",
        ]);

        return {
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/describe-result",
            path: "/docs/roadmap",
            schema: {},
          }),
          stderr: "",
          code: 0,
        };
      });

      let toolExecute: Function | null = null;
      const mockContext = { cwd: "/home/shared/rootline" };
      const pi = {
        exec: execMock,
        tool: (_name: string, def: any) => {
          toolExecute = def.execute;
        },
        registerTool: mock(() => {}),
      } as unknown as ExtensionAPI;

      describeExt(pi);
      await toolExecute?.({ path: "/docs/roadmap" }, mockContext);

      expect(execMock).toHaveBeenCalled();
    });

    it("includes byDomain filter when provided", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        expect(args).toContain("--by-domain");
        expect(args).toContain("governance");

        return {
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/describe-result",
            path: "/docs/roadmap",
            schema: {},
          }),
          stderr: "",
          code: 0,
        };
      });

      let toolExecute: Function | null = null;
      const mockContext = { cwd: "/home/shared/rootline" };
      const pi = {
        exec: execMock,
        tool: (_name: string, def: any) => {
          toolExecute = def.execute;
        },
        registerTool: mock(() => {}),
      } as unknown as ExtensionAPI;

      describeExt(pi);
      await toolExecute?.({ path: "/docs/roadmap", byDomain: "governance" }, mockContext);

      expect(execMock).toHaveBeenCalled();
    });
  });

  describe("success path", () => {
    it("returns full schema when no field is specified", async () => {
      const execMock = mock(async () => ({
        stdout: JSON.stringify({
          version: 1,
          kind: "rootline/describe-result",
          path: "/docs/roadmap",
          schema: {
            estado: {
              type: "string",
              enum: ["Pending", "In Progress", "Completed"],
            },
            tipo: { type: "string" },
            prioridad: { type: "number" },
          },
        }),
        stderr: "",
        code: 0,
      }));

      let toolExecute: Function | null = null;
      let result: any = null;
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

      describeExt(pi);
      const mockContext = { cwd: "/home/shared/rootline" };
      await toolExecute?.({ path: "/docs/roadmap" }, mockContext);

      expect(result).toContain("estado");
      expect(result).toContain("tipo");
      expect(result).toContain("prioridad");
    });

    it("extracts field from schema when field parameter is provided", async () => {
      const execMock = mock(async () => ({
        stdout: JSON.stringify({
          version: 1,
          kind: "rootline/describe-result",
          path: "/docs/roadmap",
          schema: {
            estado: {
              type: "string",
              enum: ["Pending", "In Progress", "Completed"],
              required: true,
            },
            tipo: { type: "string" },
          },
        }),
        stderr: "",
        code: 0,
      }));

      let toolExecute: Function | null = null;
      let result: any = null;
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

      describeExt(pi);
      const mockContext = { cwd: "/home/shared/rootline" };
      await toolExecute?.(
        { path: "/docs/roadmap", field: "estado" },
        mockContext
      );

      expect(result).toContain("estado");
      expect(result).toContain("field");
      expect(result).toContain("Pending");
    });

    it("returns error message when field not found in schema", async () => {
      const execMock = mock(async () => ({
        stdout: JSON.stringify({
          version: 1,
          kind: "rootline/describe-result",
          path: "/docs/roadmap",
          schema: {
            estado: { type: "string" },
            tipo: { type: "string" },
          },
        }),
        stderr: "",
        code: 0,
      }));

      let toolExecute: Function | null = null;
      let result: any = null;
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

      describeExt(pi);
      const mockContext = { cwd: "/home/shared/rootline" };
      await toolExecute?.(
        { path: "/docs/roadmap", field: "nonexistent" },
        mockContext
      );

      expect(result).toContain("not found");
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
        tool: (_name: string, def: any) => {
          toolExecute = async (params: any, ctx: any) => {
            result = await def.execute(params, ctx);
            return result;
          };
        },
        registerTool: mock(() => {}),
      } as unknown as ExtensionAPI;

      describeExt(pi);
      const mockContext = { cwd: "/home/shared/rootline" };
      await toolExecute?.({ path: "/docs/roadmap" }, mockContext);

      expect(result).toContain("Error:");
    });

    it("returns error when path is invalid", async () => {
      const execMock = mock(async () => ({
        stdout: "",
        stderr: "directory not found: /invalid",
        code: 1,
      }));

      let toolExecute: Function | null = null;
      let result: any = null;
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

      describeExt(pi);
      const mockContext = { cwd: "/home/shared/rootline" };
      await toolExecute?.({ path: "/invalid" }, mockContext);

      expect(result).toContain("Error:");
    });
  });
});
