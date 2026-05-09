import { describe, it, expect, mock } from "bun:test";
import { createRunner, type RunnerOptions } from "./runner";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

/**
 * Unit tests for the rootline CLI runner.
 *
 * These tests verify core error handling paths using mocked pi.exec:
 * - Success path with valid JSON
 * - JSON parse failure
 * - Non-zero exit code handling
 * - Timeout detection
 * - Abort signal handling
 *
 * Integration tests (with actual rootline binary) are deferred to T005.
 */

describe("createRunner", () => {
  describe("success path", () => {
    it("returns data when rootline exits with code 0 and valid JSON", async () => {
      const mockExec = mock(async () => ({
        stdout: JSON.stringify({ status: "ok", results: [{ name: "test" }] }),
        stderr: "",
        code: 0,
      }));

      const pi = { exec: mockExec } as unknown as ExtensionAPI;
      const run = createRunner(pi);

      const result = await run(["query", "--output", "json"], {
        cwd: "/repo",
        timeout: 5000,
      });

      expect(result.ok).toBe(true);
      if (result.ok) {
        expect(result.data).toEqual({
          status: "ok",
          results: [{ name: "test" }],
        });
      }
      expect(mockExec).toHaveBeenCalledWith("rootline", ["query", "--output", "json"], {
        cwd: "/repo",
        timeout: 5000,
      });
    });

    it("uses default timeout if not specified", async () => {
      const mockExec = mock(async () => ({
        stdout: JSON.stringify({}),
        stderr: "",
        code: 0,
      }));

      const pi = { exec: mockExec } as unknown as ExtensionAPI;
      const run = createRunner(pi);

      await run(["validate"]);

      expect(mockExec).toHaveBeenCalledWith("rootline", ["validate"], {
        cwd: undefined,
        timeout: 10_000,
      });
    });
  });

  describe("JSON parse error", () => {
    it("returns json_parse_failed when stdout is not valid JSON", async () => {
      const mockExec = mock(async () => ({
        stdout: "this is not json\nit's a shell error message",
        stderr: "",
        code: 0,
      }));

      const pi = { exec: mockExec } as unknown as ExtensionAPI;
      const run = createRunner(pi);

      const result = await run(["describe"]);

      expect(result.ok).toBe(false);
      if (!result.ok) {
        expect(result.error.code).toBe("json_parse_failed");
        expect(result.error.message).toContain("not valid JSON");
        expect(result.error.stderr).toContain("not json");
      }
    });
  });

  describe("exit code error", () => {
    it("returns exec_failed when rootline exits with non-zero code", async () => {
      const mockExec = mock(async () => ({
        stdout: "",
        stderr: "directory not found: /invalid/path",
        code: 1,
      }));

      const pi = { exec: mockExec } as unknown as ExtensionAPI;
      const run = createRunner(pi);

      const result = await run(["validate", "--all", "/invalid/path"]);

      expect(result.ok).toBe(false);
      if (!result.ok) {
        expect(result.error.code).toBe("exec_failed");
        expect(result.error.exitCode).toBe(1);
        expect(result.error.stderr).toContain("not found");
        expect(result.error.message).toContain("not found");
      }
    });

    it("uses stderr as message when available on non-zero exit", async () => {
      const mockExec = mock(async () => ({
        stdout: "",
        stderr: "schema validation failed",
        code: 2,
      }));

      const pi = { exec: mockExec } as unknown as ExtensionAPI;
      const run = createRunner(pi);

      const result = await run(["validate"]);

      expect(result.ok).toBe(false);
      if (!result.ok) {
        expect(result.error.message).toBe("schema validation failed");
        expect(result.error.exitCode).toBe(2);
      }
    });
  });

  describe("timeout handling", () => {
    it("returns timeout error when pi.exec throws timeout error", async () => {
      const mockExec = mock(async () => {
        const err = new Error("timeout: Command timed out after 5000ms");
        throw err;
      });

      const pi = { exec: mockExec } as unknown as ExtensionAPI;
      const run = createRunner(pi);

      const result = await run(["analyze", "--all"], { timeout: 5000 });

      expect(result.ok).toBe(false);
      if (!result.ok) {
        expect(result.error.code).toBe("timeout");
        expect(result.error.message).toContain("timed out");
        expect(result.error.message).toContain("5000ms");
      }
    });

    it("detects ETIMEDOUT in error message", async () => {
      const mockExec = mock(async () => {
        const err = new Error("ETIMEDOUT");
        throw err;
      });

      const pi = { exec: mockExec } as unknown as ExtensionAPI;
      const run = createRunner(pi);

      const result = await run(["query"], { timeout: 3000 });

      expect(result.ok).toBe(false);
      if (!result.ok) {
        expect(result.error.code).toBe("timeout");
      }
    });
  });

  describe("not found error", () => {
    it("returns not_found when rootline binary is missing", async () => {
      const mockExec = mock(async () => {
        throw new Error("ENOENT: rootline not found");
      });

      const pi = { exec: mockExec } as unknown as ExtensionAPI;
      const run = createRunner(pi);

      const result = await run(["--version"]);

      expect(result.ok).toBe(false);
      if (!result.ok) {
        expect(result.error.code).toBe("not_found");
        expect(result.error.message).toContain("not found");
      }
    });
  });

  describe("abort signal", () => {
    it("returns exec_failed when signal is aborted before execution", async () => {
      const mockExec = mock(async () => {
        throw new Error("Should not be called");
      });

      const abortController = new AbortController();
      abortController.abort();

      const pi = { exec: mockExec } as unknown as ExtensionAPI;
      const run = createRunner(pi);

      const result = await run(["query"], {
        signal: abortController.signal,
      });

      expect(result.ok).toBe(false);
      if (!result.ok) {
        expect(result.error.code).toBe("exec_failed");
        expect(result.error.message).toContain("aborted");
      }
      expect(mockExec).not.toHaveBeenCalled();
    });
  });

  describe("generic execution error", () => {
    it("returns exec_failed for unknown errors", async () => {
      const mockExec = mock(async () => {
        throw new Error("Unexpected error during execution");
      });

      const pi = { exec: mockExec } as unknown as ExtensionAPI;
      const run = createRunner(pi);

      const result = await run(["tree"]);

      expect(result.ok).toBe(false);
      if (!result.ok) {
        expect(result.error.code).toBe("exec_failed");
        expect(result.error.message).toContain("Unexpected");
      }
    });

    it("handles non-Error thrown objects", async () => {
      const mockExec = mock(async () => {
        throw "something weird happened";
      });

      const pi = { exec: mockExec } as unknown as ExtensionAPI;
      const run = createRunner(pi);

      const result = await run(["describe"]);

      expect(result.ok).toBe(false);
      if (!result.ok) {
        expect(result.error.code).toBe("exec_failed");
        expect(result.error.message).toBe("something weird happened");
      }
    });
  });

  describe("type safety", () => {
    it("properly types result data", async () => {
      interface QueryResult {
        records: Array<{ path: string; name: string }>;
        total: number;
      }

      const mockExec = mock(async () => ({
        stdout: JSON.stringify({
          records: [{ path: "docs/file.md", name: "file" }],
          total: 1,
        }),
        stderr: "",
        code: 0,
      }));

      const pi = { exec: mockExec } as unknown as ExtensionAPI;
      const run = createRunner(pi);

      const result = await run<QueryResult>(["query"]);

      expect(result.ok).toBe(true);
      if (result.ok) {
        const typed: QueryResult = result.data;
        expect(typed.total).toBe(1);
        expect(typed.records[0].path).toBe("docs/file.md");
      }
    });
  });
});
