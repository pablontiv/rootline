package skilldist

import "testing"

func TestInstallPlanIsStableIdempotentAndPreimageBound(t *testing.T) {
	source := Source{RepoRoot: "/repo", SkillPath: "/repo/.claude/skills/rootline", Commit: "abc", Digest: "sha256:source"}
	states := []DestinationState{
		{ID: DestinationClaude, Path: "/home/.claude/skills/rootline", Kind: KindDirectory, Digest: "sha256:old"},
		{ID: DestinationAgents, Path: "/home/.agents/skills/rootline", Kind: KindAbsent},
	}
	plan, err := BuildInstallPlan(source, states)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Operation != OperationInstall {
		t.Fatalf("Operation = %q, want %q", plan.Operation, OperationInstall)
	}
	if len(plan.Actions) != 2 || plan.Actions[0].Kind != ActionReplaceWithSymlink || plan.Actions[1].Kind != ActionCreateSymlink {
		t.Fatalf("actions = %#v", plan.Actions)
	}
	repeated, err := BuildInstallPlan(source, states)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Digest != repeated.Digest {
		t.Fatalf("plan digest is unstable: %q != %q", plan.Digest, repeated.Digest)
	}
	states[0].Digest = "sha256:changed"
	changed, err := BuildInstallPlan(source, states)
	if err != nil {
		t.Fatal(err)
	}
	if changed.Digest == plan.Digest {
		t.Fatal("preimage change did not invalidate approval")
	}
}

func TestInstallPlanNoOpsExactSymlinks(t *testing.T) {
	source := Source{RepoRoot: "/repo", SkillPath: "/repo/.claude/skills/rootline", Commit: "abc", Digest: "sha256:source"}
	states := []DestinationState{
		{ID: DestinationClaude, Path: "/home/.claude/skills/rootline", Kind: KindCorrectSymlink, Digest: "sha256:source", LexicalTarget: source.SkillPath, CanonicalTarget: source.SkillPath},
		{ID: DestinationAgents, Path: "/home/.agents/skills/rootline", Kind: KindCorrectSymlink, Digest: "sha256:source", LexicalTarget: source.SkillPath, CanonicalTarget: source.SkillPath},
	}
	plan, err := BuildInstallPlan(source, states)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) != len(states) {
		t.Fatalf("len(actions) = %d, want %d", len(plan.Actions), len(states))
	}
	for i, action := range plan.Actions {
		if action.Kind != ActionNoOp {
			t.Fatalf("action[%d] = %#v, want no-op", i, action)
		}
	}
	if plan.Digest == "" {
		t.Fatal("plan digest is empty")
	}
}

func TestInstallPlanRejectsUnsupportedDestinationBeforeApproval(t *testing.T) {
	source := Source{RepoRoot: "/repo", SkillPath: "/repo/.claude/skills/rootline", Commit: "abc", Digest: "sha256:source"}
	states := []DestinationState{
		{ID: DestinationClaude, Path: "/home/.claude/skills/rootline", Kind: KindUnsupported},
	}
	_, err := BuildInstallPlan(source, states)
	assertOperationErrorCode(t, err, ErrUnsupportedFileType)
}

func TestInstallPlanRejectsUnsupportedDestinationID(t *testing.T) {
	source := Source{RepoRoot: "/repo", SkillPath: "/repo/.claude/skills/rootline", Commit: "abc", Digest: "sha256:source"}
	states := []DestinationState{
		{ID: DestinationID("legacy"), Path: "/home/unsupported/rootline", Kind: KindAbsent},
	}
	_, err := BuildInstallPlan(source, states)
	assertOperationErrorCode(t, err, ErrUnsupportedFileType)
}

func TestUninstallPlanIsReceiptAndCurrentStateBound(t *testing.T) {
	source := Source{RepoRoot: "/repo", SkillPath: "/repo/.claude/skills/rootline", Commit: "abc", Digest: "sha256:source"}
	receipt := completeInstallReceiptForPlan("receipt-1", source)
	states := []DestinationState{
		{ID: DestinationClaude, Path: "/home/.claude/skills/rootline", Kind: KindCorrectSymlink, Digest: source.Digest, LexicalTarget: source.SkillPath, CanonicalTarget: source.SkillPath},
		{ID: DestinationAgents, Path: "/home/.agents/skills/rootline", Kind: KindCorrectSymlink, Digest: source.Digest, LexicalTarget: source.SkillPath, CanonicalTarget: source.SkillPath},
	}

	plan, err := BuildUninstallPlan(receipt, states)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Operation != OperationUninstall || len(plan.Actions) != 2 {
		t.Fatalf("plan = %#v", plan)
	}
	for _, action := range plan.Actions {
		if action.Kind != ActionRemoveManagedSymlink {
			t.Fatalf("action = %#v, want remove managed symlink", action)
		}
	}
	repeated, err := BuildUninstallPlan(receipt, states)
	if err != nil {
		t.Fatal(err)
	}
	if repeated.Digest != plan.Digest {
		t.Fatalf("digest unstable: %q != %q", repeated.Digest, plan.Digest)
	}
	receipt.ID = "receipt-2"
	changedReceipt, err := BuildUninstallPlan(receipt, states)
	if err != nil {
		t.Fatal(err)
	}
	if changedReceipt.Digest == plan.Digest {
		t.Fatal("receipt change did not invalidate uninstall approval")
	}
	states[0].Digest = "sha256:changed"
	_, err = BuildUninstallPlan(completeInstallReceiptForPlan("receipt-1", source), states)
	assertOperationErrorCode(t, err, ErrRestoreConflict)
}

