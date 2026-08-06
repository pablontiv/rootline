package main

import (
	"path/filepath"
	"testing"
)

// TestResolveReportRoot tests the four-level precedence chain for report root resolution.
func TestResolveReportRoot(t *testing.T) {
	tests := []struct {
		name           string
		explicitRoot   string
		reportRoot     string
		reportPath     string
		reportFilePath string
		expected       string
		expectError    bool
	}{
		{
			name:           "precedence: explicit root wins",
			explicitRoot:   "/explicit",
			reportRoot:     "/report/root",
			reportPath:     "relative/path",
			reportFilePath: "/some/where/report.json",
			expected:       "/explicit",
			expectError:    false,
		},
		{
			name:           "precedence: report root second",
			explicitRoot:   "",
			reportRoot:     "/report/root",
			reportPath:     "relative/path",
			reportFilePath: "/some/where/report.json",
			expected:       "/report/root",
			expectError:    false,
		},
		{
			name:           "precedence: report path made absolute third",
			explicitRoot:   "",
			reportRoot:     "",
			reportPath:     "relative/path",
			reportFilePath: "/some/where/report.json",
			expected:       func() string { abs, _ := filepath.Abs("relative/path"); return abs }(),
			expectError:    false,
		},
		{
			name:           "precedence: report file directory fourth (legacy fallback)",
			explicitRoot:   "",
			reportRoot:     "",
			reportPath:     "",
			reportFilePath: "/some/where/report.json",
			expected:       "/some/where",
			expectError:    false,
		},
		{
			// A relative --report argument must still yield an absolute root.
			// The behaviour this rung preserves was filepath.Abs(filepath.Dir(...)),
			// so dropping the Abs here would be a regression, not a preservation.
			name:           "legacy: a relative report path still resolves absolute",
			explicitRoot:   "",
			reportRoot:     "",
			reportPath:     "",
			reportFilePath: "report.json",
			expected:       func() string { abs, _ := filepath.Abs("."); return abs }(),
			expectError:    false,
		},
		{
			name:           "absolute reportPath is used as-is",
			explicitRoot:   "",
			reportRoot:     "",
			reportPath:     "/absolute/scan/root",
			reportFilePath: "/some/where/report.json",
			expected:       "/absolute/scan/root",
			expectError:    false,
		},
		{
			name:           "explicit root overrides absolute reportPath",
			explicitRoot:   "/override",
			reportRoot:     "",
			reportPath:     "/absolute/scan/root",
			reportFilePath: "/some/where/report.json",
			expected:       "/override",
			expectError:    false,
		},
		{
			name:           "explicit root overrides reportRoot",
			explicitRoot:   "/override",
			reportRoot:     "/report/root",
			reportPath:     "/absolute/scan/root",
			reportFilePath: "/some/where/report.json",
			expected:       "/override",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolveReportRoot(tt.explicitRoot, tt.reportRoot, tt.reportPath, tt.reportFilePath)

			if tt.expectError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError && result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
