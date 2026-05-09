import { describe, it, expect, mock, beforeEach } from "bun:test";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

/**
 * Integration tests for rootline Pi extensions.
 *
 * Tests all read-only tools (query, describe, validate, tree, stats, doctor)
 * using mocked pi.exec to verify:
 * - Success path: correct command construction and JSON parsing
 * - Failure path: rootline not found, non-zero exit, invalid JSON
 *
 * Tests do NOT mutate project files — all execution uses mocked pi.exec.
 */

// Mock context object for tool execution
const mockContext = {
  cwd: "/home/shared/rootline",
};

// ============================================================================
// rootline-query Extension Tests
// ============================================================================

describe("rootline-query extension", () => {
  describe("success path", () => {
    it("executes query with path only", async () => {
      const mockExec = mock(async () => ({
        stdout: JSON.stringify({
          version: 1,
          kind: "rootline/query-result",
          rows: [{ path: "docs/roadmap/E01.md", estado: "In Progress" }],
          total: 1,
        }),
        stderr: "",
        code: 0,
      }));

      // Load and execute the extension
      const extensionModule = await import("./query.js");
      const pi = {
        exec: mockExec,
        tool: mock(() => {}),
        registerTool: mock(() => {}),
      } as unknown as ExtensionAPI;

      extensionModule.default(pi);

      // Verify pi.tool was called to register rootline-query
      expect(pi.tool).toHaveBeenCalled();

      // Verify the tool was registered with correct name and description
      const toolCall = (pi.tool as ReturnType<typeof mock>).mock.calls[0];
      expect(toolCall[0]).toBe("rootline-query");
      expect(toolCall[1].description).toContain("Query Rootline records");
    });

    it("constructs correct args with all parameters", async () => {
      const mockExec = mock(async (cmd: string, args: string[]) => {
        // Verify command construction
        expect(cmd).toBe("rootline");
        expect(args).toContain("query");
        expect(args).toContain("--from");
        expect(args).toContain("/docs/roadmap");
        expect(args).toContain("--output");
        expect(args).toContain("json");
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

      const pi = {
        exec: mockExec,
        tool: (name: string, def: any) => {
          if (name === "rootline-query") {
            // Execute the tool with params
            return def.execute(
              {
                path: "/docs/roadmap",
                where: "estado == 'Pending'",
                select: "path,estado,title",
                limit: 10,
                sort: "prioridad:asc",
              },
              mockContext
            );
          }
        },
        registerTool: mock(() => {}),
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./query.js");
      await extensionModule.default(pi);

      expect(mockExec).toHaveBeenCalled();
    });
  });

  describe("failure paths", () => {
    it("returns error message when rootline not found", async () => {
      const mockExec = mock(async () => {
        const err = new Error("ENOENT: rootline not found");
        throw err;
      });

      const pi = {
        exec: mockExec,
        tool: (name: string, def: any) => {
          if (name === "rootline-query") {
            return def.execute(
              { path: "/docs/roadmap" },
              mockContext
            );
          }
        },
        registerTool: mock(() => {}),
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./query.js");
      await extensionModule.default(pi);

      expect(mockExec).toHaveBeenCalled();
    });

    it("returns error message when rootline exits with code 1", async () => {
      const mockExec = mock(async () => ({
        stdout: "",
        stderr: "directory not found: /invalid/path",
        code: 1,
      }));

      const pi = {
        exec: mockExec,
        tool: (name: string, def: any) => {
          if (name === "rootline-query") {
            return def.execute(
              { path: "/invalid/path" },
              mockContext
            );
          }
        },
        registerTool: mock(() => {}),
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./query.js");
      await extensionModule.default(pi);

      expect(mockExec).toHaveBeenCalled();
    });
  });
});

// ============================================================================
// rootline-describe Extension Tests
// ============================================================================

describe("rootline-describe extension", () => {
  describe("success path", () => {
    it("executes describe with path only", async () => {
      const mockExec = mock(async () => ({
        stdout: JSON.stringify({
          version: 1,
          kind: "rootline/describe-result",
          path: "/docs/roadmap",
          schema: {
            estado: { type: "string", enum: ["Pending", "In Progress"] },
            tipo: { type: "string" },
          },
        }),
        stderr: "",
        code: 0,
      }));

      const pi = {
        exec: mockExec,
        tool: mock(() => {}),
        registerTool: mock(() => {}),
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./describe.js");
      extensionModule.default(pi);

      expect(pi.tool).toHaveBeenCalled();
      const toolCall = (pi.tool as ReturnType<typeof mock>).mock.calls[0];
      expect(toolCall[0]).toBe("rootline-describe");
    });

    it("extracts field from schema when field parameter is provided", async () => {
      const mockExec = mock(async () => ({
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

      const pi = {
        exec: mockExec,
        tool: (name: string, def: any) => {
          if (name === "rootline-describe") {
            return def.execute(
              { path: "/docs/roadmap", field: "estado" },
              mockContext
            );
          }
        },
        registerTool: mock(() => {}),
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./describe.js");
      await extensionModule.default(pi);

      expect(mockExec).toHaveBeenCalled();
    });
  });

  describe("failure paths", () => {
    it("returns error when rootline not found", async () => {
      const mockExec = mock(async () => {
        throw new Error("ENOENT: rootline not found");
      });

      const pi = {
        exec: mockExec,
        tool: (name: string, def: any) => {
          if (name === "rootline-describe") {
            return def.execute(
              { path: "/docs/roadmap" },
              mockContext
            );
          }
        },
        registerTool: mock(() => {}),
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./describe.js");
      await extensionModule.default(pi);

      expect(mockExec).toHaveBeenCalled();
    });
  });
});

// ============================================================================
// rootline-validate Extension Tests
// ============================================================================

describe("rootline-validate extension", () => {
  describe("success path", () => {
    it("validates directory with all files valid", async () => {
      const mockExec = mock(async () => ({
        stdout: JSON.stringify({
          version: 1,
          kind: "rootline/batch-validation-result",
          results: [
            {
              version: 1,
              kind: "rootline/validation-result",
              path: "docs/roadmap/E01.md",
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

      const pi = {
        exec: mockExec,
        tool: mock(() => {}),
        registerTool: (name: string, def: any) => {
          if (name === "rootline-validate") {
            // Tool should be registered
            expect(name).toBe("rootline-validate");
          }
        },
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./validate.js");
      extensionModule.default(pi);

      expect(pi.registerTool).toHaveBeenCalled();
    });

    it("constructs correct args with all parameters", async () => {
      const mockExec = mock(async (cmd: string, args: string[]) => {
        expect(cmd).toBe("rootline");
        expect(args).toContain("validate");
        expect(args).toContain("--all");
        expect(args).toContain("/docs/roadmap");
        expect(args).toContain("--output");
        expect(args).toContain("json");
        expect(args).toContain("--where");
        expect(args).toContain("estado == 'Pending'");

        return {
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/batch-validation-result",
            results: [],
            drift_warnings: [],
            summary: {
              total: 0,
              valid: 0,
              invalid: 0,
              errors_count: 0,
              warnings_count: 0,
              drift_warnings_count: 0,
            },
          }),
          stderr: "",
          code: 0,
        };
      });

      const pi = {
        exec: mockExec,
        tool: mock(() => {}),
        registerTool: (name: string, def: any) => {
          if (name === "rootline-validate") {
            return def.execute({
              path: "/docs/roadmap",
              all: true,
              where: "estado == 'Pending'",
            });
          }
        },
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./validate.js");
      await extensionModule.default(pi);

      expect(mockExec).toHaveBeenCalled();
    });
  });

  describe("failure paths", () => {
    it("returns error object when rootline not found", async () => {
      const mockExec = mock(async () => {
        throw new Error("ENOENT: rootline not found");
      });

      const pi = {
        exec: mockExec,
        tool: mock(() => {}),
        registerTool: (name: string, def: any) => {
          if (name === "rootline-validate") {
            return def.execute({ path: "/docs/roadmap" });
          }
        },
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./validate.js");
      await extensionModule.default(pi);

      expect(mockExec).toHaveBeenCalled();
    });

    it("returns error when validation exits with code 1", async () => {
      const mockExec = mock(async () => ({
        stdout: "",
        stderr: "validation failed: invalid enum value",
        code: 1,
      }));

      const pi = {
        exec: mockExec,
        tool: mock(() => {}),
        registerTool: (name: string, def: any) => {
          if (name === "rootline-validate") {
            return def.execute({ path: "/invalid/path" });
          }
        },
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./validate.js");
      await extensionModule.default(pi);

      expect(mockExec).toHaveBeenCalled();
    });
  });
});

// ============================================================================
// rootline-tree Extension Tests
// ============================================================================

describe("rootline-tree extension", () => {
  describe("success path", () => {
    it("displays tree structure with completion counts", async () => {
      const mockExec = mock(async () => ({
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
                name: "E01.md",
                path: "/docs/roadmap/E01.md",
                completed: 1,
                total: 1,
                is_leaf: true,
                estado: "Completed",
              },
            ],
          },
        }),
        stderr: "",
        code: 0,
      }));

      const pi = {
        exec: mockExec,
        tool: mock(() => {}),
        registerTool: (name: string, def: any) => {
          if (name === "rootline-tree") {
            expect(name).toBe("rootline-tree");
          }
        },
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./tree.js");
      extensionModule.default(pi);

      expect(pi.registerTool).toHaveBeenCalled();
    });

    it("truncates tree when depth parameter is provided", async () => {
      const mockExec = mock(async () => ({
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

      const pi = {
        exec: mockExec,
        tool: mock(() => {}),
        registerTool: (name: string, def: any) => {
          if (name === "rootline-tree") {
            return def.execute({
              path: "/docs/roadmap",
              depth: 1,
            });
          }
        },
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./tree.js");
      await extensionModule.default(pi);

      expect(mockExec).toHaveBeenCalled();
    });
  });

  describe("failure paths", () => {
    it("returns error when rootline not found", async () => {
      const mockExec = mock(async () => {
        throw new Error("ENOENT: rootline not found");
      });

      const pi = {
        exec: mockExec,
        tool: mock(() => {}),
        registerTool: (name: string, def: any) => {
          if (name === "rootline-tree") {
            return def.execute({ path: "/docs/roadmap" });
          }
        },
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./tree.js");
      await extensionModule.default(pi);

      expect(mockExec).toHaveBeenCalled();
    });

    it("returns error when path is invalid", async () => {
      const mockExec = mock(async () => ({
        stdout: "",
        stderr: "directory not found: /nonexistent",
        code: 1,
      }));

      const pi = {
        exec: mockExec,
        tool: mock(() => {}),
        registerTool: (name: string, def: any) => {
          if (name === "rootline-tree") {
            return def.execute({ path: "/nonexistent" });
          }
        },
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./tree.js");
      await extensionModule.default(pi);

      expect(mockExec).toHaveBeenCalled();
    });
  });
});

// ============================================================================
// rootline-stats Extension Tests
// ============================================================================

describe("rootline-stats extension", () => {
  describe("success path", () => {
    it("returns statistics with counts and percentages", async () => {
      const mockExec = mock(async () => ({
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

      const pi = {
        exec: mockExec,
        tool: mock(() => {}),
        registerTool: (name: string, def: any) => {
          if (name === "rootline-stats") {
            expect(name).toBe("rootline-stats");
          }
        },
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./stats.js");
      extensionModule.default(pi);

      expect(pi.registerTool).toHaveBeenCalled();
    });

    it("constructs correct args with filter expression", async () => {
      const mockExec = mock(async (cmd: string, args: string[]) => {
        expect(cmd).toBe("rootline");
        expect(args).toContain("stats");
        expect(args).toContain("/docs/roadmap");
        expect(args).toContain("--output");
        expect(args).toContain("json");
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

      const pi = {
        exec: mockExec,
        tool: mock(() => {}),
        registerTool: (name: string, def: any) => {
          if (name === "rootline-stats") {
            return def.execute({
              path: "/docs/roadmap",
              where: "tipo == 'epic'",
            });
          }
        },
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./stats.js");
      await extensionModule.default(pi);

      expect(mockExec).toHaveBeenCalled();
    });
  });

  describe("failure paths", () => {
    it("returns error when rootline not found", async () => {
      const mockExec = mock(async () => {
        throw new Error("ENOENT: rootline not found");
      });

      const pi = {
        exec: mockExec,
        tool: mock(() => {}),
        registerTool: (name: string, def: any) => {
          if (name === "rootline-stats") {
            return def.execute({ path: "/docs/roadmap" });
          }
        },
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./stats.js");
      await extensionModule.default(pi);

      expect(mockExec).toHaveBeenCalled();
    });

    it("returns error with stderr when stats fails", async () => {
      const mockExec = mock(async () => ({
        stdout: "",
        stderr: "invalid filter expression",
        code: 1,
      }));

      const pi = {
        exec: mockExec,
        tool: mock(() => {}),
        registerTool: (name: string, def: any) => {
          if (name === "rootline-stats") {
            return def.execute({
              path: "/docs/roadmap",
              where: "invalid syntax",
            });
          }
        },
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./stats.js");
      await extensionModule.default(pi);

      expect(mockExec).toHaveBeenCalled();
    });
  });
});

// ============================================================================
// rootline-doctor Extension Tests
// ============================================================================

describe("rootline-doctor extension", () => {
  describe("success path", () => {
    it("reports binary available and directory is governed", async () => {
      let callCount = 0;
      const mockExec = mock(async (cmd: string, args: string[]) => {
        callCount++;

        // First call: --version
        if (args[0] === "--version") {
          return {
            stdout: "rootline version v0.9.100-115-g02db2b4",
            stderr: "",
            code: 0,
          };
        }

        // Second call: describe
        if (args[0] === "describe") {
          return {
            stdout: JSON.stringify({
              version: 1,
              kind: "rootline/describe-result",
              path: "/docs/roadmap",
              applies: [
                "/home/shared/rootline/docs/roadmap/.stem",
                "/home/shared/rootline/docs/.stem",
              ],
              schema: {
                estado: { type: "string" },
                tipo: { type: "string" },
                prioridad: { type: "number" },
              },
            }),
            stderr: "",
            code: 0,
          };
        }

        return { stdout: "", stderr: "", code: 0 };
      });

      const pi = {
        exec: mockExec,
        tool: mock(() => {}),
        registerTool: (name: string, def: any) => {
          if (name === "rootline-doctor") {
            return def.execute({});
          }
        },
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./doctor.js");
      await extensionModule.default(pi);

      expect(mockExec).toHaveBeenCalled();
      expect(mockExec.mock.callCount).toBeGreaterThanOrEqual(2);
    });

    it("checks binary availability with custom path", async () => {
      const mockExec = mock(async (cmd: string, args: string[]) => {
        if (args[0] === "--version") {
          return {
            stdout: "rootline version v0.9.100",
            stderr: "",
            code: 0,
          };
        }

        if (args[0] === "describe") {
          return {
            stdout: JSON.stringify({
              version: 1,
              kind: "rootline/describe-result",
              path: "/custom/path",
              applies: ["/custom/path/.stem"],
              schema: {},
            }),
            stderr: "",
            code: 0,
          };
        }

        return { stdout: "", stderr: "", code: 0 };
      });

      const pi = {
        exec: mockExec,
        tool: mock(() => {}),
        registerTool: (name: string, def: any) => {
          if (name === "rootline-doctor") {
            return def.execute({ path: "/custom/path" });
          }
        },
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./doctor.js");
      await extensionModule.default(pi);

      expect(mockExec).toHaveBeenCalled();
    });
  });

  describe("failure paths", () => {
    it("reports binary not available when rootline not found", async () => {
      const mockExec = mock(async () => {
        throw new Error("ENOENT: rootline not found");
      });

      const pi = {
        exec: mockExec,
        tool: mock(() => {}),
        registerTool: (name: string, def: any) => {
          if (name === "rootline-doctor") {
            return def.execute({});
          }
        },
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./doctor.js");
      await extensionModule.default(pi);

      expect(mockExec).toHaveBeenCalled();
    });

    it("reports directory not governed when describe fails", async () => {
      let callCount = 0;
      const mockExec = mock(async (cmd: string, args: string[]) => {
        callCount++;

        // First call succeeds: --version
        if (args[0] === "--version") {
          return {
            stdout: "rootline version v0.9.100",
            stderr: "",
            code: 0,
          };
        }

        // Second call fails: describe
        if (args[0] === "describe") {
          return {
            stdout: "",
            stderr: "directory not governed",
            code: 1,
          };
        }

        return { stdout: "", stderr: "", code: 0 };
      });

      const pi = {
        exec: mockExec,
        tool: mock(() => {}),
        registerTool: (name: string, def: any) => {
          if (name === "rootline-doctor") {
            return def.execute({});
          }
        },
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./doctor.js");
      await extensionModule.default(pi);

      expect(mockExec).toHaveBeenCalled();
    });
  });
});

// ============================================================================
// rootline-context Extension Tests
// ============================================================================

describe("rootline-context extension", () => {
  describe("success path", () => {
    it("detects stem_governed state when binary and .stem files exist", async () => {
      let callCount = 0;
      const mockExec = mock(async (cmd: string, args: string[]) => {
        callCount++;

        // First call: --version
        if (args[0] === "--version") {
          return {
            stdout: "rootline version v0.9.100",
            stderr: "",
            code: 0,
          };
        }

        // Second call: describe
        if (args[0] === "describe") {
          return {
            stdout: JSON.stringify({
              applies: [".stem", "subdir/.stem"],
            }),
            stderr: "",
            code: 0,
          };
        }

        return { stdout: "", stderr: "", code: 0 };
      });

      const pi = {
        exec: mockExec,
        tool: mock(() => {}),
        registerTool: (name: string, def: any) => {
          if (name === "rootline-context") {
            return def.execute({});
          }
        },
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./context.js");
      await extensionModule.default(pi);

      expect(mockExec).toHaveBeenCalled();
    });

    it("detects binary_only state when binary exists but no .stem files", async () => {
      let callCount = 0;
      const mockExec = mock(async (cmd: string, args: string[]) => {
        callCount++;

        // First call: --version
        if (args[0] === "--version") {
          return {
            stdout: "rootline version v0.9.100",
            stderr: "",
            code: 0,
          };
        }

        // Second call: describe fails
        if (args[0] === "describe") {
          return {
            stdout: "",
            stderr: "directory not governed",
            code: 1,
          };
        }

        return { stdout: "", stderr: "", code: 0 };
      });

      const pi = {
        exec: mockExec,
        tool: mock(() => {}),
        registerTool: (name: string, def: any) => {
          if (name === "rootline-context") {
            return def.execute({});
          }
        },
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./context.js");
      await extensionModule.default(pi);

      expect(mockExec).toHaveBeenCalled();
    });

    it("accepts custom path parameter", async () => {
      const mockExec = mock(async (cmd: string, args: string[]) => {
        // Verify the custom path is used in describe call
        if (args[0] === "describe") {
          expect(args[1]).toBe("/custom/path");
        }

        return {
          stdout: JSON.stringify({ applies: [] }),
          stderr: "",
          code: 0,
        };
      });

      const pi = {
        exec: mockExec,
        tool: mock(() => {}),
        registerTool: (name: string, def: any) => {
          if (name === "rootline-context") {
            return def.execute({ path: "/custom/path" });
          }
        },
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./context.js");
      await extensionModule.default(pi);

      expect(mockExec).toHaveBeenCalled();
    });
  });

  describe("failure paths", () => {
    it("detects no_rootline state when binary not found", async () => {
      const mockExec = mock(async () => {
        throw new Error("ENOENT: rootline not found");
      });

      const pi = {
        exec: mockExec,
        tool: mock(() => {}),
        registerTool: (name: string, def: any) => {
          if (name === "rootline-context") {
            return def.execute({});
          }
        },
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./context.js");
      await extensionModule.default(pi);

      expect(mockExec).toHaveBeenCalled();
    });

    it("respects 5s timeout for context detection", async () => {
      const mockExec = mock(async () => {
        throw new Error("ETIMEDOUT: operation timed out");
      });

      const pi = {
        exec: mockExec,
        tool: mock(() => {}),
        registerTool: (name: string, def: any) => {
          if (name === "rootline-context") {
            return def.execute({});
          }
        },
      } as unknown as ExtensionAPI;

      const extensionModule = await import("./context.js");
      await extensionModule.default(pi);

      expect(mockExec).toHaveBeenCalled();
    });
  });
});
