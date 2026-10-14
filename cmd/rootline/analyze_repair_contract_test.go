package main

import (
	"strings"
	"testing"
)

func TestAnalyzeHelpDoesNotAdvertiseRepairApply(t *testing.T) {
	if strings.Contains(analyzeCmd.Long, "repair apply") {
		t.Fatalf("analyze reports are not repair proposal reports; help must not advertise repair apply:\n%s", analyzeCmd.Long)
	}
}

func TestRepairParentShowsHelp(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, args := range [][]string{
		{"repair"},
		{"repair", "--help"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			out, err := runCmd(t, args...)
			if err != nil {
				t.Fatalf("help-only repair parent returned an error: %v\n%s", err, out)
			}
			for _, want := range []string{
				"rootline repair [command]",
				"rootline repair apply --report <file>",
				"version 1 rootline/proposals",
				"rootline fix --all <dir> --dry-run -o json",
				"Available Commands:",
				"apply",
			} {
				if !strings.Contains(out, want) {
					t.Errorf("repair help missing %q:\n%s", want, out)
				}
			}
			if strings.Contains(out, "--report is required") {
				t.Errorf("repair parent ran apply preflight instead of showing help:\n%s", out)
			}
		})
	}
}
