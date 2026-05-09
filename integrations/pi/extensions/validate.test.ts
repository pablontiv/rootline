import { describe, it, expect, mock } from "bun:test";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import validate from "./validate";

/**
 * Tests for rootline-validate extension.
 *
 * Tests verify:
 * - Tool registration with correct schema
 * - Command construction with filter expressions
 * - Success path: returns validation result summary
 * - Failure path: rootline not found, validation errors, invalid paths
 */

describe("rootline-validate extension", () => {
  describe("tool registration", () => {
    it("registers rootline-validate with correct input schema", () => {
      const registerToolMock = mock(() => {});
      const pi = {
        exec: mock(async () => ({
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/batch-validation-result",
            results: [],
            drift_warnings: [],
            summary: { total: 0, valid: 0, invalid: 0, errors_count: 0, warnings_count: 0, drift_warnings_count: 0 },
          }),
          stderr: "",
          code: 0,
        })),
        tool: mock(() => {}),
        registerTool: registerToolMock,
      } as unknown as ExtensionAPI;

      validate(pi);

      expect(registerToolMock).toHaveBeenCalled();
      const call = registerToolMock.mock.calls[0];
      const toolDef = call[0];
      expect(toolDef.name).toBe("rootline-validate");
      expect(toolDef.description).toContain("Validate documents");
      expect(toolDef.input_schema.properties.path).toBeDefined();
      expect(toolDef.input_schema.required).toContain("path");
    });
  });

  describe("command construction", () => {
    it("constructs validate command with default all=true", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        expect(cmd).toBe("rootline");
        expect(args).toContain("validate");
        expect(args).toContain("--all");
        expect(args).toContain("/docs/roadmap");
        expect(args).toContain("--output");
        expect(args).toContain("json");

        return {
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/batch-validation-result",
            results: [],
            drift_warnings: [],
            summary: { total: 0, valid: 0, invalid: 0, errors_count: 0, warnings_count: 0, drift_warnings_count: 0 },
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

      validate(pi);
      await toolExecute?.({ path: "/docs/roadmap" });

      expect(execMock).toHaveBeenCalled();
    });

    it("includes where filter when provided", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        expect(args).toContain("--where");
        expect(args).toContain("estado == 'Pending'");

        return {
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/batch-validation-result",
            results: [],
            drift_warnings: [],
            summary: { total: 0, valid: 0, invalid: 0, errors_count: 0, warnings_count: 0, drift_warnings_count: 0 },
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

      validate(pi);
      await toolExecute?.({ path: "/docs/roadmap", where: "estado == 'Pending'" });

      expect(execMock).toHaveBeenCalled();
    });

    it("respects all=false parameter", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        expect(args).not.toContain("--all");

        return {
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/batch-validation-result",
            results: [],
            drift_warnings: [],
            summary: { total: 0, valid: 0, invalid: 0, errors_count: 0, warnings_count: 0, drift_warnings_count: 0 },
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

      validate(pi);
      await toolExecute?.({ path: "/docs/roadmap", all: false });

      expect(execMock).toHaveBeenCalled();
    });
  });

  describe("success path", () => {
    it("returns valid result when all files pass validation", async () => {
      const execMock = mock(async () => ({
        stdout: JSON.stringify({
          version: 1,
          kind: "rootline/batch-validation-result",
          results: [
            {
              version: 1,
              kind: "rootline/validation-result",
              path: "docs/E01.md",
              valid: true,
              errors: [],
              warnings: [],
            },
          ],
          drift_warnings: [],
          summary: {
            total: 1,
            valid: 1,
            invalid: 0,
            errors_count: 0,
            warnings_count: 0,
            drift_warnings_count: 0,
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

      validate(pi);
      await toolExecute?.({ path: "/docs/roadmap" });

      expect(result).toBeDefined();
      expect(result.valid).toBe(true);
      expect(result.summary.invalid).toBe(0);
    });

    it("includes errors and warnings in result", async () => {
      const execMock = mock(async () => ({
        stdout: JSON.stringify({
          version: 1,
          kind: "rootline/batch-validation-result",
          results: [
            {
              version: 1,
              kind: "rootline/validation-result",
              path: "docs/E01.md",
              valid: false,
              errors: [
                {
                  path: "docs/E01.md",
                  field: "estado",
                  rule: "enum",
                  message: "invalid enum value",
                  severity: "error",
                },
              ],
              warnings: [],
            },
          ],
          drift_warnings: [],
          summary: {
            total: 1,
            valid: 0,
            invalid: 1,
            errors_count: 1,
            warnings_count: 0,
            drift_warnings_count: 0,
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

      validate(pi);
      await toolExecute?.({ path: "/docs/roadmap" });

      expect(result.valid).toBe(false);
      expect(result.results).toHaveLength(1);
      expect(result.results[0].errors).toHaveLength(1);
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

      validate(pi);
      await toolExecute?.({ path: "/docs/roadmap" });

      expect(result.error).toBeDefined();
      expect(result.code).toBe("not_found");
    });

    it("returns error with stderr when validation fails", async () => {
      const execMock = mock(async () => ({
        stdout: "",
        stderr: "validation failed: schema error",
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

      validate(pi);
      await toolExecute?.({ path: "/invalid/path" });

      expect(result.error).toBeDefined();
      expect(result.code).toBe("exec_failed");
    });

    it("returns error on timeout", async () => {
      const execMock = mock(async () => {
        throw new Error("ETIMEDOUT");
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

      validate(pi);
      await toolExecute?.({ path: "/docs/roadmap" });

      expect(result.error).toBeDefined();
      expect(result.code).toBe("timeout");
    });
  });
});
