import { describe, it, expect, mock } from "bun:test";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import autocompleteExt from "./autocomplete";

/**
 * Tests for rootline-autocomplete extension.
 *
 * Tests verify:
 * - Tool registration with correct schema and description
 * - Disabled mode returns early with disabled: true
 * - Schema field extraction and formatting
 * - Field filtering by prefix
 * - Enum value extraction
 * - Error handling when rootline is not found
 */

describe("rootline-autocomplete extension", () => {
  describe("tool registration", () => {
    it("registers rootline-autocomplete with correct description", () => {
      const registerToolMock = mock(() => {});
      const pi = {
        exec: mock(async () => ({
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/describe-result",
            path: "/docs/roadmap",
            schema: {},
          }),
          stderr: "",
          code: 0,
        })),
        registerTool: registerToolMock,
      } as unknown as ExtensionAPI;

      autocompleteExt(pi);

      expect(registerToolMock).toHaveBeenCalled();
      const call = registerToolMock.mock.calls[0];
      const toolDef = call[0];
      expect(toolDef.name).toBe("rootline-autocomplete");
      expect(toolDef.description).toContain("autocomplete suggestions");
      expect(toolDef.description).toContain("schema fields");
    });

    it("has correct input schema with required path parameter", () => {
      const registerToolMock = mock(() => {});
      const pi = {
        exec: mock(async () => ({
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/describe-result",
            path: "/docs/roadmap",
            schema: {},
          }),
          stderr: "",
          code: 0,
        })),
        registerTool: registerToolMock,
      } as unknown as ExtensionAPI;

      autocompleteExt(pi);

      const call = registerToolMock.mock.calls[0];
      const toolDef = call[0];
      expect(toolDef.input_schema).toBeDefined();
      expect(toolDef.input_schema.type).toBe("object");
      expect(toolDef.input_schema.properties.path).toBeDefined();
      expect(toolDef.input_schema.required).toContain("path");
      expect(toolDef.input_schema.properties.field).toBeDefined();
      expect(toolDef.input_schema.properties.disabled).toBeDefined();
    });
  });

  describe("disabled mode", () => {
    it("returns disabled: true when disabled parameter is true", async () => {
      let toolExecute: Function | null = null;
      const execMock = mock(async () => ({
        stdout: JSON.stringify({
          version: 1,
          kind: "rootline/describe-result",
          path: "/docs/roadmap",
          schema: {
            estado: {
              type: "enum",
              enum: ["Pending", "In Progress", "Completed"],
            },
          },
        }),
        stderr: "",
        code: 0,
      }));

      const pi = {
        exec: execMock,
        registerTool: mock((toolDef: any) => {
          toolExecute = toolDef.execute;
        }),
      } as unknown as ExtensionAPI;

      autocompleteExt(pi);
      const result = await toolExecute?.({
        path: "/docs/roadmap",
        disabled: true,
      });

      expect(result.disabled).toBe(true);
      expect(result.fields).toEqual([]);
      expect(result.suggestions).toEqual([]);
      expect(execMock).not.toHaveBeenCalled();
    });

    it("returns disabled: false by default", async () => {
      let toolExecute: Function | null = null;
      const pi = {
        exec: mock(async () => ({
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/describe-result",
            path: "/docs/roadmap",
            schema: {
              estado: { type: "enum", enum: ["Pending", "In Progress"] },
            },
          }),
          stderr: "",
          code: 0,
        })),
        registerTool: mock((toolDef: any) => {
          toolExecute = toolDef.execute;
        }),
      } as unknown as ExtensionAPI;

      autocompleteExt(pi);
      const result = await toolExecute?.({ path: "/docs/roadmap" });

      expect(result.disabled).toBe(false);
      expect(result.fields).toContain("estado");
    });
  });

  describe("schema field extraction", () => {
    it("extracts all field names from schema", async () => {
      let toolExecute: Function | null = null;
      const pi = {
        exec: mock(async () => ({
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/describe-result",
            path: "/docs/roadmap",
            schema: {
              estado: { type: "string" },
              tipo: { type: "string" },
              prioridad: { type: "number" },
            },
          }),
          stderr: "",
          code: 0,
        })),
        registerTool: mock((toolDef: any) => {
          toolExecute = toolDef.execute;
        }),
      } as unknown as ExtensionAPI;

      autocompleteExt(pi);
      const result = await toolExecute?.({ path: "/docs/roadmap" });

      expect(result.fields).toContain("estado");
      expect(result.fields).toContain("tipo");
      expect(result.fields).toContain("prioridad");
      expect(result.fields.length).toBe(3);
    });

    it("includes type information in suggestions", async () => {
      let toolExecute: Function | null = null;
      const pi = {
        exec: mock(async () => ({
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/describe-result",
            path: "/docs/roadmap",
            schema: {
              estado: { type: "enum" },
              count: { type: "number" },
              title: { type: "string" },
            },
          }),
          stderr: "",
          code: 0,
        })),
        registerTool: mock((toolDef: any) => {
          toolExecute = toolDef.execute;
        }),
      } as unknown as ExtensionAPI;

      autocompleteExt(pi);
      const result = await toolExecute?.({ path: "/docs/roadmap" });

      const estadoSug = result.suggestions.find(
        (s: any) => s.field === "estado"
      );
      const countSug = result.suggestions.find((s: any) => s.field === "count");
      const titleSug = result.suggestions.find((s: any) => s.field === "title");

      expect(estadoSug.type).toBe("enum");
      expect(countSug.type).toBe("number");
      expect(titleSug.type).toBe("string");
    });

    it("extracts enum values for enum type fields", async () => {
      let toolExecute: Function | null = null;
      const pi = {
        exec: mock(async () => ({
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/describe-result",
            path: "/docs/roadmap",
            schema: {
              estado: {
                type: "enum",
                enum: ["Pending", "In Progress", "Completed"],
              },
              tipo: { type: "string" },
            },
          }),
          stderr: "",
          code: 0,
        })),
        registerTool: mock((toolDef: any) => {
          toolExecute = toolDef.execute;
        }),
      } as unknown as ExtensionAPI;

      autocompleteExt(pi);
      const result = await toolExecute?.({ path: "/docs/roadmap" });

      const estadoSug = result.suggestions.find(
        (s: any) => s.field === "estado"
      );
      expect(estadoSug.values).toEqual([
        "Pending",
        "In Progress",
        "Completed",
      ]);

      const tipoSug = result.suggestions.find((s: any) => s.field === "tipo");
      expect(tipoSug.values).toBeUndefined();
    });

    it("includes required flag in suggestions", async () => {
      let toolExecute: Function | null = null;
      const pi = {
        exec: mock(async () => ({
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/describe-result",
            path: "/docs/roadmap",
            schema: {
              estado: {
                type: "enum",
                enum: ["Pending", "In Progress"],
                required: true,
              },
              tipo: { type: "string", required: false },
            },
          }),
          stderr: "",
          code: 0,
        })),
        registerTool: mock((toolDef: any) => {
          toolExecute = toolDef.execute;
        }),
      } as unknown as ExtensionAPI;

      autocompleteExt(pi);
      const result = await toolExecute?.({ path: "/docs/roadmap" });

      const estadoSug = result.suggestions.find(
        (s: any) => s.field === "estado"
      );
      const tipoSug = result.suggestions.find((s: any) => s.field === "tipo");

      expect(estadoSug.required).toBe(true);
      expect(tipoSug.required).toBe(false);
    });
  });

  describe("field filtering", () => {
    it("filters fields by prefix when field parameter is provided", async () => {
      let toolExecute: Function | null = null;
      const pi = {
        exec: mock(async () => ({
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/describe-result",
            path: "/docs/roadmap",
            schema: {
              estado: { type: "enum" },
              ejecutable_en: { type: "string" },
              tipo: { type: "string" },
            },
          }),
          stderr: "",
          code: 0,
        })),
        registerTool: mock((toolDef: any) => {
          toolExecute = toolDef.execute;
        }),
      } as unknown as ExtensionAPI;

      autocompleteExt(pi);
      const result = await toolExecute?.({
        path: "/docs/roadmap",
        field: "ej",
      });

      expect(result.fields).toEqual(["ejecutable_en"]);
      expect(result.suggestions.length).toBe(1);
      expect(result.suggestions[0].field).toBe("ejecutable_en");
    });

    it("performs case-insensitive filtering", async () => {
      let toolExecute: Function | null = null;
      const pi = {
        exec: mock(async () => ({
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/describe-result",
            path: "/docs/roadmap",
            schema: {
              estado: { type: "enum" },
              Estado_alt: { type: "string" },
            },
          }),
          stderr: "",
          code: 0,
        })),
        registerTool: mock((toolDef: any) => {
          toolExecute = toolDef.execute;
        }),
      } as unknown as ExtensionAPI;

      autocompleteExt(pi);
      const result = await toolExecute?.({
        path: "/docs/roadmap",
        field: "EST",
      });

      expect(result.fields.length).toBe(2);
      expect(result.fields).toContain("estado");
      expect(result.fields).toContain("Estado_alt");
    });

    it("returns empty suggestions when no fields match prefix", async () => {
      let toolExecute: Function | null = null;
      const pi = {
        exec: mock(async () => ({
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/describe-result",
            path: "/docs/roadmap",
            schema: {
              estado: { type: "enum" },
              tipo: { type: "string" },
            },
          }),
          stderr: "",
          code: 0,
        })),
        registerTool: mock((toolDef: any) => {
          toolExecute = toolDef.execute;
        }),
      } as unknown as ExtensionAPI;

      autocompleteExt(pi);
      const result = await toolExecute?.({
        path: "/docs/roadmap",
        field: "xyz",
      });

      expect(result.fields).toEqual([]);
      expect(result.suggestions).toEqual([]);
    });
  });

  describe("error handling", () => {
    it("returns error message when rootline describe fails", async () => {
      let toolExecute: Function | null = null;
      const pi = {
        exec: mock(async () => {
          throw new Error("ENOENT: rootline not found");
        }),
        registerTool: mock((toolDef: any) => {
          toolExecute = toolDef.execute;
        }),
      } as unknown as ExtensionAPI;

      autocompleteExt(pi);
      const result = await toolExecute?.({ path: "/docs/roadmap" });

      expect(result.error).toBeDefined();
      expect(result.disabled).toBe(false);
      expect(result.fields).toEqual([]);
      expect(result.suggestions).toEqual([]);
    });

    it("handles missing schema in describe result", async () => {
      let toolExecute: Function | null = null;
      const pi = {
        exec: mock(async () => ({
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/describe-result",
            path: "/docs/roadmap",
          }),
          stderr: "",
          code: 0,
        })),
        registerTool: mock((toolDef: any) => {
          toolExecute = toolDef.execute;
        }),
      } as unknown as ExtensionAPI;

      autocompleteExt(pi);
      const result = await toolExecute?.({ path: "/docs/roadmap" });

      expect(result.disabled).toBe(false);
      expect(result.fields).toEqual([]);
      expect(result.suggestions).toEqual([]);
    });

    it("handles describe exit failure", async () => {
      let toolExecute: Function | null = null;
      const pi = {
        exec: mock(async () => ({
          stdout: "",
          stderr: "directory not found: /invalid",
          code: 1,
        })),
        registerTool: mock((toolDef: any) => {
          toolExecute = toolDef.execute;
        }),
      } as unknown as ExtensionAPI;

      autocompleteExt(pi);
      const result = await toolExecute?.({ path: "/invalid" });

      expect(result.error).toBeDefined();
      expect(result.disabled).toBe(false);
      expect(result.fields).toEqual([]);
    });
  });
});
