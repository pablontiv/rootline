package fix

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pablontiv/rootline/internal/proposal"
)

func correctLinkProposal(paths ...string) proposal.Proposal {
	return proposal.Proposal{
		Type: proposal.CorrectLink, Field: "links", Paths: paths,
		From: "[[old]]", To: "[[new]]",
	}
}

func stampCorrectLinkTarget(t *testing.T, path string) time.Time {
	t.Helper()
	stamp := time.Unix(946684800, 0)
	if err := os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.ModTime()
}

func assertCorrectLinkUnchanged(t *testing.T, path, original string, stamp time.Time) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != original {
		t.Fatalf("content=%q, want unchanged %q", data, original)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(stamp) {
		t.Fatalf("mtime=%v, want unchanged %v", info.ModTime(), stamp)
	}
}

func assertCorrectLinkRepairSkipped(t *testing.T, result *RepairResult) {
	t.Helper()
	if !result.Complete || len(result.Changed) != 0 || len(result.changedPaths) != 0 ||
		len(result.Skipped) != 1 || len(result.Rejected) != 0 ||
		len(result.Errors) != 0 || len(result.RolledBack) != 0 {
		t.Fatalf("result=%+v, want one skip and no write ledger", result)
	}
	if !strings.Contains(result.Skipped[0], "a.md") {
		t.Fatalf("skip does not identify target: %v", result.Skipped)
	}
}

func TestCorrectLinkNoChangePreservesBytesMtimeAndLedger(t *testing.T) {
	for _, handler := range []string{"repair", "legacy"} {
		for _, dryRun := range []bool{false, true} {
			if handler == "legacy" && dryRun {
				continue // ApplyProposals has no dry-run mode.
			}
			for _, source := range []string{"[[absent]]", "[[new]]", ""} {
				t.Run(fmt.Sprintf("%s/dry-run=%t/from=%q", handler, dryRun, source), func(t *testing.T) {
					original := "---\nestado: Pending\n---\n\n# A\n\nSee [[new]].\n"
					root := newRollbackFixture(t, map[string]string{"a.md": original})
					path := filepath.Join(root, "a.md")
					stamp := stampCorrectLinkTarget(t, path)
					p := correctLinkProposal("a.md")
					p.From = source
					if handler == "repair" {
						result, err := ApplyRepair([]proposal.Proposal{p}, dryRun, root, false)
						if err != nil {
							t.Fatal(err)
						}
						assertCorrectLinkRepairSkipped(t, result)
					} else {
						report := &proposal.Report{Proposals: []proposal.Proposal{p}}
						skipped, err := ApplyProposals(context.Background(), report, root, nil, false)
						if err != nil {
							t.Fatal(err)
						}
						if len(report.Proposals) != 0 || !reflect.DeepEqual(skipped, []proposal.Proposal{p}) {
							t.Fatalf("applied=%v skipped=%v, want only original skipped proposal", report.Proposals, skipped)
						}
					}
					assertCorrectLinkUnchanged(t, path, original, stamp)
				})
			}
		}
	}
}

func TestApplyRepairStaleCorrectLinkDoesNotValidateUntouchedTarget(t *testing.T) {
	original := "---\nestado: INVALID\n---\n\n# A\n\nSee [[new]].\n"
	root := newRollbackFixture(t, map[string]string{"a.md": original})
	path := filepath.Join(root, "a.md")
	stamp := stampCorrectLinkTarget(t, path)
	result, err := ApplyRepair([]proposal.Proposal{correctLinkProposal("a.md")}, false, root, false)
	if err != nil {
		t.Fatal(err)
	}
	assertCorrectLinkRepairSkipped(t, result)
	assertCorrectLinkUnchanged(t, path, original, stamp)
}

