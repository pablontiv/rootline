import { describe, it, expect, mock } from "bun:test";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import registerContext from "./context";
import registerAgentGuidance from "./agent-guidance";
import registerStatus from "./status";

/**
 * Helper to create a fresh mock ExtensionAPI for testing.
 *
 * The execBehavior function should return mock data. The helper will:
 * - JSON-encode strings and objects before returning them as stdout
 * - Throw errors that are thrown by execBehavior
 */
function createMockPI(
  execBehavior: (cmd: string, args: string[]) => Promise<any>
) {
  let toolExecute: Function | null = null;
  const pi = {
    exec: mock(async (cmd: string, args: string[], opts?: any) => {
      try {
        const data = await execBehavior(cmd, args);
        // JSON-encode the response
        const stdout = JSON.stringify(data);
        return { stdout, stderr: "", code: 0 };
      } catch (err) {
        // Re-throw to simulate exec failure
        throw err;
      }
    }),
    tool: mock(() => {}),
    registerTool: (def: any) => {
      toolExecute = def.execute;
    },
  } as unknown as ExtensionAPI;
  return { pi, getExecute: () => toolExecute };
}

/**
 * Integration tests for Rootline context extensions.
 *
 * These tests verify that the context, agent-guidance, and status extensions
 * correctly distinguish between three project states:
 *
 * 1. **Rootline repo scenario** (governed directory):
 *    - Binary present and functional
 *    - .stem files exist in the target directory
 *    - Extensions return meaningful guidance and status
 *
 * 2. **Plain repo scenario** (binary present, no governance):
 *    - Binary present and functional
 *    - No .stem files in target directory
 *    - Extensions return init guidance
 *
 * 3. **No rootline scenario** (binary not in PATH):
 *    - Binary not found
 *    - Extensions return minimal/no guidance
 *
 * These tests use mocked pi.exec to simulate various scenarios without
 * requiring actual rootline binaries or directory structures.
 */

