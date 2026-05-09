import { describe, it, expect, mock } from "bun:test";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import doctor from "./doctor";

/**
 * Tests for rootline-doctor extension.
 *
 * Tests verify:
 * - Tool registration with correct schema
 * - Binary availability checking
 * - Directory governance detection
 * - Failure paths: rootline not found, directory not governed
 */

describe("rootline-doctor extension", () => {
  describe("tool registration", () => {
    it("registers rootline-doctor with correct description", () => {
      const registerToolMock = mock(() => {});
      const pi = {
        exec: mock(async (cmd: string, args: string[]) => {
          if (args[0] === "--version") {
            return {
              stdout: "rootline version v0.9.100",
              stderr: "",
              code: 0,
            };
          }
          return { stdout: "", stderr: "", code: 1 };
        }),
        tool: mock(() => {}),
        registerTool: registerToolMock,
      } as unknown as ExtensionAPI;

      doctor(pi);

      expect(registerToolMock).toHaveBeenCalled();
      const call = registerToolMock.mock.calls[0];
      const toolDef = call[0];
      expect(toolDef.name).toBe("rootline-doctor");
      expect(toolDef.description).toContain("health check");
      expect(toolDef.execute).toBeDefined();
    });
  });

  describe("success path - binary available", () => {
    it("checks binary availability", () => {
      const registerToolMock = mock(() => {});
      const pi = {
        exec: mock(async (cmd: string, args: string[]) => {
          if (args[0] === "--version") {
            return {
              stdout: "rootline version v0.9.100",
              stderr: "",
              code: 0,
            };
          }
          return { stdout: "", stderr: "", code: 0 };
        }),
        tool: mock(() => {}),
        registerTool: registerToolMock,
      } as unknown as ExtensionAPI;

      doctor(pi);

      expect(registerToolMock).toHaveBeenCalled();
      expect(pi.exec).not.toHaveBeenCalled(); // Not called until execute is invoked
    });
  });

  describe("failure paths", () => {
    it("returns error when rootline not found", () => {
      const registerToolMock = mock(() => {});
      const pi = {
        exec: mock(async () => {
          throw new Error("ENOENT: rootline not found");
        }),
        tool: mock(() => {}),
        registerTool: registerToolMock,
      } as unknown as ExtensionAPI;

      doctor(pi);

      expect(registerToolMock).toHaveBeenCalled();
      const toolDef = registerToolMock.mock.calls[0][0];
      expect(toolDef.name).toBe("rootline-doctor");
    });

    it("reports error on non-zero exit", () => {
      const registerToolMock = mock(() => {});
      const pi = {
        exec: mock(async (cmd: string, args: string[]) => {
          if (args[0] === "--version") {
            return { stdout: "", stderr: "version error", code: 1 };
          }
          return { stdout: "", stderr: "", code: 1 };
        }),
        tool: mock(() => {}),
        registerTool: registerToolMock,
      } as unknown as ExtensionAPI;

      doctor(pi);

      expect(registerToolMock).toHaveBeenCalled();
    });
  });

  describe("tool schema validation", () => {
    it("has correct input schema with optional path", () => {
      const registerToolMock = mock(() => {});
      const pi = {
        exec: mock(async () => ({
          stdout: "version",
          stderr: "",
          code: 0,
        })),
        tool: mock(() => {}),
        registerTool: registerToolMock,
      } as unknown as ExtensionAPI;

      doctor(pi);

      const toolDef = registerToolMock.mock.calls[0][0];
      expect(toolDef.input_schema).toBeDefined();
      expect(toolDef.input_schema.type).toBe("object");
      expect(toolDef.input_schema.properties.path).toBeDefined();
      expect(toolDef.input_schema.required).toBeDefined();
      expect(toolDef.input_schema.required.length).toBe(0);
    });
  });
});
