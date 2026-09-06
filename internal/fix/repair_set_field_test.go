package fix

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/proposal"
)

func TestApplyRepairSetFieldStrictEquality(t *testing.T) {
	for _, tt := range []struct {
		name, yaml, value, want string
		skipped                 int
	}{
		{"same string", "alpha: new", "new", "alpha: new", 1},
		{"boolean is not string", "alpha: true", "true", "alpha: \"true\"", 0},
		{"integer is not string", "alpha: 1", "1", "alpha: \"1\"", 0},
		{"missing is not empty string", "other: kept", "", "other: kept\nalpha: \"\"", 0},
	} {
		for _, dryRun := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/dry=%v", tt.name, dryRun), func(t *testing.T) {
				before := "---\n" + tt.yaml + "\n---\n# Body\n"
				root, rel, path, stamp := writeFixedRepairDoc(t, before)
				result, err := ApplyRepair([]proposal.Proposal{{Type: proposal.SetField, Field: "alpha", Value: tt.value, Paths: []string{rel}}}, dryRun, root, false)
				if err != nil || !result.Complete || len(result.Errors) != 0 || len(result.Skipped) != tt.skipped || len(result.Changed) != 1-tt.skipped {
					t.Fatalf("result=%+v err=%v", result, err)
				}
				if dryRun || tt.skipped == 1 {
					assertUnchangedRepairDoc(t, path, before, stamp)
				} else {
					got, err := os.ReadFile(filepath.Join(root, rel))
					if err != nil || string(got) != "---\n"+tt.want+"\n---\n# Body\n" {
						t.Fatalf("unexpected rewrite: %q, err=%v", got, err)
					}
				}
			})
		}
	}
}
