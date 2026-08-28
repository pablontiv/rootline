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
		{ID: DestinationID("legacy"), Path: "/home/.config/opencode/skills/rootline", Kind: KindAbsent},
	}
	_, err := BuildInstallPlan(source, states)
	assertOperationErrorCode(t, err, ErrUnsupportedFileType)
}