func TestCorrectLinkReapplyPreservesBytesMtimeAndLedger(t *testing.T) {
	for _, handler := range []string{"repair", "legacy"} {
		t.Run(handler, func(t *testing.T) {
			original := "---\nestado: Pending\n---\n\n# A\n\nSee [[old]].\n"
			root := newRollbackFixture(t, map[string]string{"a.md": original})
			path := filepath.Join(root, "a.md")
			p := correctLinkProposal("a.md")
			if handler == "repair" {
				result, err := ApplyRepair([]proposal.Proposal{p}, false, root, false)
				if err != nil || !result.Complete || len(result.Changed) != 1 {
					t.Fatalf("initial repair: result=%+v err=%v", result, err)
				}
			} else {
				report := &proposal.Report{Proposals: []proposal.Proposal{p}}
				skipped, err := ApplyProposals(context.Background(), report, root, nil, false)
				if err != nil || len(report.Proposals) != 1 || len(skipped) != 0 {
					t.Fatalf("initial legacy repair: applied=%v skipped=%v err=%v", report.Proposals, skipped, err)
				}
			}
			want := strings.Replace(original, p.From, p.To, 1)
			stamp := stampCorrectLinkTarget(t, path)
			for _, dryRun := range []bool{true, false} {
				if handler == "repair" {
					result, err := ApplyRepair([]proposal.Proposal{p}, dryRun, root, false)
					if err != nil {
						t.Fatal(err)
					}
					assertCorrectLinkRepairSkipped(t, result)
				} else if !dryRun {
					report := &proposal.Report{Proposals: []proposal.Proposal{p}}
					skipped, err := ApplyProposals(context.Background(), report, root, nil, false)
					if err != nil || len(report.Proposals) != 0 || len(skipped) != 1 {
						t.Fatalf("reapply: applied=%v skipped=%v err=%v", report.Proposals, skipped, err)
					}
				}
				assertCorrectLinkUnchanged(t, path, want, stamp)
			}
		})
	}
}

