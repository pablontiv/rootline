import { describe, it, expect, mock } from "bun:test";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import setExt from "./set";

/**
 * Tests for rootline-set extension.
 *
 * Tests verify:
 * - Path validation: rejects unsafe paths (../ escape, outside cwd)
 * - Confirmation requirements: enforces confirmation in non-interactive mode
 * - Command construction: builds correct rootline set arguments
 * - Dry-run mode: returns preview without mutation
 * - Success path: updates field and validates after write
 * - Failure paths: invalid fields, validation errors, rootline failures
 */

describe("rootline-set extension", () => {
  describe("tool registration", () => {
    it("registers rootline-set with correct description", () => {
      const registerMock = mock(() => {});
      const pi = {
        exec: mock(async () => ({
          stdout: "set estado = \"Pending\"\n",
          stderr: "",
          code: 0,
        })),
        registerTool: registerMock,
      } as unknown as ExtensionAPI;

      setExt(pi);

      expect(registerMock).toHaveBeenCalled();
      const call = registerMock.mock.calls[0];
      const toolDef = call[0] as any;
      expect(toolDef.name).toBe("rootline-set");
      expect(toolDef.description).toContain("path safety guardrails");
    });
  });

  describe("path validation", () => {
    it("rejects paths outside cwd", async () => {
      const execMock = mock(async () => {
        throw new Error("Should not execute");
      });

      let toolExecute: Function | null = null;
      let result: any = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = async (params: any, ctx: any) => {
            result = await def.execute(params, ctx);
            return result;
          };
        },
      } as unknown as ExtensionAPI;

      setExt(pi);
      const mockContext = { cwd: "/home/shared/rootline" };
      await toolExecute?.(
        {
          path: "../../../etc/passwd",
          fields: [{ field: "estado", value: "pending" }],
        },
        mockContext
      );

      expect(result.updated).toBe(false);
      expect(result.error).toContain("escapes");
    });

    it("allows safe relative paths within cwd", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        expect(args[0]).toBe("set");
        expect(args).toContain("estado=pending");
        return {
          stdout: 'set estado = "pending"\n',
          stderr: "",
          code: 0,
        };
      });

      let toolExecute: Function | null = null;
      let result: any = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = async (params: any, ctx: any) => {
            result = await def.execute(params, ctx);
            return result;
          };
        },
      } as unknown as ExtensionAPI;

      setExt(pi);
      const mockContext = { cwd: "/home/shared/rootline" };
      await toolExecute?.(
        {
          path: "docs/roadmap/E01.md",
          fields: [{ field: "estado", value: "pending" }],
          interactive: true, // Skip confirmation in interactive mode
        },
        mockContext
      );

      // Should have called exec with safe path
      expect(execMock).toHaveBeenCalled();
    });
  });

  describe("confirmation requirements", () => {
    it("requires confirmation in non-interactive mode", async () => {
      const execMock = mock(async () => {
        throw new Error("Should not execute");
      });

      let toolExecute: Function | null = null;
      let result: any = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = async (params: any, ctx: any) => {
            result = await def.execute(params, ctx);
            return result;
          };
        },
      } as unknown as ExtensionAPI;

      setExt(pi);
      const mockContext = { cwd: "/home/shared/rootline" };
      await toolExecute?.(
        {
          path: "docs/roadmap/E01.md",
          fields: [{ field: "estado", value: "pending" }],
          interactive: false, // Non-interactive
          confirmed: false, // Not confirmed
        },
        mockContext
      );

      expect(result.updated).toBe(false);
      expect(result.error).toContain("Confirmation required");
    });

    it("allows mutation when confirmed in non-interactive mode", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        return {
          stdout: 'set estado = "pending"\n',
          stderr: "",
          code: 0,
        };
      });

      let toolExecute: Function | null = null;
      let result: any = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = async (params: any, ctx: any) => {
            result = await def.execute(params, ctx);
            return result;
          };
        },
      } as unknown as ExtensionAPI;

      setExt(pi);
      const mockContext = { cwd: "/home/shared/rootline" };
      await toolExecute?.(
        {
          path: "docs/roadmap/E01.md",
          fields: [{ field: "estado", value: "pending" }],
          interactive: false,
          confirmed: true, // Confirmed
        },
        mockContext
      );

      expect(execMock).toHaveBeenCalled();
    });

    it("allows mutation without confirmation in interactive mode", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        return {
          stdout: 'set estado = "pending"\n',
          stderr: "",
          code: 0,
        };
      });

      let toolExecute: Function | null = null;
      let result: any = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = async (params: any, ctx: any) => {
            result = await def.execute(params, ctx);
            return result;
          };
        },
      } as unknown as ExtensionAPI;

      setExt(pi);
      const mockContext = { cwd: "/home/shared/rootline" };
      await toolExecute?.(
        {
          path: "docs/roadmap/E01.md",
          fields: [{ field: "estado", value: "pending" }],
          interactive: true, // Interactive mode
          confirmed: false, // Not explicitly confirmed (not needed)
        },
        mockContext
      );

      expect(execMock).toHaveBeenCalled();
    });
  });

  describe("command construction", () => {
    it("builds correct set command with single field", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        expect(cmd).toBe("rootline");
        expect(args[0]).toBe("set");
        expect(args[1]).toContain("/home/shared/rootline/docs/roadmap/E01.md");
        expect(args[2]).toBe("estado=pending");
        return {
          stdout: 'set estado = "pending"\n',
          stderr: "",
          code: 0,
        };
      });

      let toolExecute: Function | null = null;
      let result: any = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = async (params: any, ctx: any) => {
            result = await def.execute(params, ctx);
            return result;
          };
        },
      } as unknown as ExtensionAPI;

      setExt(pi);
      const mockContext = { cwd: "/home/shared/rootline" };
      await toolExecute?.(
        {
          path: "docs/roadmap/E01.md",
          fields: [{ field: "estado", value: "pending" }],
          interactive: true,
        },
        mockContext
      );

      expect(execMock).toHaveBeenCalled();
    });

    it("builds correct set command with multiple fields", async () => {
      const fieldArgs: string[] = [];
      const execMock = mock(async (cmd: string, args: string[]) => {
        expect(cmd).toBe("rootline");
        expect(args[0]).toBe("set");
        // Capture field args (skip "set" and path)
        for (let i = 2; i < args.length; i++) {
          if (!args[i].startsWith("--")) {
            fieldArgs.push(args[i]);
          }
        }
        return {
          stdout:
            'set estado = "pending"\nset prioridad = "high"\n',
          stderr: "",
          code: 0,
        };
      });

      let toolExecute: Function | null = null;
      let result: any = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = async (params: any, ctx: any) => {
            result = await def.execute(params, ctx);
            return result;
          };
        },
      } as unknown as ExtensionAPI;

      setExt(pi);
      const mockContext = { cwd: "/home/shared/rootline" };
      await toolExecute?.(
        {
          path: "docs/roadmap/E01.md",
          fields: [
            { field: "estado", value: "pending" },
            { field: "prioridad", value: "high" },
          ],
          interactive: true,
        },
        mockContext
      );

      expect(fieldArgs).toContain("estado=pending");
      expect(fieldArgs).toContain("prioridad=high");
    });

    it("includes --dry-run flag when requested", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        expect(args).toContain("--dry-run");
        return {
          stdout: 'would set estado = "pending"\n',
          stderr: "",
          code: 0,
        };
      });

      let toolExecute: Function | null = null;
      let result: any = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = async (params: any, ctx: any) => {
            result = await def.execute(params, ctx);
            return result;
          };
        },
      } as unknown as ExtensionAPI;

      setExt(pi);
      const mockContext = { cwd: "/home/shared/rootline" };
      await toolExecute?.(
        {
          path: "docs/roadmap/E01.md",
          fields: [{ field: "estado", value: "pending" }],
          dry_run: true,
          interactive: true,
        },
        mockContext
      );

      expect(execMock).toHaveBeenCalled();
    });
  });

  describe("dry-run mode", () => {
    it("returns preview without mutation", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        expect(args).toContain("--dry-run");
        return {
          stdout: 'would set estado = "pending"\nwould set prioridad = "high"\n',
          stderr: "",
          code: 0,
        };
      });

      let toolExecute: Function | null = null;
      let result: any = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = async (params: any, ctx: any) => {
            result = await def.execute(params, ctx);
            return result;
          };
        },
      } as unknown as ExtensionAPI;

      setExt(pi);
      const mockContext = { cwd: "/home/shared/rootline" };
      await toolExecute?.(
        {
          path: "docs/roadmap/E01.md",
          fields: [
            { field: "estado", value: "pending" },
            { field: "prioridad", value: "high" },
          ],
          dry_run: true,
          interactive: true,
        },
        mockContext
      );

      expect(result.updated).toBe(false);
      expect(result.dry_run_preview).toBeDefined();
      expect(result.dry_run_preview?.length).toBeGreaterThan(0);
    });
  });

  describe("success path with validation", () => {
    it("updates field and validates after write", async () => {
      const validateMock = mock(async (cmd: string, args: string[]) => {
        if (args[0] === "set") {
          return {
            stdout: 'set estado = "pending"\n',
            stderr: "",
            code: 0,
          };
        } else if (args[0] === "validate") {
          return {
            stdout: JSON.stringify({
              version: 1,
              kind: "rootline/batch-validation-result",
              results: [
                {
                  path: "E01.md",
                  valid: true,
                  errors: [],
                },
              ],
              summary: {
                total: 1,
                valid: 1,
                invalid: 0,
              },
            }),
            stderr: "",
            code: 0,
          };
        }
        return {
          stdout: "{}",
          stderr: "",
          code: 0,
        };
      });

      let toolExecute: Function | null = null;
      let result: any = null;
      const pi = {
        exec: validateMock,
        registerTool: (def: any) => {
          toolExecute = async (params: any, ctx: any) => {
            result = await def.execute(params, ctx);
            return result;
          };
        },
      } as unknown as ExtensionAPI;

      setExt(pi);
      const mockContext = { cwd: "/home/shared/rootline" };
      await toolExecute?.(
        {
          path: "docs/roadmap/E01.md",
          fields: [{ field: "estado", value: "pending" }],
          interactive: true,
        },
        mockContext
      );

      expect(result.updated).toBe(true);
      expect(result.fields_set).toContain("estado=pending");
      expect(result.validation?.ok).toBe(true);
    });
  });

  describe("failure paths", () => {
    it("returns error when rootline set fails", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        return {
          stdout: "",
          stderr: "field not found: invalid_field",
          code: 1,
        };
      });

      let toolExecute: Function | null = null;
      let result: any = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = async (params: any, ctx: any) => {
            result = await def.execute(params, ctx);
            return result;
          };
        },
      } as unknown as ExtensionAPI;

      setExt(pi);
      const mockContext = { cwd: "/home/shared/rootline" };
      await toolExecute?.(
        {
          path: "docs/roadmap/E01.md",
          fields: [{ field: "invalid_field", value: "value" }],
          interactive: true,
        },
        mockContext
      );

      expect(result.updated).toBe(false);
      expect(result.error).toBeDefined();
    });

    it("returns error when post-mutation validation fails", async () => {
      const validateMock = mock(async (cmd: string, args: string[]) => {
        if (args[0] === "set") {
          return {
            stdout: 'set estado = "invalid"\n',
            stderr: "",
            code: 0,
          };
        } else if (args[0] === "validate") {
          return {
            stdout: JSON.stringify({
              version: 1,
              kind: "rootline/batch-validation-result",
              results: [
                {
                  path: "E01.md",
                  valid: false,
                  errors: [
                    {
                      field: "estado",
                      rule: "enum",
                      message: "value not in allowed enum",
                    },
                  ],
                },
              ],
              summary: {
                total: 1,
                valid: 0,
                invalid: 1,
              },
            }),
            stderr: "",
            code: 0,
          };
        }
        return {
          stdout: "{}",
          stderr: "",
          code: 0,
        };
      });

      let toolExecute: Function | null = null;
      let result: any = null;
      const pi = {
        exec: validateMock,
        registerTool: (def: any) => {
          toolExecute = async (params: any, ctx: any) => {
            result = await def.execute(params, ctx);
            return result;
          };
        },
      } as unknown as ExtensionAPI;

      setExt(pi);
      const mockContext = { cwd: "/home/shared/rootline" };
      await toolExecute?.(
        {
          path: "docs/roadmap/E01.md",
          fields: [{ field: "estado", value: "invalid" }],
          interactive: true,
        },
        mockContext
      );

      expect(result.updated).toBe(false);
      expect(result.error).toContain("validation failed");
    });

    it("returns error when rootline binary not found", async () => {
      const execMock = mock(async () => {
        throw new Error("ENOENT: rootline not found");
      });

      let toolExecute: Function | null = null;
      let result: any = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = async (params: any, ctx: any) => {
            result = await def.execute(params, ctx);
            return result;
          };
        },
      } as unknown as ExtensionAPI;

      setExt(pi);
      const mockContext = { cwd: "/home/shared/rootline" };
      await toolExecute?.(
        {
          path: "docs/roadmap/E01.md",
          fields: [{ field: "estado", value: "pending" }],
          interactive: true,
        },
        mockContext
      );

      expect(result.updated).toBe(false);
      expect(result.error).toContain("Failed to execute");
    });
  });

  describe("input validation", () => {
    it("handles empty fields array", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        // Should still run set command even with no fields
        return {
          stdout: "",
          stderr: "",
          code: 0,
        };
      });

      let toolExecute: Function | null = null;
      let result: any = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = async (params: any, ctx: any) => {
            result = await def.execute(params, ctx);
            return result;
          };
        },
      } as unknown as ExtensionAPI;

      setExt(pi);
      const mockContext = { cwd: "/home/shared/rootline" };
      await toolExecute?.(
        {
          path: "docs/roadmap/E01.md",
          fields: [],
          interactive: true,
        },
        mockContext
      );

      expect(execMock).toHaveBeenCalled();
    });

    it("handles special characters in field values", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        expect(args).toContain('description=This is a "quoted" value');
        return {
          stdout: 'set description = "This is a \\"quoted\\" value"\n',
          stderr: "",
          code: 0,
        };
      });

      let toolExecute: Function | null = null;
      let result: any = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = async (params: any, ctx: any) => {
            result = await def.execute(params, ctx);
            return result;
          };
        },
      } as unknown as ExtensionAPI;

      setExt(pi);
      const mockContext = { cwd: "/home/shared/rootline" };
      await toolExecute?.(
        {
          path: "docs/roadmap/E01.md",
          fields: [{ field: "description", value: 'This is a "quoted" value' }],
          interactive: true,
        },
        mockContext
      );

      expect(execMock).toHaveBeenCalled();
    });
  });
});
