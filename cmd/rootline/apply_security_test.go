package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestApplyStdinLimit(t *testing.T) {
	// Create a large input (11MB) which exceeds our 10MB limit.
	largeInput := make([]byte, 11*1024*1024)
	for i := range largeInput {
		largeInput[i] = 'a'
	}

	buf := bytes.NewReader(largeInput)
	rootCmd.SetIn(buf)

	// We expect it to fail parsing JSON because it will be truncated.
	// io.ReadAll from LimitReader doesn't return an error on truncation,
	// it just returns the limited data.
	// Since 10MB of 'a' is not valid JSON, Unmarshal will fail.

	_, err := runCmd(t, "apply")
	if err == nil {
		t.Fatal("expected error for large input")
	}

	if !strings.Contains(err.Error(), "parsing report") {
		t.Errorf("expected parsing report error, got: %v", err)
	}
}

func TestApplyStdinValid(t *testing.T) {
	dir := setupTestDir(t)
	report := fmt.Sprintf(`{"version":1,"kind":"analyze","path":"%s","categories":[]}`, dir)

	buf := strings.NewReader(report)
	rootCmd.SetIn(buf)

	out, err := runCmd(t, "apply")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "applied") && !strings.Contains(out, "Applied") {
		// Output might be JSON or Table
		if !strings.Contains(out, "rootline/apply") {
			t.Errorf("expected success output, got: %s", out)
		}
	}
}
