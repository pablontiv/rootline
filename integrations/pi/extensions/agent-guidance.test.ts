import { describe, it, expect, mock } from "bun:test";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import registerAgentGuidance from "./agent-guidance";

/**
 * Tests for agent-guidance extension.
 *
 * Tests verify:
 * - Tool registration with correct schema and description
 * - Input schema validation (optional path parameter)
 * - Guidance tool is available for pre-agent context injection
 *
 * Note: Actual execution behavior (state detection) is tested via
 * integration tests with the real rootline binary, not unit tests,
 * because the JSON parsing behavior depends on rootline's actual output format.
 */

describe("agent-guidance extension", () => {
  describe("tool registration", () => {
    it("registers rootline-agent-guidance tool", () => {
      const registerToolMock = mock(() => {});
      const pi = {
        registerTool: registerToolMock,
        exec: mock(async () => ({
          stdout: "rootline version v0.9.0",
          stderr: "",
          code: 0,
        })),
        tool: mock(() => {}),
      } as unknown as ExtensionAPI;

      registerAgentGuidance(pi);

      expect(registerToolMock).toHaveBeenCalled();
      const toolDef = registerToolMock.mock.calls[0][0];
      expect(toolDef.name).toBe("rootline-agent-guidance");
    });

    it("provides compact guidance description", () => {
      const registerToolMock = mock(() => {});
      const pi = {
        registerTool: registerToolMock,
        exec: mock(async () => ({
          stdout: "rootline version v0.9.0",
          stderr: "",
          code: 0,
        })),
        tool: mock(() => {}),
      } as unknown as ExtensionAPI;

      registerAgentGuidance(pi);

      const toolDef = registerToolMock.mock.calls[0][0];
      expect(toolDef.description).toContain("Inject compact Rootline guidance");
      expect(toolDef.description).toContain("state-specific");
    });
  });

  describe("input schema", () => {
    it("accepts optional path parameter", () => {
      const registerToolMock = mock(() => {});
      const pi = {
        registerTool: registerToolMock,
        exec: mock(async () => ({
          stdout: "rootline version v0.9.0",
          stderr: "",
          code: 0,
        })),
        tool: mock(() => {}),
      } as unknown as ExtensionAPI;

      registerAgentGuidance(pi);

      const toolDef = registerToolMock.mock.calls[0][0];
      expect(toolDef.input_schema).toBeDefined();
      expect(toolDef.input_schema.type).toBe("object");
      expect(toolDef.input_schema.properties.path).toBeDefined();
      expect(toolDef.input_schema.required).toBeDefined();
      expect(toolDef.input_schema.required.length).toBe(0);
    });

    it("has valid tool schema", () => {
      const registerToolMock = mock(() => {});
      const pi = {
        registerTool: registerToolMock,
        exec: mock(async () => ({
          stdout: "rootline version v0.9.0",
          stderr: "",
          code: 0,
        })),
        tool: mock(() => {}),
      } as unknown as ExtensionAPI;

      registerAgentGuidance(pi);

      const toolDef = registerToolMock.mock.calls[0][0];
      expect(toolDef.execute).toBeDefined();
      expect(typeof toolDef.execute).toBe("function");
    });
  });
});
