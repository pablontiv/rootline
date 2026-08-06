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