func TestRestorePlanUsesRecordedPreimagesAndBindsReceiptAndCurrentState(t *testing.T) {
	source := Source{RepoRoot: "/repo", SkillPath: "/repo/.claude/skills/rootline", Commit: "abc", Digest: "sha256:source"}
	receipt := completeInstallReceiptForPlan("receipt-1", source)
	receipt.Actions[0].Before = DestinationState{ID: DestinationClaude, Path: "/home/.claude/skills/rootline", Kind: KindDirectory, Digest: "sha256:old"}
	receipt.Backups = []Backup{{Destination: DestinationClaude, OriginalPath: "/home/.claude/skills/rootline", StoredPath: "/state/backups/receipt-1/claude", Kind: KindDirectory, Digest: "sha256:old"}}
	states := []DestinationState{
		{ID: DestinationClaude, Path: "/home/.claude/skills/rootline", Kind: KindCorrectSymlink, Digest: source.Digest, LexicalTarget: source.SkillPath, CanonicalTarget: source.SkillPath},
		{ID: DestinationAgents, Path: "/home/.agents/skills/rootline", Kind: KindCorrectSymlink, Digest: source.Digest, LexicalTarget: source.SkillPath, CanonicalTarget: source.SkillPath},
	}

	plan, err := BuildRestorePlan(receipt, states)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Operation != OperationRestore || len(plan.Actions) != 2 {
		t.Fatalf("plan = %#v", plan)
	}
	if plan.Actions[0].Kind != ActionRestorePreimage || plan.Actions[1].Kind != ActionRemoveManagedSymlink {
		t.Fatalf("actions = %#v", plan.Actions)
	}
	repeated, err := BuildRestorePlan(receipt, states)
	if err != nil {
		t.Fatal(err)
	}
	if repeated.Digest != plan.Digest {
		t.Fatalf("digest unstable: %q != %q", repeated.Digest, plan.Digest)
	}
	receipt.ID = "receipt-2"
	changedReceipt, err := BuildRestorePlan(receipt, states)
	if err != nil {
		t.Fatal(err)
	}
	if changedReceipt.Digest == plan.Digest {
		t.Fatal("receipt change did not invalidate restore approval")
	}
	states[0].Digest = "sha256:changed"
	changedState, err := BuildRestorePlan(completeInstallReceiptForPlan("receipt-1", source), states)
	if err != nil {
		t.Fatal(err)
	}
	if changedState.Digest == plan.Digest {
		t.Fatal("current state change did not invalidate restore approval")
	}
}

func completeInstallReceiptForPlan(id string, source Source) Receipt {
	return Receipt{
		ID:        id,
		Operation: OperationInstall,
		Complete:  true,
		Source:    &source,
		Actions: []ActionResult{
			{
				Destination: DestinationClaude,
				Action:      ActionCreateSymlink,
				Before:      DestinationState{ID: DestinationClaude, Path: "/home/.claude/skills/rootline", Kind: KindAbsent},
				After:       DestinationState{ID: DestinationClaude, Path: "/home/.claude/skills/rootline", Kind: KindCorrectSymlink, Digest: source.Digest, LexicalTarget: source.SkillPath, CanonicalTarget: source.SkillPath},
				Complete:    true,
			},
			{
				Destination: DestinationAgents,
				Action:      ActionCreateSymlink,
				Before:      DestinationState{ID: DestinationAgents, Path: "/home/.agents/skills/rootline", Kind: KindAbsent},
				After:       DestinationState{ID: DestinationAgents, Path: "/home/.agents/skills/rootline", Kind: KindCorrectSymlink, Digest: source.Digest, LexicalTarget: source.SkillPath, CanonicalTarget: source.SkillPath},
				Complete:    true,
			},
		},
	}
}
