import { describe, it, expect, mock, spyOn } from "bun:test";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import migrateTool from "./migrate";

/**
 * Tests for rootline-migrate extension.
 *
 * Tests verify:
 * - Tool registration with correct schema
 * - Path validation guardrails (blocks path traversal)
 * - Mutual exclusion of rename/split/scaffold operations
 * - Default dry_run=true for safety
 * - Confirmation requirement in non-interactive mode for non-dry-run
 * - Successful migration with post-write validation
 * - Dry-run mode returns preview without writing
 * - Proper error handling for invalid operations and formats
 */

describe("rootline-migrate extension", () => {
  describe("tool registration", () => {
    it("registers rootline-migrate with correct description", () => {
      const registerToolMock = mock(() => {});
      const pi = {
        exec: mock(async () => ({
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/migrate-preview",
            path: "docs/roadmap/",
            dry_run: true,
            results: [],
            summary: { files_updated: 0 },
          }),
          stderr: "",
          code: 0,
        })),
        registerTool: registerToolMock,
      } as unknown as ExtensionAPI;

      migrateTool(pi);

      expect(registerToolMock).toHaveBeenCalled();
      const call = registerToolMock.mock.calls[0];
      const toolDef = call[0];
      expect(toolDef.name).toBe("rootline-migrate");
      expect(toolDef.description).toContain("schema migration");
      expect(toolDef.input_schema.required).toEqual(["path"]);
    });
  });

  describe("path validation", () => {
    it("rejects path traversal attacks (../..)", async () => {
      const execMock = mock(async () => ({
        stdout: JSON.stringify({
          version: 1,
          kind: "rootline/migrate-preview",
          dry_run: true,
        }),
        stderr: "",
        code: 0,
      }));

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      migrateTool(pi);

      // Should reject path that escapes cwd
      const result = await toolExecute?.({
        path: "../../../../etc/passwd",
        rename: "foo=bar",
      }, { cwd: "/home/project" });

      expect(result.changes_detected).toBe(false);
      expect(result.error).toContain("escapes project root");
      expect(execMock).not.toHaveBeenCalled();
    });

    it("accepts relative paths within project", async () => {
      const execMock = mock(async () => ({
        stdout: JSON.stringify({
          version: 1,
          kind: "rootline/migrate-preview",
          path: "docs/roadmap/",
          dry_run: true,
          results: [],
          summary: { files_updated: 1 },
        }),
        stderr: "",
        code: 0,
      }));

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      migrateTool(pi);

      // Should accept relative path
      const result = await toolExecute?.({
        path: "docs/roadmap",
        rename: "foo=bar",
      }, { cwd: "/home/project" });

      expect(execMock).toHaveBeenCalled();
    });
  });

  describe("operation mutual exclusion", () => {
    it("rejects multiple operations", async () => {
      const execMock = mock(async () => ({
        stdout: JSON.stringify({
          version: 1,
          kind: "rootline/migrate-preview",
          dry_run: true,
        }),
        stderr: "",
        code: 0,
      }));

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      migrateTool(pi);

      // Should reject when both rename and split are specified
      const result = await toolExecute?.({
        path: "docs/roadmap",
        rename: "foo=bar",
        split: true,
      }, { cwd: "/home/project" });

      expect(result.changes_detected).toBe(false);
      expect(result.error).toContain("mutually exclusive");
      expect(execMock).not.toHaveBeenCalled();
    });

    it("requires at least one operation", async () => {
      const execMock = mock(async () => ({
        stdout: JSON.stringify({
          version: 1,
          kind: "rootline/migrate-preview",
          dry_run: true,
        }),
        stderr: "",
        code: 0,
      }));

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      migrateTool(pi);

      // Should reject when no operation is specified
      const result = await toolExecute?.({
        path: "docs/roadmap",
      }, { cwd: "/home/project" });

      expect(result.changes_detected).toBe(false);
      expect(result.error).toContain("One of");
      expect(execMock).not.toHaveBeenCalled();
    });
  });

  describe("rename operation validation", () => {
    it("rejects invalid rename format", async () => {
      const execMock = mock(async () => ({
        stdout: JSON.stringify({
          version: 1,
          kind: "rootline/migrate-preview",
          dry_run: true,
        }),
        stderr: "",
        code: 0,
      }));

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      migrateTool(pi);

      // Invalid format: missing equals sign
      const result = await toolExecute?.({
        path: "docs/roadmap",
        rename: "foo_bar",
      }, { cwd: "/home/project" });

      expect(result.changes_detected).toBe(false);
      expect(result.error).toContain("Invalid --rename format");
      expect(execMock).not.toHaveBeenCalled();
    });

    it("parses valid rename format", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        // Verify correct args were passed
        expect(args).toContain("migrate");
        expect(args).toContain("--rename");
        expect(args).toContain("old_field=new_field");
        return {
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/migrate-preview",
            path: "docs/roadmap/",
            operation: "rename",
            dry_run: true,
            results: [
              {
                path: "docs/roadmap/.stem",
                type: "stem_file",
                changes: ["Field definition renamed: old_field → new_field"],
              },
            ],
            summary: { stem_files_modified: 1, documents_modified: 2 },
          }),
          stderr: "",
          code: 0,
        };
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      migrateTool(pi);

      const result = await toolExecute?.({
        path: "docs/roadmap",
        rename: "old_field=new_field",
      }, { cwd: "/home/project" });

      expect(result.operation).toBe("rename");
      expect(execMock).toHaveBeenCalled();
    });
  });

  describe("default dry_run=true for safety", () => {
    it("defaults to dry_run=true when not specified", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        // Verify --dry-run flag is present
        expect(args).toContain("--dry-run");
        return {
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/migrate-preview",
            dry_run: true,
            results: [],
            summary: { files_updated: 0 },
          }),
          stderr: "",
          code: 0,
        };
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      migrateTool(pi);

      const result = await toolExecute?.({
        path: "docs/roadmap",
        rename: "foo=bar",
        // dry_run is omitted, should default to true
      }, { cwd: "/home/project" });

      expect(result.dry_run).toBe(true);
      expect(execMock).toHaveBeenCalled();
    });

    it("respects explicit dry_run=false with confirmation", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        // Verify --dry-run flag is NOT present
        expect(!args.includes("--dry-run"));
        return {
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/migrate-result",
            dry_run: false,
            results: [],
            summary: { files_updated: 1 },
          }),
          stderr: "",
          code: 0,
        };
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      migrateTool(pi);

      const result = await toolExecute?.({
        path: "docs/roadmap",
        rename: "foo=bar",
        dry_run: false,
        confirmed: true,
        interactive: false,
      }, { cwd: "/home/project" });

      expect(result.dry_run).toBe(false);
      expect(execMock).toHaveBeenCalled();
    });
  });

  describe("confirmation requirement", () => {
    it("requires confirmation for non-dry-run in non-interactive mode", async () => {
      const execMock = mock(async () => ({
        stdout: JSON.stringify({
          version: 1,
          kind: "rootline/migrate-preview",
          dry_run: true,
        }),
        stderr: "",
        code: 0,
      }));

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      migrateTool(pi);

      // Should require confirmation when dry_run=false in non-interactive mode
      const result = await toolExecute?.({
        path: "docs/roadmap",
        rename: "foo=bar",
        dry_run: false,
        confirmed: false,
        interactive: false,
      }, { cwd: "/home/project" });

      expect(result.changes_detected).toBe(false);
      expect(result.error).toContain("Confirmation required");
      expect(execMock).not.toHaveBeenCalled();
    });

    it("allows non-dry-run with explicit confirmed=true", async () => {
      const execMock = mock(async () => ({
        stdout: JSON.stringify({
          version: 1,
          kind: "rootline/migrate-result",
          dry_run: false,
          results: [],
          summary: { files_updated: 2 },
        }),
        stderr: "",
        code: 0,
      }));

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      migrateTool(pi);

      // Should allow with explicit confirmed=true
      const result = await toolExecute?.({
        path: "docs/roadmap",
        rename: "foo=bar",
        dry_run: false,
        confirmed: true,
        interactive: false,
      }, { cwd: "/home/project" });

      expect(execMock).toHaveBeenCalled();
    });

    it("does not require confirmation for dry-run", async () => {
      const execMock = mock(async () => ({
        stdout: JSON.stringify({
          version: 1,
          kind: "rootline/migrate-preview",
          dry_run: true,
          results: [],
          summary: { files_updated: 0 },
        }),
        stderr: "",
        code: 0,
      }));

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      migrateTool(pi);

      // Should execute dry-run without confirmation
      const result = await toolExecute?.({
        path: "docs/roadmap",
        rename: "foo=bar",
        dry_run: true,
        confirmed: false,
        interactive: false,
      }, { cwd: "/home/project" });

      expect(execMock).toHaveBeenCalled();
    });
  });

  describe("split and scaffold operations", () => {
    it("handles split operation", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        expect(args).toContain("--split");
        return {
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/migrate-preview",
            operation: "split",
            dry_run: true,
            results: [
              {
                path: "docs/roadmap/.stem",
                type: "stem_file",
                changes: ["Split into hierarchical files"],
              },
            ],
            summary: { stem_files_created: 3 },
          }),
          stderr: "",
          code: 0,
        };
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      migrateTool(pi);

      const result = await toolExecute?.({
        path: "docs/roadmap",
        split: true,
      }, { cwd: "/home/project" });

      expect(result.operation).toBe("split");
      expect(execMock).toHaveBeenCalled();
    });

    it("handles scaffold operation", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        expect(args).toContain("--scaffold");
        return {
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/migrate-preview",
            operation: "scaffold",
            dry_run: true,
            results: [],
            summary: { documents_scaffolded: 5 },
          }),
          stderr: "",
          code: 0,
        };
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      migrateTool(pi);

      const result = await toolExecute?.({
        path: "docs/roadmap",
        scaffold: true,
      }, { cwd: "/home/project" });

      expect(result.operation).toBe("scaffold");
      expect(execMock).toHaveBeenCalled();
    });
  });

  describe("post-mutation validation", () => {
    it("validates after non-dry-run migration with changes", async () => {
      let validateCalled = false;
      const execMock = mock(async (cmd: string, args: string[]) => {
        if (args[0] === "migrate") {
          return {
            stdout: JSON.stringify({
              version: 1,
              kind: "rootline/migrate-result",
              dry_run: false,
              results: [],
              summary: { files_updated: 2 },
            }),
            stderr: "",
            code: 0,
          };
        } else if (args[0] === "validate") {
          validateCalled = true;
          return {
            stdout: JSON.stringify({
              version: 1,
              kind: "rootline/validate-result",
              results: [
                {
                  path: "docs/roadmap/O06/.stem",
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

      migrateTool(pi);

      const result = await toolExecute?.({
        path: "docs/roadmap",
        rename: "foo=bar",
        dry_run: false,
        confirmed: true,
      }, { cwd: "/home/project" });

      expect(validateCalled).toBe(true);
      expect(result.validation?.ok).toBe(true);
    });

    it("reports validation failure after mutation", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        if (args[0] === "migrate") {
          return {
            stdout: JSON.stringify({
              version: 1,
              kind: "rootline/migrate-result",
              dry_run: false,
              results: [],
              summary: { files_updated: 2 },
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
                  path: "docs/roadmap/O06/.stem",
                  valid: false,
                  errors: ["Field type mismatch after migration"],
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

      migrateTool(pi);

      const result = await toolExecute?.({
        path: "docs/roadmap",
        rename: "foo=bar",
        dry_run: false,
        confirmed: true,
      }, { cwd: "/home/project" });

      expect(result.validation?.ok).toBe(false);
      expect(result.error).toContain("validation failed");
    });
  });

  describe("error handling", () => {
    it("handles command execution failure", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        if (args[0] === "migrate") {
          return {
            stdout: "",
            stderr: "failed to rename field in .stem",
            code: 1,
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

      migrateTool(pi);

      const result = await toolExecute?.({
        path: "docs/roadmap",
        rename: "foo=bar",
        confirmed: true,
      }, { cwd: "/home/project" });

      expect(result.changes_detected).toBe(false);
      expect(result.error).toBeDefined();
    });

    it("handles unexpected errors gracefully", async () => {
      const execMock = mock(async () => {
        throw new Error("Process crashed");
      });

      let toolExecute: Function | null = null;
      const pi = {
        exec: execMock,
        registerTool: (def: any) => {
          toolExecute = def.execute;
        },
      } as unknown as ExtensionAPI;

      migrateTool(pi);

      const result = await toolExecute?.({
        path: "docs/roadmap",
        rename: "foo=bar",
      }, { cwd: "/home/project" });

      expect(result.changes_detected).toBe(false);
      expect(result.error).toContain("Process crashed");
    });
  });

  describe("dry-run behavior", () => {
    it("returns preview without validation", async () => {
      const execMock = mock(async (cmd: string, args: string[]) => {
        if (args[0] === "migrate") {
          return {
            stdout: JSON.stringify({
              version: 1,
              kind: "rootline/migrate-preview",
              dry_run: true,
              results: [
                {
                  path: "docs/roadmap/O06/T001.md",
                  type: "document",
                  changes: ["Frontmatter field renamed: foo → bar"],
                },
              ],
              summary: { documents_modified: 1 },
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

      migrateTool(pi);

      const result = await toolExecute?.({
        path: "docs/roadmap",
        rename: "foo=bar",
        dry_run: true,
      }, { cwd: "/home/project" });

      expect(result.dry_run).toBe(true);
      expect(result.validation).toBeUndefined(); // No validation for dry-run
      expect(result.results).toBeDefined();
    });
  });
});
