package main

import (
	"errors"
	"testing"
)

func TestApplyExitError(t *testing.T) {
	tests := []struct {
		name       string
		errCount   int
		rolledBack int
		wantMsg    string // "" means the run succeeded and no error is expected
	}{
		{
			name: "a run with nothing to report succeeds",
		},
		{
			name:     "a single error fails and is named in the singular",
			errCount: 1,
			wantMsg:  "apply failed: 1 error",
		},
		{
			name:     "several errors are named in the plural",
			errCount: 3,
			wantMsg:  "apply failed: 3 errors",
		},
		{
			// The disjunct that matters: after the per-file rollback landed, a
			// successful revert leaves errors[] empty. A rule reading only
			// errCount would call this run a success.
			name:       "a rollback alone fails even with no errors",
			rolledBack: 1,
			wantMsg:    "apply failed: 1 file rolled back",
		},
		{
			name:       "several rollbacks are named in the plural",
			rolledBack: 2,
			wantMsg:    "apply failed: 2 files rolled back",
		},
		{
			name:       "both counts are reported together",
			errCount:   2,
			rolledBack: 1,
			wantMsg:    "apply failed: 2 errors, 1 file rolled back",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := applyExitError(tt.errCount, tt.rolledBack)

			if tt.wantMsg == "" {
				if err != nil {
					t.Fatalf("applyExitError(%d, %d) = %v; want nil", tt.errCount, tt.rolledBack, err)
				}
				return
			}

			if err == nil {
				t.Fatalf("applyExitError(%d, %d) = nil; want %q", tt.errCount, tt.rolledBack, tt.wantMsg)
			}
			if !errors.Is(err, errApplyFailure) {
				t.Errorf("error does not wrap errApplyFailure, so callers cannot match on it: %v", err)
			}
			if got := err.Error(); got != tt.wantMsg {
				t.Errorf("message = %q; want %q", got, tt.wantMsg)
			}
		})
	}
}