func TestApplyRepairCorrectLinkChangeStillRollsBack(t *testing.T) {
	original := "---\nestado: Pending\n---\n\n# A\n"
	root := newRollbackFixture(t, map[string]string{"a.md": original})
	p := correctLinkProposal("a.md")
	p.From, p.To = "Pending", "INVALID"
	result, err := ApplyRepair([]proposal.Proposal{p}, false, root, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Complete || len(result.RolledBack) != 1 || result.RolledBack[0].Path != "a.md" ||
		len(result.Changed) != 0 || len(result.changedPaths) != 0 || len(result.Skipped) != 0 || len(result.Errors) != 0 {
		t.Fatalf("result=%+v, want rollback without successful changes", result)
	}
	data, err := os.ReadFile(filepath.Join(root, "a.md"))
	if err != nil || string(data) != original {
		t.Fatalf("content=%q err=%v, want restored %q", data, err, original)
	}
}

func TestCorrectLinkReplacesFirstOccurrence(t *testing.T) {
	for _, handler := range []string{"repair", "legacy", "legacy-absolute"} {
		for _, dryRun := range []bool{false, true} {
			if handler != "repair" && dryRun {
				continue
			}
			t.Run(fmt.Sprintf("%s/dry-run=%t", handler, dryRun), func(t *testing.T) {
				original := "---\nestado: Pending\n---\n\n# A\n\n[[old]] and [[old]]\n"
				root := newRollbackFixture(t, map[string]string{"a.md": original})
				path := filepath.Join(root, "a.md")
				stamp := stampCorrectLinkTarget(t, path)
				p := correctLinkProposal("a.md")
				if handler == "repair" {
					result, err := ApplyRepair([]proposal.Proposal{p}, dryRun, root, false)
					if err != nil || !result.Complete || len(result.Changed) != 1 || len(result.Skipped) != 0 || len(result.Rejected) != 0 {
						t.Fatalf("result=%+v err=%v", result, err)
					}
				} else {
					if handler == "legacy-absolute" {
						p.Paths = []string{path}
					}
					report := &proposal.Report{Proposals: []proposal.Proposal{p}}
					skipped, err := ApplyProposals(context.Background(), report, root, nil, false)
					if err != nil || len(skipped) != 0 || !reflect.DeepEqual(report.Proposals, []proposal.Proposal{p}) {
						t.Fatalf("applied=%v skipped=%v err=%v", report.Proposals, skipped, err)
					}
				}
				if dryRun {
					assertCorrectLinkUnchanged(t, path, original, stamp)
				} else {
					got, err := os.ReadFile(path)
					want := strings.Replace(original, p.From, p.To, 1)
					if err != nil || string(got) != want {
						t.Fatalf("content=%q err=%v, want %q", got, err, want)
					}
				}
			})
		}
	}
}

func TestApplyRepairCorrectLinkSequentialDryRunParity(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		t.Run(fmt.Sprintf("dry-run=%t", dryRun), func(t *testing.T) {
			original := "---\nestado: Pending\n---\n\n# A\n\n[[old]] and [[old]]\n"
			root := newRollbackFixture(t, map[string]string{"a.md": original})
			path := filepath.Join(root, "a.md")
			stamp := stampCorrectLinkTarget(t, path)
			p := correctLinkProposal("a.md")
			result, err := ApplyRepair([]proposal.Proposal{p, p, p}, dryRun, root, false)
			if err != nil || !result.Complete || len(result.Changed) != 2 || len(result.Skipped) != 1 || len(result.Rejected) != 0 {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if dryRun {
				assertCorrectLinkUnchanged(t, path, original, stamp)
			} else {
				got, err := os.ReadFile(path)
				want := strings.ReplaceAll(original, p.From, p.To)
				if err != nil || string(got) != want {
					t.Fatalf("content=%q err=%v, want %q", got, err, want)
				}
			}
		})
	}
}

func TestApplyProposalsCorrectLinkSplitsChangedAndSkippedPaths(t *testing.T) {
	changed := "---\nestado: Pending\n---\n\n[[old]]\n"
	stale := "---\nestado: Pending\n---\n\n[[new]]\n"
	root := newRollbackFixture(t, map[string]string{"a.md": changed, "b.md": stale})
	stamp := stampCorrectLinkTarget(t, filepath.Join(root, "b.md"))
	p := correctLinkProposal("a.md", "b.md")
	report := &proposal.Report{Proposals: []proposal.Proposal{p}}
	skipped, err := ApplyProposals(context.Background(), report, root, nil, false)
	if err != nil || len(report.Proposals) != 1 || len(skipped) != 1 ||
		!reflect.DeepEqual(report.Proposals[0].Paths, []string{"a.md"}) ||
		!reflect.DeepEqual(skipped[0].Paths, []string{"b.md"}) {
		t.Fatalf("applied=%v skipped=%v err=%v", report.Proposals, skipped, err)
	}
	assertCorrectLinkUnchanged(t, filepath.Join(root, "b.md"), stale, stamp)
}

func TestCorrectLinkContainmentParity(t *testing.T) {
	for _, escape := range []string{"relative", "absolute", "file-symlink", "directory-symlink"} {
		for _, handler := range []string{"repair", "legacy"} {
			for _, dryRun := range []bool{false, true} {
				if handler == "legacy" && dryRun {
					continue
				}
				t.Run(fmt.Sprintf("%s/%s/dry-run=%t", escape, handler, dryRun), func(t *testing.T) {
					base := t.TempDir()
					root := filepath.Join(base, "root")
					if err := os.Mkdir(root, 0o755); err != nil {
						t.Fatal(err)
					}
					original := "See [[old]].\n"
					inside := filepath.Join(root, "inside.md")
					outside := filepath.Join(base, "outside.md")
					mustWriteFile(t, inside, []byte(original))
					mustWriteFile(t, outside, []byte(original))
					insideStamp := stampCorrectLinkTarget(t, inside)
					outsideStamp := stampCorrectLinkTarget(t, outside)
					path := "../outside.md"
					switch escape {
					case "absolute":
						path = outside
					case "file-symlink", "directory-symlink":
						target := outside
						path = "alias.md"
						if escape == "directory-symlink" {
							target = base
							path = filepath.Join("alias.md", "outside.md")
						}
						if err := os.Symlink(target, filepath.Join(root, "alias.md")); err != nil {
							t.Skipf("symlink unavailable: %v", err)
						}
					}
					// An earlier contained target must not be accessed or written by
					// a proposal whose later target fails containment.
					p := correctLinkProposal("inside.md", path)
					if handler == "repair" {
						result, err := ApplyRepair([]proposal.Proposal{p}, dryRun, root, false)
						if err != nil || !result.Complete || len(result.Rejected) != 1 || len(result.Changed) != 0 || len(result.Skipped) != 0 {
							t.Fatalf("result=%+v err=%v", result, err)
						}
					} else {
						report := &proposal.Report{Proposals: []proposal.Proposal{p}}
						_, err := ApplyProposals(context.Background(), report, root, nil, false)
						if err == nil || !strings.Contains(err.Error(), "containment violation") {
							t.Fatalf("error=%v, want legacy containment failure", err)
						}
					}
					assertCorrectLinkUnchanged(t, inside, original, insideStamp)
					assertCorrectLinkUnchanged(t, outside, original, outsideStamp)
				})
			}
		}
	}
}
