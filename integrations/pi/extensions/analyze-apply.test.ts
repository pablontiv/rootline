import { describe, it, expect, mock, spyOn } from "bun:test";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import analyzeApplyTools from "./analyze-apply";

/**
 * Tests for rootline-analyze and rootline-apply extensions.
 *
 * Tests verify:
 * - Tool registration with correct schema
 * - Path validation guardrails (blocks path traversal)
 * - Analyze: read-only operation with no confirmation requirement
 * - Apply: requires confirmation in non-interactive mode when not dry-run
 * - Apply dry-run: returns preview without confirmation
 * - Apply success: runs post-apply validation
 * - Error handling: validation errors, missing reports
 */

describe("rootline-analyze and rootline-apply extensions", () => {
  describe("tool registration", () => {
    it("registers rootline-analyze with correct description", () => {
      const registerToolMock = mock(() => {});
      const pi = {
        exec: mock(async () => ({
          stdout: "{}",
          stderr: "",
          code: 0,
        })),
        registerTool: registerToolMock,
      } as unknown as ExtensionAPI;

      analyzeApplyTools(pi);

      expect(registerToolMock).toHaveBeenCalledTimes(2);

      // Check analyze tool
      const analyzeCall = registerToolMock.mock.calls[0];
      const analyzeTool = analyzeCall[0];
      expect(analyzeTool.name).toBe("rootline-analyze");
      expect(analyzeTool.description).toContain("Analyze documents");
      expect(analyzeTool.input_schema.required).toEqual(["path"]);
      expect(analyzeTool.input_schema.properties).toHaveProperty("incremental");
      expect(analyzeTool.input_schema.properties).toHaveProperty("threshold");
    });

    it("registers rootline-apply with correct description", () => {
      const registerToolMock = mock(() => {});
      const pi = {
        exec: mock(async () => ({
          stdout: "{}",
          stderr: "",
          code: 0,
        })),
        registerTool: registerToolMock,
      } as unknown as ExtensionAPI;

      analyzeApplyTools(pi);

      const applyCall = registerToolMock.mock.calls[1];
      const applyTool = applyCall[0];
      expect(applyTool.name).toBe("rootline-apply");
      expect(applyTool.description).toContain("Apply schema and/or repair");
      expect(applyTool.input_schema.required).toEqual(["report_path", "mode"]);
      expect(applyTool.input_schema.properties).toHaveProperty("dry_run");
      expect(applyTool.input_schema.properties).toHaveProperty("confirmed");
    });
  });

  describe("rootline-analyze", () => {
    describe("path validation", () => {
      it("rejects path traversal attacks", async () => {
        const execMock = mock(async () => ({
          stdout: "{}",
          stderr: "",
          code: 0,
        }));

        let analyzeExecute: Function | null = null;
        const pi = {
          exec: execMock,
          registerTool: (def: any) => {
            if (def.name === "rootline-analyze") {
              analyzeExecute = def.execute;
            }
          },
        } as unknown as ExtensionAPI;

        analyzeApplyTools(pi);

        // Should reject path that escapes cwd
        const result = await analyzeExecute?.({
          path: "../../../../etc/passwd",
        });

        expect(result.analyzed).toBe(false);
        expect(result.error).toContain("escapes project root");
        expect(execMock).not.toHaveBeenCalled();
      });

      it("accepts relative paths within project", async () => {
        const execMock = mock(async () => ({
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/analyze-report",
            path: "/home/project/docs",
            categories: [],
            summary: { total_inferences: 0, agent_required: 0, engine_resolved: 0 },
          }),
          stderr: "",
          code: 0,
        }));

        let analyzeExecute: Function | null = null;
        const pi = {
          exec: execMock,
          registerTool: (def: any) => {
            if (def.name === "rootline-analyze") {
              analyzeExecute = def.execute;
            }
          },
        } as unknown as ExtensionAPI;

        analyzeApplyTools(pi);

        const result = await analyzeExecute?.({
          path: "docs",
        });

        expect(execMock).toHaveBeenCalled();
      });
    });

    describe("analyze success", () => {
      it("runs analyze and returns full report", async () => {
        const mockReport = {
          version: 1,
          kind: "rootline/analyze-report",
          path: "/home/project/docs",
          incremental: false,
          categories: [
            {
              id: "data_inference",
              name: "Data Inference",
              inference_count: 2,
              inferences: [
                {
                  type: "field_type_inferred",
                  source: "docs/O06/",
                  field: "estado",
                  message: "Field type inferred as string",
                  requires_agent: false,
                },
                {
                  type: "enum_values_inferred",
                  source: "docs/O06/",
                  field: "status",
                  message: "Enum values: ['Pending', 'In Progress', 'Done']",
                  requires_agent: false,
                },
              ],
            },
          ],
          summary: {
            total_inferences: 2,
            agent_required: 0,
            engine_resolved: 2,
          },
        };

        const execMock = mock(async () => ({
          stdout: JSON.stringify(mockReport),
          stderr: "",
          code: 0,
        }));

        let analyzeExecute: Function | null = null;
        const pi = {
          exec: execMock,
          registerTool: (def: any) => {
            if (def.name === "rootline-analyze") {
              analyzeExecute = def.execute;
            }
          },
        } as unknown as ExtensionAPI;

        analyzeApplyTools(pi);

        const result = await analyzeExecute?.({
          path: "docs",
          incremental: false,
        });

        expect(result.analyzed).toBe(true);
        expect(result.report).toBeDefined();
        expect(result.report?.version).toBe(1);
        expect(result.report?.kind).toBe("rootline/analyze-report");
        expect(result.report?.categories).toHaveLength(1);
        expect(result.report?.categories?.[0].inferences).toHaveLength(2);

        // Verify analyze was called with correct args
        const callArgs = execMock.mock.calls[0];
        expect(callArgs[1]).toContain("analyze");
        expect(callArgs[1]).toContain("--output");
        expect(callArgs[1]).toContain("json");
      });

      it("passes incremental flag to analyze command", async () => {
        const execMock = mock(async () => ({
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/analyze-report",
            path: "/home/project/docs",
            incremental: true,
            categories: [],
            summary: { total_inferences: 0, agent_required: 0, engine_resolved: 0 },
          }),
          stderr: "",
          code: 0,
        }));

        let analyzeExecute: Function | null = null;
        const pi = {
          exec: execMock,
          registerTool: (def: any) => {
            if (def.name === "rootline-analyze") {
              analyzeExecute = def.execute;
            }
          },
        } as unknown as ExtensionAPI;

        analyzeApplyTools(pi);

        await analyzeExecute?.({
          path: "docs",
          incremental: true,
        });

        const callArgs = execMock.mock.calls[0];
        expect(callArgs[1]).toContain("--incremental");
      });

      it("passes threshold flag to analyze command", async () => {
        const execMock = mock(async () => ({
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/analyze-report",
            path: "/home/project/docs",
            categories: [],
            summary: { total_inferences: 0, agent_required: 0, engine_resolved: 0 },
          }),
          stderr: "",
          code: 0,
        }));

        let analyzeExecute: Function | null = null;
        const pi = {
          exec: execMock,
          registerTool: (def: any) => {
            if (def.name === "rootline-analyze") {
              analyzeExecute = def.execute;
            }
          },
        } as unknown as ExtensionAPI;

        analyzeApplyTools(pi);

        await analyzeExecute?.({
          path: "docs",
          threshold: 0.75,
        });

        const callArgs = execMock.mock.calls[0];
        expect(callArgs[1]).toContain("--threshold");
        expect(callArgs[1]).toContain("0.75");
      });

      it("does not require confirmation (read-only operation)", async () => {
        const execMock = mock(async () => ({
          stdout: JSON.stringify({
            version: 1,
            kind: "rootline/analyze-report",
            path: "/home/project/docs",
            categories: [],
            summary: { total_inferences: 0, agent_required: 0, engine_resolved: 0 },
          }),
          stderr: "",
          code: 0,
        }));

        let analyzeExecute: Function | null = null;
        const pi = {
          exec: execMock,
          registerTool: (def: any) => {
            if (def.name === "rootline-analyze") {
              analyzeExecute = def.execute;
            }
          },
        } as unknown as ExtensionAPI;

        analyzeApplyTools(pi);

        // Should work without confirmed flag
        const result = await analyzeExecute?.({
          path: "docs",
          interactive: false,
          confirmed: false,
        });

        expect(result.analyzed).toBe(true);
        expect(execMock).toHaveBeenCalled();
      });
    });

    describe("analyze error handling", () => {
      it("handles analyze command failure", async () => {
        const execMock = mock(async () => ({
          stdout: "",
          stderr: "directory not found",
          code: 1,
        }));

        let analyzeExecute: Function | null = null;
        const pi = {
          exec: execMock,
          registerTool: (def: any) => {
            if (def.name === "rootline-analyze") {
              analyzeExecute = def.execute;
            }
          },
        } as unknown as ExtensionAPI;

        analyzeApplyTools(pi);

        const result = await analyzeExecute?.({
          path: "docs",
        });

        expect(result.analyzed).toBe(false);
        expect(result.error).toBeDefined();
        expect(result.error).toContain("Failed to analyze");
      });
    });
  });

  describe("rootline-apply", () => {
    describe("mode validation", () => {
      it("rejects invalid mode", async () => {
        const execMock = mock(async () => ({
          stdout: "{}",
          stderr: "",
          code: 0,
        }));

        let applyExecute: Function | null = null;
        const pi = {
          exec: execMock,
          registerTool: (def: any) => {
            if (def.name === "rootline-apply") {
              applyExecute = def.execute;
            }
          },
        } as unknown as ExtensionAPI;

        analyzeApplyTools(pi);

        const result = await applyExecute?.({
          report_path: "report.json",
          mode: "invalid",
        });

        expect(result.applied).toBe(false);
        expect(result.error).toContain("Invalid mode");
        expect(execMock).not.toHaveBeenCalled();
      });
    });

    describe("path validation", () => {
      it("rejects path traversal in report_path", async () => {
        const execMock = mock(async () => ({
          stdout: "{}",
          stderr: "",
          code: 0,
        }));

        let applyExecute: Function | null = null;
        const pi = {
          exec: execMock,
          registerTool: (def: any) => {
            if (def.name === "rootline-apply") {
              applyExecute = def.execute;
            }
          },
        } as unknown as ExtensionAPI;

        analyzeApplyTools(pi);

        const result = await applyExecute?.({
          report_path: "../../../../etc/passwd",
          mode: "schema",
        });

        expect(result.applied).toBe(false);
        expect(result.error).toContain("escapes project root");
        expect(execMock).not.toHaveBeenCalled();
      });
    });

    describe("apply dry-run", () => {
      it("returns preview without confirmation requirement", async () => {
        const mockSchemaResult = {
          version: 1,
          kind: "rootline/schema-apply",
          dry_run: true,
          path: "/home/project/docs",
          results: [
            {
              proposal_type: "extend_enum",
              source: "docs/.stem",
              field: "estado",
              action: "Add 'On Hold' to enum",
              status: "would_apply",
              affected_stem_file: "docs/.stem",
            },
          ],
          summary: {
            total_proposals_in_report: 1,
            applied_count: 1,
            skipped_count: 0,
            requires_agent_count: 0,
          },
          validation: {
            valid: true,
            schema_conflicts: 0,
            stem_health_warnings: [],
          },
        };

        const execMock = mock(async () => ({
          stdout: JSON.stringify(mockSchemaResult),
          stderr: "",
          code: 0,
        }));

        let applyExecute: Function | null = null;
        const pi = {
          exec: execMock,
          registerTool: (def: any) => {
            if (def.name === "rootline-apply") {
              applyExecute = def.execute;
            }
          },
        } as unknown as ExtensionAPI;

        analyzeApplyTools(pi);

        // Should work without confirmed flag in dry-run
        const result = await applyExecute?.({
          report_path: "report.json",
          mode: "schema",
          dry_run: true,
          interactive: false,
          confirmed: false,
        });

        expect(result.applied).toBe(true);
        expect(result.dry_run).toBe(true);
        expect(result.schema_result).toBeDefined();
        expect(execMock).toHaveBeenCalled();

        // Verify --dry-run was passed
        const callArgs = execMock.mock.calls[0];
        expect(callArgs[1]).toContain("--dry-run");
      });
    });

    describe("apply confirmation requirement", () => {
      it("requires confirmation in non-interactive mode when applying (not dry-run)", async () => {
        const execMock = mock(async () => ({
          stdout: "{}",
          stderr: "",
          code: 0,
        }));

        let applyExecute: Function | null = null;
        const pi = {
          exec: execMock,
          registerTool: (def: any) => {
            if (def.name === "rootline-apply") {
              applyExecute = def.execute;
            }
          },
        } as unknown as ExtensionAPI;

        analyzeApplyTools(pi);

        // Should reject without confirmation when not dry-run
        const result = await applyExecute?.({
          report_path: "report.json",
          mode: "schema",
          dry_run: false,
          interactive: false,
          confirmed: false,
        });

        expect(result.applied).toBe(false);
        expect(result.error).toContain("Confirmation required");
        expect(execMock).not.toHaveBeenCalled();
      });

      it("allows apply with explicit confirmed flag", async () => {
        const mockSchemaResult = {
          version: 1,
          kind: "rootline/schema-apply",
          dry_run: false,
          path: "/home/project/docs",
          results: [],
          summary: {
            total_proposals_in_report: 0,
            applied_count: 0,
            skipped_count: 0,
            requires_agent_count: 0,
          },
          validation: { valid: true },
        };

        const execMock = mock(async () => ({
          stdout: JSON.stringify(mockSchemaResult),
          stderr: "",
          code: 0,
        }));

        let applyExecute: Function | null = null;
        const pi = {
          exec: execMock,
          registerTool: (def: any) => {
            if (def.name === "rootline-apply") {
              applyExecute = def.execute;
            }
          },
        } as unknown as ExtensionAPI;

        analyzeApplyTools(pi);

        const result = await applyExecute?.({
          report_path: "report.json",
          mode: "schema",
          dry_run: false,
          interactive: false,
          confirmed: true,
        });

        expect(result.applied).toBe(true);
        expect(execMock).toHaveBeenCalled();

        // Verify --dry-run was NOT passed
        const callArgs = execMock.mock.calls[0];
        expect(callArgs[1]).not.toContain("--dry-run");
      });
    });

    describe("apply success", () => {
      it("applies schema proposals and returns result", async () => {
        const mockSchemaResult = {
          version: 1,
          kind: "rootline/schema-apply",
          dry_run: false,
          path: "/home/project/docs",
          results: [
            {
              proposal_type: "extend_enum",
              source: "docs/.stem",
              field: "estado",
              action: "Add 'On Hold' to enum",
              status: "applied",
              affected_stem_file: "docs/.stem",
            },
          ],
          summary: {
            total_proposals_in_report: 1,
            applied_count: 1,
            skipped_count: 0,
            requires_agent_count: 0,
          },
          validation: {
            valid: true,
            schema_conflicts: 0,
            stem_health_warnings: [],
          },
        };

        const execMock = mock(async () => ({
          stdout: JSON.stringify(mockSchemaResult),
          stderr: "",
          code: 0,
        }));

        let applyExecute: Function | null = null;
        const pi = {
          exec: execMock,
          registerTool: (def: any) => {
            if (def.name === "rootline-apply") {
              applyExecute = def.execute;
            }
          },
        } as unknown as ExtensionAPI;

        analyzeApplyTools(pi);

        const result = await applyExecute?.({
          report_path: "report.json",
          mode: "schema",
          dry_run: false,
          confirmed: true,
        });

        expect(result.applied).toBe(true);
        expect(result.mode).toBe("schema");
        expect(result.schema_result).toBeDefined();
        expect(result.schema_result?.summary?.applied_count).toBe(1);
      });

      it("applies repair proposals and returns result", async () => {
        const mockRepairResult = {
          version: 1,
          kind: "rootline/repair-apply",
          dry_run: false,
          report_source: "report.json",
          path: "/home/project/docs",
          results: [
            {
              path: "docs/O06/T001.md",
              status: "repaired",
              changes: [
                {
                  field: "estado",
                  old_value: "Completd",
                  new_value: "Completed",
                  reason: "Enum correction (typo)",
                },
              ],
            },
          ],
          summary: {
            total_files: 1,
            would_repair: 1,
            skipped: 0,
            total_fields_changed: 1,
            fields_added: 0,
            values_corrected: 1,
          },
          repair_surface_count: 1,
          validation_summary: {
            valid: true,
            total_files: 1,
            error_count: 0,
            warning_count: 0,
          },
        };

        const execMock = mock(async () => ({
          stdout: JSON.stringify(mockRepairResult),
          stderr: "",
          code: 0,
        }));

        let applyExecute: Function | null = null;
        const pi = {
          exec: execMock,
          registerTool: (def: any) => {
            if (def.name === "rootline-apply") {
              applyExecute = def.execute;
            }
          },
        } as unknown as ExtensionAPI;

        analyzeApplyTools(pi);

        const result = await applyExecute?.({
          report_path: "report.json",
          mode: "repair",
          dry_run: false,
          confirmed: true,
        });

        expect(result.applied).toBe(true);
        expect(result.mode).toBe("repair");
        expect(result.repair_result).toBeDefined();
        expect(result.repair_result?.summary?.total_fields_changed).toBe(1);
      });

      it("applies both schema and repair when mode is 'both'", async () => {
        const mockSchemaResult = {
          version: 1,
          kind: "rootline/schema-apply",
          dry_run: false,
          results: [],
          summary: {
            total_proposals_in_report: 0,
            applied_count: 0,
            skipped_count: 0,
            requires_agent_count: 0,
          },
        };

        const mockRepairResult = {
          version: 1,
          kind: "rootline/repair-apply",
          dry_run: false,
          results: [],
          summary: {
            total_files: 0,
            total_fields_changed: 0,
          },
        };

        let callCount = 0;
        const execMock = mock(async () => {
          callCount++;
          if (callCount === 1) {
            return {
              stdout: JSON.stringify(mockSchemaResult),
              stderr: "",
              code: 0,
            };
          } else {
            return {
              stdout: JSON.stringify(mockRepairResult),
              stderr: "",
              code: 0,
            };
          }
        });

        let applyExecute: Function | null = null;
        const pi = {
          exec: execMock,
          registerTool: (def: any) => {
            if (def.name === "rootline-apply") {
              applyExecute = def.execute;
            }
          },
        } as unknown as ExtensionAPI;

        analyzeApplyTools(pi);

        const result = await applyExecute?.({
          report_path: "report.json",
          mode: "both",
          dry_run: true,
          confirmed: true,
        });

        expect(result.applied).toBe(true);
        expect(result.mode).toBe("both");
        expect(result.schema_result).toBeDefined();
        expect(result.repair_result).toBeDefined();
        expect(execMock).toHaveBeenCalledTimes(2);
      });
    });

    describe("apply error handling", () => {
      it("handles schema apply command failure", async () => {
        const execMock = mock(async () => ({
          stdout: "",
          stderr: "invalid report format",
          code: 1,
        }));

        let applyExecute: Function | null = null;
        const pi = {
          exec: execMock,
          registerTool: (def: any) => {
            if (def.name === "rootline-apply") {
              applyExecute = def.execute;
            }
          },
        } as unknown as ExtensionAPI;

        analyzeApplyTools(pi);

        const result = await applyExecute?.({
          report_path: "report.json",
          mode: "schema",
          dry_run: true,
          confirmed: true,
        });

        expect(result.applied).toBe(false);
        expect(result.error).toContain("Schema apply failed");
      });

      it("handles repair apply command failure", async () => {
        const execMock = mock(async () => ({
          stdout: "",
          stderr: "no repair surface proposals found",
          code: 1,
        }));

        let applyExecute: Function | null = null;
        const pi = {
          exec: execMock,
          registerTool: (def: any) => {
            if (def.name === "rootline-apply") {
              applyExecute = def.execute;
            }
          },
        } as unknown as ExtensionAPI;

        analyzeApplyTools(pi);

        const result = await applyExecute?.({
          report_path: "report.json",
          mode: "repair",
          dry_run: true,
          confirmed: true,
        });

        expect(result.applied).toBe(false);
        expect(result.error).toContain("Repair apply failed");
      });
    });
  });
});
