import { describe, it, expect, mock, spyOn } from "bun:test";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import fixTool from "./fix";

/**
 * Tests for rootline-fix extension.
 *
 * Tests verify:
 * - Tool registration with correct schema
 * - Path validation guardrails (blocks path traversal)
 * - Confirmation requirement in non-interactive mode for non-dry-run
 * - Dry-run mode (safe by default) returns preview without writing
 * - Apply mode with confirmation writes and validates
 * - Post-validation runs after fixes
 * - Proper error handling for invalid paths and failed operations
 * - Optional --where filter is passed through correctly
 */

describe("rootline-fix extension", () => {
  describe("tool registration", () => {
    it("registers rootline-fix with correct description", () => {
      const registerToolMock = mock(() => {});
      const pi = {
        exec: mock(async () => ({
          stdout: "",
          stderr: "",
          code: 0,
        })),
        tool: mock(() => {}),
        registerTool: registerToolMock,
      } as unknown as ExtensionAPI;

      fixTool(pi);

      expect(registerToolMock).toHaveBeenCalled();
      const call = registerToolMock.mock.calls[0];
      const toolDef = call[0];
      expect(toolDef.name).toBe("rootline-fix");
      expect(toolDef.description).toContain("Bulk fix validation errors");
      expect(toolDef.input_schema.required).toEqual(["path"]);
      expect(toolDef.input_schema.properties.dry_run).toBeDefined();
      expect(toolDef.input_schema.properties.confirmed).toBeDefined();
    });

    it("declares default dry_run as true in description", () => {
      const registerToolMock = mock(() => {});
      const pi = {
        exec: mock(async () => ({
          stdout: "",
          stderr: "",
          code: 0,
        })),
        tool: mock(() => {}),
        registerTool: registerToolMock,
      } as unknown as ExtensionAPI;

      fixTool(pi);

      const toolDef = registerToolMock.mock.calls[0][0];
      expect(toolDef.input_schema.properties.dry_run.description).toContain("default: true");
    });
  });

  describe("path validation", () => {
    it("rejects path traversal attacks (../..)", async () => {
      const execMock = mock(async () => ({
        stdout: JSON.stringify({ version: 1, kind: "rootline/fix-batch", results: [], summary: { total: 0, fixed: 0, skipped: 0 } }),
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

      fixTool(pi);

      // Should reject path that escapes cwd
      const result = await toolExecute?.({
        path: "../../../../etc/passwd",
      });

      expect(result.success).toBe(false);
      expect(result.error).toContain("escapes project root");
      expect(execMock).not.toHaveBeenCalled();
    });

    it("accepts relative paths within project", async () => {
      const execMock = mock(async () => ({
        stdout: JSON.stringify({
          version: 1,
          kind: "rootline/fix-batch",
          results: [],
          summary: { total: 10, fixed: 0, skipped: 10 },
        }),
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

      fixTool(pi);

      // Should accept relative path
      const result = await toolExecute?.({
        path: "docs/roadmap/",
        dry_run: true,
      });

      expect(execMock).toHaveBeenCalled();
      expect(result.success).toBe(true);
    });
  });

  describe("confirmation requirement", () => {
    it("requires confirmation for non-dry-run in non-interactive mode", async () => {
      const execMock = mock(async () => ({
        stdout: JSON.stringify({ version: 1, kind: "rootline/fix-batch", results: [], summary: { total: 0, fixed: 0, skipped: 0 } }),
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

      fixTool(pi);

      // Should reject non-dry-run without confirmation in non-interactive mode
      const result = await toolExecute?.({
        path: "docs/roadmap/",
        dry_run: false,
        interactive: false,
        confirmed: false,
      });

      expect(result.success).toBe(false);
      expect(result.error).toContain("Confirmation required");
      expect(execMock).not.toHaveBeenCalled();
    });

    it("allows non-dry-run with explicit confirmed flag", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        if (args[0] === "fix") {
          return {
            stdout: JSON.stringify({
              version: 1,
              kind: "rootline/fix-batch",
              results: [
                {
                  path: "docs/roadmap/O06/T001.md",
                  fixed: true,
                  fields_added: 1,
                  values_corrected: 0,
                  changes: ["add dependency"],
                },
              ],
              summary: { total: 1, fixed: 1, skipped: 0 },
            }),
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
                  path: "docs/roadmap/O06/T001.md",
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

      fixTool(pi);

      // confirmed flag should bypass requirement
      const result = await toolExecute?.({
        path: "docs/roadmap/",
        dry_run: false,
        interactive: false,
        confirmed: true,
      });

      expect(execMock).toHaveBeenCalled();
      expect(result.success).toBe(true);
    });

    it("allows non-dry-run in interactive mode without explicit confirmation", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        if (args[0] === "fix") {
          return {
            stdout: JSON.stringify({
              version: 1,
              kind: "rootline/fix-batch",
              results: [],
              summary: { total: 5, fixed: 1, skipped: 4 },
            }),
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
                  path: "docs/roadmap/O06/T001.md",
                  valid: true,
                  errors: [],
                },
              ],
              summary: {
                total: 5,
                valid: 5,
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

      fixTool(pi);

      // In interactive mode, Pi handles confirmation
      const result = await toolExecute?.({
        path: "docs/roadmap/",
        dry_run: false,
        interactive: true,
        confirmed: false,
      });

      expect(execMock).toHaveBeenCalled();
      expect(result.success).toBe(true);
    });
  });

  describe("dry-run mode (safe by default)", () => {
    it("defaults to dry-run when not specified", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        // Verify --dry-run is passed by default
        expect(args).toContain("--dry-run");
        return {
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/fix-batch",
            results: [
              {
                path: "docs/roadmap/O06/T001.md",
                fixed: true,
                fields_added: 1,
                values_corrected: 0,
                changes: ["would add dependency"],
              },
            ],
            summary: { total: 10, fixed: 1, skipped: 9 },
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

      fixTool(pi);

      const result = await toolExecute?.({
        path: "docs/roadmap/",
        // dry_run not specified — should default to true
      });

      expect(execMock).toHaveBeenCalled();
      expect(result.dry_run).toBe(true);
      expect(result.success).toBe(true);
    });

    it("returns preview without validation in dry-run mode", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        if (args[0] === "fix") {
          return {
            stdout: JSON.stringify({
              version: 1,
              kind: "rootline/fix-batch",
              results: [
                {
                  path: "docs/roadmap/O06/T001.md",
                  fixed: true,
                  fields_added: 1,
                  values_corrected: 1,
                  changes: ["add dependency", "correct estado"],
                },
              ],
              summary: { total: 10, fixed: 2, skipped: 8 },
              schema_suggestions: 0,
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

      fixTool(pi);

      const result = await toolExecute?.({
        path: "docs/roadmap/",
        dry_run: true,
      });

      expect(result.success).toBe(true);
      expect(result.dry_run).toBe(true);
      expect(result.summary.fixed).toBe(2);
      // Validation should not be called in dry-run mode
      const callCount = execMock.mock.calls.length;
      expect(callCount).toBe(1); // Only fix, not validate
    });
  });

  describe("apply mode with validation", () => {
    it("validates after applying fixes in non-dry-run mode", async () => {
      let callCount = 0;
      const execMock = mock(async (cmd: string, args: string[]) => {
        callCount++;
        if (args[0] === "fix") {
          return {
            stdout: JSON.stringify({
              version: 1,
              kind: "rootline/fix-batch",
              results: [
                {
                  path: "docs/roadmap/O06/T001.md",
                  fixed: true,
                  fields_added: 1,
                  values_corrected: 0,
                  changes: ["add dependency"],
                },
              ],
              summary: { total: 10, fixed: 1, skipped: 9 },
            }),
            stderr: "",
            code: 0,
          };
        } else if (args[0] === "validate") {
          // Return validation results for all 10 files
          const validationResults = Array.from({ length: 10 }, (_, i) => ({
            path: `docs/roadmap/O06/T${i.toString().padStart(3, "0")}.md`,
            valid: true,
            errors: [],
          }));
          return {
            stdout: JSON.stringify({
              version: 1,
              kind: "rootline/validate-result",
              results: validationResults,
              summary: {
                total: 10,
                valid: 10,
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

      fixTool(pi);

      const result = await toolExecute?.({
        path: "docs/roadmap/",
        dry_run: false,
        confirmed: true,
      });

      expect(result.success).toBe(true);
      expect(result.validation).toBeDefined();
      expect(result.validation?.total_files).toBe(10);
      expect(result.validation?.valid_count).toBe(10);
      // Should have called both fix and validate
      expect(execMock.mock.calls.length).toBe(2);
    });

    it("includes validation result in output", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        if (args[0] === "fix") {
          return {
            stdout: JSON.stringify({
              version: 1,
              kind: "rootline/fix-batch",
              results: [],
              summary: { total: 10, fixed: 0, skipped: 10 },
            }),
            stderr: "",
            code: 0,
          };
        } else if (args[0] === "validate") {
          // Create 10 validation results: 9 valid, 1 invalid
          const validationResults = Array.from({ length: 9 }, (_, i) => ({
            path: `docs/roadmap/O06/T${i.toString().padStart(3, "0")}.md`,
            valid: true,
            errors: [],
          }));
          validationResults.push({
            path: "docs/roadmap/O06/T010.md",
            valid: false,
            errors: ["missing title"],
          });
          return {
            stdout: JSON.stringify({
              version: 1,
              kind: "rootline/validate-result",
              results: validationResults,
              summary: { total: 10, valid: 9, invalid: 1 },
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

      fixTool(pi);

      const result = await toolExecute?.({
        path: "docs/roadmap/",
        dry_run: false,
        confirmed: true,
      });

      expect(result.validation?.total_files).toBe(10);
      expect(result.validation?.valid_count).toBe(9);
      expect(result.validation?.error_count).toBe(1);
    });
  });

  describe("schema suggestions", () => {
    it("includes schema_suggestions count in output", async () => {
      const execMock = mock(async () => ({
        stdout: JSON.stringify({
          version: 1,
          kind: "rootline/fix-batch",
          results: [
            {
              path: "docs/roadmap/O06/T001.md",
              fixed: true,
              fields_added: 0,
              values_corrected: 1,
              changes: ["correct estado"],
            },
          ],
          summary: { total: 10, fixed: 1, skipped: 9 },
          schema_suggestions: 2,
        }),
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

      fixTool(pi);

      const result = await toolExecute?.({
        path: "docs/roadmap/",
        dry_run: true,
      });

      expect(result.schema_suggestions).toBe(2);
    });
  });

  describe("where filter", () => {
    it("passes --where filter to rootline command", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        if (args[0] === "fix") {
          // Verify --where is passed
          const whereIdx = args.indexOf("--where");
          expect(whereIdx).toBeGreaterThanOrEqual(0);
          expect(args[whereIdx + 1]).toBe("tipo=Epic");
        }
        return {
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/fix-batch",
            results: [
              {
                path: "docs/roadmap/O06/E001.md",
                fixed: true,
                fields_added: 1,
                values_corrected: 0,
                changes: ["add priority"],
              },
            ],
            summary: { total: 5, fixed: 1, skipped: 4 },
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

      fixTool(pi);

      const result = await toolExecute?.({
        path: "docs/roadmap/",
        dry_run: true,
        where: "tipo=Epic",
      });

      expect(execMock).toHaveBeenCalled();
      expect(result.success).toBe(true);
    });
  });

  describe("error handling", () => {
    it("handles command execution failure", async () => {
      const execMock = mock(async () => ({
        stdout: "",
        stderr: "path not found: /nonexistent",
        code: 1,
      }));

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        tool: mock(() => {}),
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      fixTool(pi);

      const result = await toolExecute?.({
        path: "docs/roadmap/",
        dry_run: true,
      });

      expect(result.success).toBe(false);
      expect(result.error).toBeDefined();
      expect(result.error).toContain("rootline fix failed");
    });

    it("handles invalid JSON output", async () => {
      const execMock = mock(async () => ({
        stdout: "invalid json output",
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

      fixTool(pi);

      const result = await toolExecute?.({
        path: "docs/roadmap/",
        dry_run: true,
      });

      expect(result.success).toBe(false);
      expect(result.error).toBeDefined();
    });

    it("returns error when path outside project root", async () => {
      const execMock = mock(async () => ({
        stdout: "",
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

      fixTool(pi);

      const result = await toolExecute?.({
        path: "/etc/passwd",
      });

      expect(result.success).toBe(false);
      expect(result.error).toContain("Path validation failed");
      expect(execMock).not.toHaveBeenCalled();
    });
  });

  describe("result structure", () => {
    it("returns properly structured fix result", async () => {
      const execMock = mock(async () => ({
        stdout: JSON.stringify({
          version: 1,
          kind: "rootline/fix-batch",
          results: [
            {
              path: "docs/roadmap/O06/T001.md",
              fixed: true,
              fields_added: 2,
              values_corrected: 1,
              changes: ["add type", "add priority", "correct estado"],
            },
            {
              path: "docs/roadmap/O06/T002.md",
              fixed: false,
              fields_added: 0,
              values_corrected: 0,
              changes: [],
            },
          ],
          summary: { total: 10, fixed: 1, skipped: 9 },
          schema_suggestions: 0,
        }),
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

      fixTool(pi);

      const result = await toolExecute?.({
        path: "docs/roadmap/",
        dry_run: true,
      });

      // Verify structure
      expect(result.success).toBe(true);
      expect(result.dry_run).toBe(true);
      expect(result.path).toBeTruthy();
      expect(result.summary).toBeDefined();
      expect(result.summary.total).toBe(10);
      expect(result.summary.fixed).toBe(1);
      expect(result.summary.skipped).toBe(9);
      expect(result.results).toHaveLength(2);
      expect(result.results[0].fixed).toBe(true);
      expect(result.results[0].fields_added).toBe(2);
      expect(result.results[0].values_corrected).toBe(1);
    });
  });
});
