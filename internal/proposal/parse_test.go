package proposal

import (
	"reflect"
	"testing"
)

func TestParseBlockingInfo(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantBase    string
		wantTargets []string
		wantNotes   []string
	}{
		{
			name:        "standard compound",
			input:       "Pending (blocked by T001 + E04/F05 + human)",
			wantBase:    "Pending",
			wantTargets: []string{"T001", "E04/F05"},
			wantNotes:   []string{"human"},
		},
		{
			name:        "simple single target",
			input:       "In Progress (blocked by T001)",
			wantBase:    "In Progress",
			wantTargets: []string{"T001"},
			wantNotes:   nil,
		},
		{
			name:        "multiple ID-like targets",
			input:       "Blocked (blocked by T001 + T002 + T003)",
			wantBase:    "Blocked",
			wantTargets: []string{"T001", "T002", "T003"},
			wantNotes:   nil,
		},
		{
			name:        "all free-text notes",
			input:       "Pending (blocked by human + review + approval)",
			wantBase:    "Pending",
			wantTargets: nil,
			wantNotes:   []string{"human", "review", "approval"},
		},
		{
			name:        "no blocking info",
			input:       "Pending",
			wantBase:    "Pending",
			wantTargets: nil,
			wantNotes:   nil,
		},
		{
			name:        "parentheses without blocked by",
			input:       "Pending (waiting for review)",
			wantBase:    "Pending (waiting for review)",
			wantTargets: nil,
			wantNotes:   nil,
		},
		{
			name:        "empty input",
			input:       "",
			wantBase:    "",
			wantTargets: nil,
			wantNotes:   nil,
		},
		{
			name:        "empty blocked by content",
			input:       "Pending (blocked by )",
			wantBase:    "Pending (blocked by )",
			wantTargets: nil,
			wantNotes:   nil,
		},
		{
			name:        "blocked by with only empty parts",
			input:       "Pending (blocked by   +   )",
			wantBase:    "Pending",
			wantTargets: nil,
			wantNotes:   nil,
		},
		{
			name:        "complex targets and mixed case",
			input:       "Status (blocked by T123 + E01-F02 + some note)",
			wantBase:    "Status",
			wantTargets: []string{"T123", "E01-F02"},
			wantNotes:   []string{"some note"},
		},
		{
			name:        "multiple blocked by (regex non-greedy match)",
			input:       "Multiple (blocked by T001) (blocked by T002)",
			wantBase:    "Multiple",
			wantTargets: nil,
			wantNotes:   []string{"T001) (blocked by T002"},
		},
		{
			name:        "path-like target",
			input:       "Pending (blocked by E04/F01)",
			wantBase:    "Pending",
			wantTargets: []string{"E04/F01"},
			wantNotes:   nil,
		},
		{
			name:        "base with extra spaces",
			input:       "  In Progress   (blocked by T001)",
			wantBase:    "In Progress",
			wantTargets: []string{"T001"},
			wantNotes:   nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotBase, gotTargets, gotNotes := ParseBlockingInfo(tc.input)
			if gotBase != tc.wantBase {
				t.Errorf("base = %q, want %q", gotBase, tc.wantBase)
			}
			if !reflect.DeepEqual(gotTargets, tc.wantTargets) {
				t.Errorf("targets = %v, want %v", gotTargets, tc.wantTargets)
			}
			if !reflect.DeepEqual(gotNotes, tc.wantNotes) {
				t.Errorf("notes = %v, want %v", gotNotes, tc.wantNotes)
			}
		})
	}
}
