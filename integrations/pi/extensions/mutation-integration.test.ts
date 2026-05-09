import { describe, it, expect, mock } from "bun:test";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import newTool from "./new";
import setTool from "./set";

/**
 * Integration tests for rootline mutation tools.
 *
 * Tests verify end-to-end scenarios combining rootline-new and rootline-set with validation:
 * - Blocked path case: path traversal attacks are rejected
 * - Confirmation required case: non-interactive mode enforces confirmation requirement
 * - Validation failure surfaced to model: errors are accessible in tool results
 * - Success case end-to-end: mutations succeed with valid result
 * - Binary not found case: graceful error handling when rootline is missing
 */

describe("rootline mutation tools integration", () => {
  describe("blocked path case", () => {
    it("rootline-new rejects path traversal attacks", async () => {
      const execMock = mock(async () => {
        throw new Error("Should not execute");
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        tool: mock(() => {}),
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      newTool(pi);

      const result = await toolExecute?.({
        path: "../../../../etc/passwd",
      });

      expect(result.created).toBe(false);
      expect(result.error).toBeDefined();
      expect(result.error).toContain("escapes");
      expect(execMock).not.toHaveBeenCalled();
    });

    it("rootline-set rejects path traversal attacks", async () => {
      const execMock = mock(async () => {
        throw new Error("Should not execute");
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      setTool(pi);
      const mockContext = { cwd: "/home/shared/rootline" };

      const result = await toolExecute?.(
        {
          path: "../../../../etc/passwd",
          fields: [{ field: "estado", value: "pending" }],
        },
        mockContext
      );

      expect(result.updated).toBe(false);
      expect(result.error).toBeDefined();
      expect(result.error).toContain("escapes");
      expect(execMock).not.toHaveBeenCalled();
    });

    it("blocks absolute paths outside cwd for rootline-new", async () => {
      const execMock = mock(async () => {
        throw new Error("Should not execute");
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        tool: mock(() => {}),
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      newTool(pi);

      const result = await toolExecute?.({
        path: "/etc/passwd",
      });

      expect(result.created).toBe(false);
      expect(result.error).toBeDefined();
      expect(result.error).toContain("outside");
      expect(execMock).not.toHaveBeenCalled();
    });

    it("blocks absolute paths outside cwd for rootline-set", async () => {
      const execMock = mock(async () => {
        throw new Error("Should not execute");
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      setTool(pi);
      const mockContext = { cwd: "/home/shared/rootline" };

      const result = await toolExecute?.(
        {
          path: "/etc/passwd",
          fields: [{ field: "estado", value: "pending" }],
        },
        mockContext
      );

      expect(result.updated).toBe(false);
      expect(result.error).toBeDefined();
      expect(execMock).not.toHaveBeenCalled();
    });
  });

  describe("confirmation required case", () => {
    it("rootline-new requires confirmation in non-interactive mode without confirmed flag", async () => {
      const execMock = mock(async () => {
        throw new Error("Should not execute");
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        tool: mock(() => {}),
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      newTool(pi);

      const result = await toolExecute?.({
        path: "docs/file.md",
        interactive: false,
        confirmed: false,
      });

      expect(result.created).toBe(false);
      expect(result.error).toBeDefined();
      expect(result.error).toContain("Confirmation required");
      expect(execMock).not.toHaveBeenCalled();
    });

    it("rootline-new succeeds with confirmed: true in non-interactive mode", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        if (args[0] === "new") {
          return {
            stdout: "Created /path/to/file.md\n",
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
                  path: "/path/to/file.md",
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
        throw new Error("Unexpected command");
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        tool: mock(() => {}),
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      newTool(pi);

      const result = await toolExecute?.({
        path: "docs/file.md",
        interactive: false,
        confirmed: true,
      });

      expect(result.created).toBe(true);
      expect(execMock).toHaveBeenCalled();
    });

    it("rootline-set requires confirmation in non-interactive mode without confirmed flag", async () => {
      const execMock = mock(async () => {
        throw new Error("Should not execute");
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      setTool(pi);
      const mockContext = { cwd: "/home/shared/rootline" };

      const result = await toolExecute?.(
        {
          path: "docs/file.md",
          fields: [{ field: "estado", value: "pending" }],
          interactive: false,
          confirmed: false,
        },
        mockContext
      );

      expect(result.updated).toBe(false);
      expect(result.error).toBeDefined();
      expect(result.error).toContain("Confirmation required");
      expect(execMock).not.toHaveBeenCalled();
    });

    it("rootline-set succeeds with confirmed: true in non-interactive mode", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
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
                  path: "file.md",
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
        throw new Error("Unexpected command");
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      setTool(pi);
      const mockContext = { cwd: "/home/shared/rootline" };

      const result = await toolExecute?.(
        {
          path: "docs/file.md",
          fields: [{ field: "estado", value: "pending" }],
          interactive: false,
          confirmed: true,
        },
        mockContext
      );

      expect(result.updated).toBe(true);
      expect(execMock).toHaveBeenCalled();
    });

    it("rootline-new allows creation without confirmation in interactive mode", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        if (args[0] === "new") {
          return {
            stdout: "Created /path/to/file.md\n",
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
                  path: "/path/to/file.md",
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
        throw new Error("Unexpected command");
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        tool: mock(() => {}),
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      newTool(pi);

      const result = await toolExecute?.({
        path: "docs/file.md",
        interactive: true,
        confirmed: false,
      });

      expect(result.created).toBe(true);
      expect(execMock).toHaveBeenCalled();
    });

    it("rootline-set allows update without confirmation in interactive mode", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
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
                  path: "file.md",
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
        throw new Error("Unexpected command");
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      setTool(pi);
      const mockContext = { cwd: "/home/shared/rootline" };

      const result = await toolExecute?.(
        {
          path: "docs/file.md",
          fields: [{ field: "estado", value: "pending" }],
          interactive: true,
          confirmed: false,
        },
        mockContext
      );

      expect(result.updated).toBe(true);
      expect(execMock).toHaveBeenCalled();
    });
  });

  describe("validation failure surfaced to model", () => {
    it("rootline-new surfaces validation errors in result.validation.errors", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        if (args[0] === "new") {
          return {
            stdout: "Created /path/to/file.md\n",
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
                  path: "/path/to/file.md",
                  valid: false,
                  errors: [
                    {
                      field: "title",
                      rule: "required",
                      message: "required field missing: title",
                    },
                    {
                      field: "estado",
                      rule: "enum",
                      message: "value not in allowed enum: ['pending', 'active', 'done']",
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
        throw new Error("Unexpected command");
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        tool: mock(() => {}),
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      newTool(pi);

      const result = await toolExecute?.({
        path: "docs/file.md",
        confirmed: true,
      });

      // Verify that created is still true (file was created)
      expect(result.created).toBe(true);

      // Verify that validation result contains the errors
      expect(result.validation).toBeDefined();
      expect(result.validation?.ok).toBe(false);
      expect(result.validation?.errors).toBeDefined();
      expect(Array.isArray(result.validation?.errors)).toBe(true);
      expect(result.validation?.errors?.length).toBeGreaterThan(0);

      // Verify the model can read individual error details
      const errors = result.validation?.errors as any[];
      if (errors && errors.length > 0) {
        const firstError = errors[0];
        expect(typeof firstError).toBe("object");
        expect(firstError.field || firstError.message).toBeDefined();
      }
    });

    it("rootline-set surfaces validation errors in result.validation.errors", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
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
                  path: "file.md",
                  valid: false,
                  errors: [
                    {
                      field: "estado",
                      rule: "enum",
                      message: "value not in allowed enum: ['pending', 'active', 'done']",
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
        throw new Error("Unexpected command");
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      setTool(pi);
      const mockContext = { cwd: "/home/shared/rootline" };

      const result = await toolExecute?.(
        {
          path: "docs/file.md",
          fields: [{ field: "estado", value: "invalid" }],
          interactive: true,
        },
        mockContext
      );

      // Verify mutation failed due to validation
      expect(result.updated).toBe(false);
      expect(result.error).toBeDefined();

      // Verify that validation result contains the errors
      expect(result.validation).toBeDefined();
      expect(result.validation?.ok).toBe(false);
      expect(result.validation?.errors).toBeDefined();
      expect(Array.isArray(result.validation?.errors)).toBe(true);
      expect(result.validation?.errors?.length).toBeGreaterThan(0);

      // Verify the model can read individual error details
      const errors = result.validation?.errors as any[];
      if (errors && errors.length > 0) {
        const firstError = errors[0];
        expect(typeof firstError).toBe("object");
        expect(firstError.field || firstError.message).toBeDefined();
      }
    });

    it("rootline-new includes validation error message accessible to model", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        if (args[0] === "new") {
          return {
            stdout: "Created /path/to/file.md\n",
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
                  path: "/path/to/file.md",
                  valid: false,
                  errors: [
                    {
                      message: "required field 'title' is missing",
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
        throw new Error("Unexpected command");
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        tool: mock(() => {}),
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      newTool(pi);

      const result = await toolExecute?.({
        path: "docs/file.md",
        confirmed: true,
      });

      expect(result.validation?.errors).toBeDefined();
      const errors = result.validation?.errors as any[];
      expect(errors.some((e) => e.message && e.message.includes("title"))).toBe(
        true
      );
    });
  });

  describe("success case end-to-end", () => {
    it("rootline-new creates file and returns validation ok", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        if (args[0] === "new") {
          return {
            stdout: "Created /home/shared/rootline/docs/file.md\n",
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
                  path: "/home/shared/rootline/docs/file.md",
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
        throw new Error("Unexpected command");
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        tool: mock(() => {}),
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      newTool(pi);

      const result = await toolExecute?.({
        path: "docs/file.md",
        confirmed: true,
      });

      expect(result.created).toBe(true);
      expect(result.path).toBeTruthy();
      expect(result.validation?.ok).toBe(true);
      expect(result.validation?.summary).toBeDefined();
      if (result.validation?.ok) {
        expect(result.validation.summary.valid).toBe(1);
        expect(result.validation.summary.invalid).toBe(0);
      }
    });

    it("rootline-set updates field and returns validation ok", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        if (args[0] === "set") {
          return {
            stdout: 'set estado = "active"\n',
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
                  path: "file.md",
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
        throw new Error("Unexpected command");
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      setTool(pi);
      const mockContext = { cwd: "/home/shared/rootline" };

      const result = await toolExecute?.(
        {
          path: "docs/file.md",
          fields: [{ field: "estado", value: "active" }],
          interactive: true,
        },
        mockContext
      );

      expect(result.updated).toBe(true);
      expect(result.fields_set).toContain("estado=active");
      expect(result.validation?.ok).toBe(true);
      expect(result.validation?.summary).toBeDefined();
    });

    it("chain: create file then update its field", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        if (args[0] === "new") {
          return {
            stdout: "Created /home/shared/rootline/docs/file.md\n",
            stderr: "",
            code: 0,
          };
        } else if (args[0] === "set") {
          return {
            stdout: 'set estado = "active"\n',
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
                  path: "/home/shared/rootline/docs/file.md",
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
        throw new Error("Unexpected command");
      });

      let newExecute: Function | null = null;
      let setExecute: Function | null = null;

      const pi = {
        exec: execMock,
        tool: mock(() => {}),
        registerTool: (def: any) => {
          if (def.name === "rootline-new") {
            newExecute = def.execute;
          } else if (def.name === "rootline-set") {
            setExecute = def.execute;
          }
        },
      } as unknown as ExtensionAPI;

      newTool(pi);
      setTool(pi);

      // Step 1: Create file
      const createResult = await newExecute?.({
        path: "docs/file.md",
        confirmed: true,
      });

      expect(createResult.created).toBe(true);
      expect(createResult.validation?.ok).toBe(true);

      // Step 2: Update field
      const mockContext = { cwd: "/home/shared/rootline" };
      const updateResult = await setExecute?.(
        {
          path: "docs/file.md",
          fields: [{ field: "estado", value: "active" }],
          interactive: true,
        },
        mockContext
      );

      expect(updateResult.updated).toBe(true);
      expect(updateResult.validation?.ok).toBe(true);
    });
  });

  describe("binary not found case", () => {
    it("rootline-new handles ENOENT gracefully", async () => {
      const execMock = mock(async () => {
        const err = new Error("not found");
        (err as any).code = "ENOENT";
        throw err;
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        tool: mock(() => {}),
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      newTool(pi);

      const result = await toolExecute?.({
        path: "docs/file.md",
        confirmed: true,
      });

      expect(result.created).toBe(false);
      expect(result.error).toBeDefined();
      expect(result.error).toContain("Failed to execute");
    });

    it("rootline-set handles ENOENT gracefully", async () => {
      const execMock = mock(async () => {
        const err = new Error("not found");
        (err as any).code = "ENOENT";
        throw err;
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      setTool(pi);
      const mockContext = { cwd: "/home/shared/rootline" };

      const result = await toolExecute?.(
        {
          path: "docs/file.md",
          fields: [{ field: "estado", value: "pending" }],
          interactive: true,
        },
        mockContext
      );

      expect(result.updated).toBe(false);
      expect(result.error).toBeDefined();
      expect(result.error).toContain("Failed to execute");
    });

    it("rootline-new does not crash on generic exec error", async () => {
      const execMock = mock(async () => {
        throw new Error("Generic execution error");
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        tool: mock(() => {}),
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      newTool(pi);

      const result = await toolExecute?.({
        path: "docs/file.md",
        confirmed: true,
      });

      expect(result.created).toBe(false);
      expect(result.error).toBeDefined();
      expect(() => result.error).not.toThrow();
    });

    it("rootline-set does not crash on generic exec error", async () => {
      const execMock = mock(async () => {
        throw new Error("Generic execution error");
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      setTool(pi);
      const mockContext = { cwd: "/home/shared/rootline" };

      const result = await toolExecute?.(
        {
          path: "docs/file.md",
          fields: [{ field: "estado", value: "pending" }],
          interactive: true,
        },
        mockContext
      );

      expect(result.updated).toBe(false);
      expect(result.error).toBeDefined();
      expect(() => result.error).not.toThrow();
    });
  });
});
