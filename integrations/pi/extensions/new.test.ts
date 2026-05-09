import { describe, it, expect, mock, spyOn } from "bun:test";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import newTool from "./new";

/**
 * Tests for rootline-new extension.
 *
 * Tests verify:
 * - Tool registration with correct schema
 * - Path validation guardrails (blocks path traversal)
 * - Confirmation requirement in non-interactive mode
 * - Successful file creation with post-write validation
 * - Dry-run mode returns content preview
 * - Failure path: validation errors, missing schema
 */

describe("rootline-new extension", () => {
  describe("tool registration", () => {
    it("registers rootline-new with correct description", () => {
      const registerToolMock = mock(() => {});
      const pi = {
        exec: mock(async () => ({
          stdout: "Created /path/to/file.md\n",
          stderr: "",
          code: 0,
        })),
        tool: mock(() => {}),
        registerTool: registerToolMock,
      } as unknown as ExtensionAPI;

      newTool(pi);

      expect(registerToolMock).toHaveBeenCalled();
      const call = registerToolMock.mock.calls[0];
      const toolDef = call[0];
      expect(toolDef.name).toBe("rootline-new");
      expect(toolDef.description).toContain("governed record");
      expect(toolDef.input_schema.required).toEqual(["path"]);
    });
  });

  describe("path validation", () => {
    it("rejects path traversal attacks (../..)", async () => {
      const execMock = mock(async () => ({
        stdout: "Created /path/to/file.md\n",
        stderr: "",
        code: 0,
      }));

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        tool: mock(() => {}),
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      newTool(pi);

      // Should reject path that escapes cwd
      const result = await toolExecute?.({
        path: "../../../../etc/passwd",
      });

      expect(result.created).toBe(false);
      expect(result.error).toContain("escapes project root");
      expect(execMock).not.toHaveBeenCalled();
    });

    it("accepts relative paths within project", async () => {
      const execMock = mock(async () => ({
        stdout: "Created /home/project/docs/file.md\n",
        stderr: "",
        code: 0,
      }));

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        tool: mock(() => {}),
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      newTool(pi);

      // Should accept relative path
      const result = await toolExecute?.({
        path: "docs/file.md",
        confirmed: true,
      });

      expect(execMock).toHaveBeenCalled();
    });
  });

  describe("confirmation requirement", () => {
    it("requires confirmation in non-interactive mode", async () => {
      const execMock = mock(async () => ({
        stdout: "Created /path/to/file.md\n",
        stderr: "",
        code: 0,
      }));

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        tool: mock(() => {}),
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      newTool(pi);

      // Should reject without confirmation in non-interactive mode
      const result = await toolExecute?.({
        path: "docs/file.md",
        interactive: false,
        confirmed: false,
      });

      expect(result.created).toBe(false);
      expect(result.error).toContain("Confirmation required");
      expect(execMock).not.toHaveBeenCalled();
    });

    it("allows creation with force flag", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        expect(args).toContain("--force");
        return {
          stdout: "Created /path/to/file.md\n",
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

      newTool(pi);

      // force flag should bypass confirmation
      await toolExecute?.({
        path: "docs/file.md",
        force: true,
        interactive: false,
      });

      expect(execMock).toHaveBeenCalled();
    });

    it("allows creation with explicit confirmed flag", async () => {
      const execMock = mock(async () => ({
        stdout: "Created /path/to/file.md\n",
        stderr: "",
        code: 0,
      }));

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        tool: mock(() => {}),
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      newTool(pi);

      // confirmed flag should bypass requirement
      await toolExecute?.({
        path: "docs/file.md",
        confirmed: true,
        interactive: false,
      });

      expect(execMock).toHaveBeenCalled();
    });
  });

  describe("dry-run mode", () => {
    it("returns content preview without writing", async () => {
      const previewContent = "---\ntitle: Test File\n---\n# Test File\n";
      const execMock = mock(async (cmd: string, args: string[]) => {
        if (args.includes("--dry-run")) {
          return {
            stdout: previewContent,
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
        dry_run: true,
        confirmed: true, // Required in non-interactive mode
      });

      expect(result.created).toBe(false);
      expect(result.content_preview).toBe(previewContent);
      expect(result.content_preview).toContain("title: Test File");
      expect(execMock).toHaveBeenCalled();
    });
  });

  describe("successful creation", () => {
    it("creates file and returns validation result", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        if (args[0] === "new") {
          return {
            stdout: "Created /path/to/file.md\n",
            stderr: "",
            code: 0,
          };
        } else if (args[0] === "validate") {
          // For validation, return valid JSON as the runner expects
          return {
            stdout: JSON.stringify({
              version: 1,
              kind: "rootline/validate-result",
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
        confirmed: true,
      });

      expect(result.created).toBe(true);
      expect(result.path).toBeTruthy();
      expect(result.validation).toBeDefined();
      expect(result.validation?.ok).toBe(true);
    });
  });

  describe("error handling", () => {
    it("handles command execution failure", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        if (args[0] === "new") {
          return {
            stdout: "",
            stderr: "no .stem schema found for /path",
            code: 1,
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

      expect(result.created).toBe(false);
      expect(result.error).toBeDefined();
      expect(result.error).toContain("no .stem schema found");
    });

    it("handles validation failure after creation", async () => {
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
              kind: "rootline/validate-result",
              results: [
                {
                  path: "/path/to/file.md",
                  valid: false,
                  errors: ["required field missing: title"],
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

      expect(result.created).toBe(true);
      expect(result.validation?.ok).toBe(false);
    });
  });
});
