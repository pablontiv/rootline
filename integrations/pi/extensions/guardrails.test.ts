import { describe, it, expect, mock } from "bun:test";
import {
  validateTargetPath,
  requiresConfirmation,
  validateAfterWrite,
} from "./guardrails";
import type { RunnerResult } from "../runner";

/**
 * Tests for guardrails module.
 *
 * Tests verify:
 * - Path restrictions: cwd-relative paths pass, paths outside cwd fail
 * - Confirmation requirements: non-interactive requires confirmation, interactive allows
 * - Post-validation: valid writes pass, invalid writes fail
 */

describe("guardrails", () => {
  describe("validateTargetPath", () => {
    it("allows paths within cwd", () => {
      const result = validateTargetPath("docs/roadmap/E01.md", "/home/user/project");
      expect(result.ok).toBe(true);
    });

    it("allows relative paths that resolve within cwd", () => {
      const result = validateTargetPath("./docs/test.md", "/home/user/project");
      expect(result.ok).toBe(true);
    });

    it("rejects paths outside cwd with ../ traversal", () => {
      const result = validateTargetPath("../../../etc/passwd", "/home/user/project");
      expect(result.ok).toBe(false);
      expect(result.ok === false && result.reason.includes("escapes")).toBe(true);
    });

    it("rejects absolute paths outside cwd", () => {
      const result = validateTargetPath("/etc/passwd", "/home/user/project");
      expect(result.ok).toBe(false);
      expect(result.ok === false && result.reason.includes("outside")).toBe(true);
    });

    it("allows absolute paths within cwd", () => {
      const result = validateTargetPath(
        "/home/user/project/docs/test.md",
        "/home/user/project"
      );
      expect(result.ok).toBe(true);
    });

    it("rejects paths with multiple ../ that escape cwd", () => {
      const result = validateTargetPath(
        "../../etc/passwd",
        "/home/user/project"
      );
      expect(result.ok).toBe(false);
    });
  });

  describe("requiresConfirmation", () => {
    it("requires confirmation for create in non-interactive mode", () => {
      const result = requiresConfirmation("create", false);
      expect(result).toBe(true);
    });

    it("requires confirmation for update in non-interactive mode", () => {
      const result = requiresConfirmation("update", false);
      expect(result).toBe(true);
    });

    it("requires confirmation for delete in non-interactive mode", () => {
      const result = requiresConfirmation("delete", false);
      expect(result).toBe(true);
    });

    it("does not require confirmation in interactive mode", () => {
      const result = requiresConfirmation("create", true);
      expect(result).toBe(false);
    });

    it("does not require confirmation for update in interactive mode", () => {
      const result = requiresConfirmation("update", true);
      expect(result).toBe(false);
    });

    it("does not require confirmation for delete in interactive mode", () => {
      const result = requiresConfirmation("delete", true);
      expect(result).toBe(false);
    });
  });

  describe("validateAfterWrite", () => {
    it("passes validation when rootline returns valid result", async () => {
      const mockRun = mock(async () => ({
        ok: true,
        data: {
          version: 1,
          kind: "rootline/batch-validation-result",
          results: [
            {
              path: "E01.md",
              valid: true,
              errors: [],
            },
          ],
          summary: {
            total: 1,
            valid: 1,
            invalid: 0,
          },
        },
      })) as any;

      const result = await validateAfterWrite("docs/E01.md", mockRun, "/home/user/project");

      expect(result.ok).toBe(true);
      if (result.ok) {
        expect(result.summary.valid).toBe(1);
        expect(result.summary.invalid).toBe(0);
      }
    });

    it("fails validation when file is invalid", async () => {
      const mockRun = mock(async () => ({
        ok: true,
        data: {
          version: 1,
          kind: "rootline/batch-validation-result",
          results: [
            {
              path: "E01.md",
              valid: false,
              errors: [
                {
                  field: "estado",
                  rule: "enum",
                  message: "invalid enum value",
                },
              ],
            },
          ],
          summary: {
            total: 1,
            valid: 0,
            invalid: 1,
          },
        },
      })) as any;

      const result = await validateAfterWrite("docs/E01.md", mockRun, "/home/user/project");

      expect(result.ok).toBe(false);
      if (!result.ok) {
        expect(result.errors.length).toBeGreaterThan(0);
      }
    });

    it("handles runner execution errors", async () => {
      const mockRun = mock(async () => ({
        ok: false,
        error: {
          code: "exec_failed",
          message: "rootline validate failed",
          stderr: "schema error",
        },
      })) as any;

      const result = await validateAfterWrite("docs/E01.md", mockRun, "/home/user/project");

      expect(result.ok).toBe(false);
      if (!result.ok) {
        expect(result.errors.length).toBeGreaterThan(0);
        const firstError = result.errors[0];
        expect(typeof firstError === "object").toBe(true);
      }
    });

    it("passes when multiple files are validated and all pass", async () => {
      const mockRun = mock(async () => ({
        ok: true,
        data: {
          version: 1,
          kind: "rootline/batch-validation-result",
          results: [
            {
              path: "E01.md",
              valid: true,
              errors: [],
            },
            {
              path: "F01.md",
              valid: true,
              errors: [],
            },
          ],
          summary: {
            total: 2,
            valid: 2,
            invalid: 0,
          },
        },
      })) as any;

      const result = await validateAfterWrite("docs/E01.md", mockRun, "/home/user/project");

      expect(result.ok).toBe(true);
      if (result.ok) {
        expect(result.summary.valid).toBe(2);
        expect(result.summary.invalid).toBe(0);
      }
    });

    it("fails when one of multiple files is invalid", async () => {
      const mockRun = mock(async () => ({
        ok: true,
        data: {
          version: 1,
          kind: "rootline/batch-validation-result",
          results: [
            {
              path: "E01.md",
              valid: true,
              errors: [],
            },
            {
              path: "E01-invalid.md",
              valid: false,
              errors: [
                {
                  field: "tipo",
                  rule: "required",
                  message: "field is required",
                },
              ],
            },
          ],
          summary: {
            total: 2,
            valid: 1,
            invalid: 1,
          },
        },
      })) as any;

      // Test with a file that matches one of the invalid results
      const result = await validateAfterWrite("docs/E01-invalid.md", mockRun, "/home/user/project");

      expect(result.ok).toBe(false);
      if (!result.ok) {
        expect(result.errors.length).toBeGreaterThan(0);
      }
    });
  });

  describe("integration scenarios", () => {
    it("combines path validation with confirmation check for safe write", () => {
      // Scenario: user wants to write to docs/roadmap/E01.md in non-interactive mode
      const pathCheck = validateTargetPath("docs/roadmap/E01.md", "/home/user/project");
      const confirmCheck = requiresConfirmation("create", false);

      expect(pathCheck.ok).toBe(true);
      expect(confirmCheck).toBe(true); // Must have confirmation in non-interactive
    });

    it("rejects unsafe path even in interactive mode", () => {
      // Scenario: even with interactive mode, path escapes are rejected
      const pathCheck = validateTargetPath("../../../etc/passwd", "/home/user/project");
      const confirmCheck = requiresConfirmation("create", true);

      expect(pathCheck.ok).toBe(false); // Path escape is always rejected
      expect(confirmCheck).toBe(false); // But confirmation would be skipped
    });

    it("allows safe paths with confirmation in non-interactive mode", () => {
      // Scenario: safe write to approved directory in non-interactive mode
      const pathCheck = validateTargetPath("docs/roadmap/F01.md", "/home/user/project");
      const confirmCheck = requiresConfirmation("update", false);

      expect(pathCheck.ok).toBe(true);
      expect(confirmCheck).toBe(true); // Requires explicit confirmation
    });
  });
});