describe("Rootline context extensions integration", () => {
  describe("Rootline repo scenario (stem_governed)", () => {
    it("context.ts returns stem_governed state with stem count", async () => {
      const { pi, getExecute } = createMockPI(
        async (cmd: string, args: string[]) => {
          if (args[0] === "--version") {
            return "rootline version v0.9.0";
          }
          if (args[0] === "describe") {
            return {
              version: 1,
              kind: "rootline/describe-result",
              path: "/home/project/docs",
              schema: {},
              applies: ["/home/project/docs/.stem", "/home/project/docs/roadmap/.stem"],
            };
          }
          throw new Error("not found");
        }
      );

      registerContext(pi);
      const toolExecute = getExecute();

      const result = await toolExecute?.({ path: "/home/project/docs" });

      expect(result).toBeDefined();
      expect(result.state).toBe("stem_governed");
      expect(result.version).toContain("v0.9.0");
      expect(result.stem_count).toBe(2);
    });

    it("agent-guidance.ts returns full tool usage guidance for governed repos", async () => {
      const { pi, getExecute } = createMockPI(
        async (cmd: string, args: string[]) => {
          if (args[0] === "--version") {
            return "rootline version v0.9.0";
          }
          if (args[0] === "describe") {
            return {
              version: 1,
              kind: "rootline/describe-result",
              path: "/home/project/docs",
              schema: {},
              applies: ["/home/project/docs/.stem"],
            };
          }
          throw new Error("not found");
        }
      );

      registerAgentGuidance(pi);
      const toolExecute = getExecute();

      const result = await toolExecute?.({ path: "/home/project/docs" });

      expect(result).toBeDefined();
      expect(result.state).toBe("stem_governed");
      expect(result.stem_count).toBe(1);
      expect(result.guidance).toContain("Rootline Context");
      expect(result.guidance).toContain("rootline-query");
      expect(result.guidance).toContain("rootline-validate");
      expect(result.guidance).toContain("rootline-describe");
      expect(result.guidance).toContain("rootline-tree");
      expect(result.guidance.length).toBeGreaterThan(50);
    });

    it("status.ts returns checkmark status for governed repos", async () => {
      const { pi, getExecute } = createMockPI(
        async (cmd: string, args: string[]) => {
          if (args[0] === "--version") {
            return "v0.9.0";
          }
          if (args[0] === "describe") {
            return {
              version: 1,
              kind: "rootline/describe-result",
              path: "/home/project/docs",
              schema: {},
              applies: ["/home/project/docs/.stem"],
            };
          }
          if (args[0] === "validate") {
            return {
              version: 1,
              kind: "rootline/validation-result",
              summary: {
                total: 10,
                valid: 10,
                invalid: 0,
                errors_count: 0,
                warnings_count: 0,
              },
            };
          }
          throw new Error("not found");
        }
      );

      registerStatus(pi);
      const toolExecute = getExecute();

      const result = await toolExecute?.({ path: "/home/project/docs" });

      expect(result).toBeDefined();
      expect(result.state).toBe("stem_governed");
      expect(result.status_line).toContain("✓");
      expect(result.status_line).toContain("governed");
      expect(result.stem_count).toBe(1);
      expect(result.valid).toBe(10);
      expect(result.errors).toBe(0);
      expect(result.warnings).toBe(0);
    });

    it("status.ts includes error count when validation fails", async () => {
      const { pi, getExecute } = createMockPI(
        async (cmd: string, args: string[]) => {
          if (args[0] === "--version") {
            return "v0.9.0";
          }
          if (args[0] === "describe") {
            return {
              version: 1,
              kind: "rootline/describe-result",
              path: "/home/project/docs",
              schema: {},
              applies: ["/home/project/docs/.stem"],
            };
          }
          if (args[0] === "validate") {
            return {
              version: 1,
              kind: "rootline/validation-result",
              summary: {
                total: 15,
                valid: 12,
                invalid: 3,
                errors_count: 3,
                warnings_count: 1,
              },
            };
          }
          throw new Error("not found");
        }
      );

      registerStatus(pi);
      const toolExecute = getExecute();

      const result = await toolExecute?.({ path: "/home/project/docs" });

      expect(result.state).toBe("stem_governed");
      expect(result.status_line).toContain("✓");
      expect(result.status_line).toContain("3 errors");
      expect(result.status_line).toContain("1 warn");
    });
  });

  describe("Plain repo scenario (binary_only)", () => {
    it("context.ts returns binary_only state when no .stem files found", async () => {
      const { pi, getExecute } = createMockPI(
        async (cmd: string, args: string[]) => {
          if (args[0] === "--version") {
            return "rootline version v0.9.0";
          }
          if (args[0] === "describe") {
            throw new Error("no .stem files found in /home/plain-repo");
          }
          throw new Error("not found");
        }
      );

      registerContext(pi);
      const toolExecute = getExecute();

      const result = await toolExecute?.({ path: "/home/plain-repo" });

      expect(result).toBeDefined();
      expect(result.state).toBe("binary_only");
      expect(result.version).toContain("v0.9.0");
      expect(result.stem_count).toBeUndefined();
    });

    it("agent-guidance.ts returns init guidance for plain repos", async () => {
      const { pi, getExecute } = createMockPI(
        async (cmd: string, args: string[]) => {
          if (args[0] === "--version") {
            return "rootline version v0.9.0";
          }
          if (args[0] === "describe") {
            throw new Error("no .stem files found");
          }
          throw new Error("not found");
        }
      );

      registerAgentGuidance(pi);
      const toolExecute = getExecute();

      const result = await toolExecute?.({ path: "/home/plain-repo" });

      expect(result).toBeDefined();
      expect(result.state).toBe("binary_only");
      expect(result.guidance).toContain("Rootline is available");
      expect(result.guidance).toContain("no .stem files");
      expect(result.guidance).toContain("rootline init");
      expect(result.guidance.length).toBeGreaterThan(0);
    });

    it("status.ts returns warning status for plain repos", async () => {
      const { pi, getExecute } = createMockPI(
        async (cmd: string, args: string[]) => {
          if (args[0] === "--version") {
            return "v0.9.0";
          }
          if (args[0] === "describe") {
            throw new Error("no .stem files found");
          }
          throw new Error("not found");
        }
      );

      registerStatus(pi);
      const toolExecute = getExecute();

      const result = await toolExecute?.({ path: "/home/plain-repo" });

      expect(result).toBeDefined();
      expect(result.state).toBe("binary_only");
      expect(result.status_line).toContain("⚠");
      expect(result.status_line).toContain("no schema");
      expect(result.version).toBe("v0.9.0");
    });
  });

  describe("No rootline scenario (no_rootline)", () => {
    it("context.ts returns no_rootline state when binary not found", async () => {
      const { pi, getExecute } = createMockPI(
        async (cmd: string, args: string[]) => {
          throw new Error("ENOENT: rootline not found in PATH");
        }
      );

      registerContext(pi);
      const toolExecute = getExecute();

      const result = await toolExecute?.({ path: "/home/some-repo" });

      expect(result).toBeDefined();
      expect(result.state).toBe("no_rootline");
      expect(result.version).toBeUndefined();
      expect(result.stem_count).toBeUndefined();
    });

    it("agent-guidance.ts returns empty guidance for no_rootline state", async () => {
      const { pi, getExecute } = createMockPI(
        async (cmd: string, args: string[]) => {
          throw new Error("ENOENT: rootline not found");
        }
      );

      registerAgentGuidance(pi);
      const toolExecute = getExecute();

      const result = await toolExecute?.({ path: "/home/some-repo" });

      expect(result).toBeDefined();
      expect(result.state).toBe("no_rootline");
      expect(result.guidance).toBe("");
    });

    it("status.ts returns cross mark for no_rootline state", async () => {
      const { pi, getExecute } = createMockPI(
        async (cmd: string, args: string[]) => {
          throw new Error("ENOENT: rootline not found");
        }
      );

      registerStatus(pi);
      const toolExecute = getExecute();

      const result = await toolExecute?.({ path: "/home/some-repo" });

      expect(result).toBeDefined();
      expect(result.state).toBe("no_rootline");
      expect(result.status_line).toContain("✗");
      expect(result.status_line).toContain("binary not found");
    });
  });

  describe("Cross-scenario consistency", () => {
    it("all three extensions handle no_rootline consistently", async () => {
      const execError = () => {
        throw new Error("ENOENT: not found");
      };

      const { pi: contextPI, getExecute: getContextExecute } = createMockPI(
        async () => execError()
      );
      const { pi: guidancePI, getExecute: getGuidanceExecute } = createMockPI(
        async () => execError()
      );
      const { pi: statusPI, getExecute: getStatusExecute } = createMockPI(
        async () => execError()
      );

      registerContext(contextPI);
      registerAgentGuidance(guidancePI);
      registerStatus(statusPI);

      const contextExecute = getContextExecute();
      const guidanceExecute = getGuidanceExecute();
      const statusExecute = getStatusExecute();

      const contextResult = await contextExecute?.({});
      const guidanceResult = await guidanceExecute?.({});
      const statusResult = await statusExecute?.({});

      expect(contextResult.state).toBe("no_rootline");
      expect(guidanceResult.state).toBe("no_rootline");
      expect(statusResult.state).toBe("no_rootline");
    });

    it("all three extensions handle stem_governed consistently", async () => {
      const execBehavior = async (cmd: string, args: string[]) => {
        if (args[0] === "--version") {
          return "v0.9.0";
        }
        if (args[0] === "describe") {
          return {
            version: 1,
            kind: "rootline/describe-result",
            applies: ["/docs/.stem"],
          };
        }
        if (args[0] === "validate") {
          return {
            version: 1,
            kind: "rootline/validation-result",
            summary: { total: 5, valid: 5, invalid: 0, errors_count: 0 },
          };
        }
        throw new Error("not found");
      };

      const { pi: contextPI, getExecute: getContextExecute } = createMockPI(execBehavior);
      const { pi: guidancePI, getExecute: getGuidanceExecute } = createMockPI(execBehavior);
      const { pi: statusPI, getExecute: getStatusExecute } = createMockPI(execBehavior);

      registerContext(contextPI);
      registerAgentGuidance(guidancePI);
      registerStatus(statusPI);

      const contextExecute = getContextExecute();
      const guidanceExecute = getGuidanceExecute();
      const statusExecute = getStatusExecute();

      const contextResult = await contextExecute?.({ path: "/docs" });
      const guidanceResult = await guidanceExecute?.({ path: "/docs" });
      const statusResult = await statusExecute?.({ path: "/docs" });

      expect(contextResult.state).toBe("stem_governed");
      expect(guidanceResult.state).toBe("stem_governed");
      expect(statusResult.state).toBe("stem_governed");

      expect(guidanceResult.guidance).toContain("rootline-query");
      expect(statusResult.status_line).toContain("✓");
    });
  });

  describe("Validation assertions", () => {
    it("validates that rootline validate --all docs/roadmap returns exit 0", async () => {
      let capturedCommand: string[] | null = null;

      const pi = {
        exec: mock(async (cmd: string, args: string[]) => {
          capturedCommand = args;
          return {
            stdout: JSON.stringify({
              version: 1,
              kind: "rootline/validation-result",
              summary: {
                total: 10,
                valid: 10,
                invalid: 0,
                errors_count: 0,
                warnings_count: 0,
              },
            }),
            stderr: "",
            code: 0,
          };
        }),
        tool: mock(() => {}),
        registerTool: mock(() => {}),
      } as unknown as ExtensionAPI;

      const result = await pi.exec("rootline", ["validate", "--all", "docs/roadmap/"], {});

      expect(capturedCommand).toEqual([
        "validate",
        "--all",
        "docs/roadmap/",
      ]);
      expect(result.code).toBe(0);
    });
  });
});
