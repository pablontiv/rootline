package skilldist

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

func BuildInstallPlan(source Source, states []DestinationState) (Plan, error) {
	actions := make([]Action, 0, len(states))
	for _, state := range states {
		if !isSupportedDestinationID(state.ID) {
			return Plan{}, operationError(ErrUnsupportedFileType, state.Path, string(state.ID), fmt.Errorf("unsupported destination %q", state.ID))
		}
		actionKind, err := installActionKind(state)
		if err != nil {
			return Plan{}, err
		}
		actions = append(actions, Action{Kind: actionKind, Destination: state})
	}

	sourceCopy := source
	plan := Plan{
		Operation: OperationInstall,
		Source:    &sourceCopy,
		Actions:   actions,
	}
	digest, err := digestPlan(plan)
	if err != nil {
		return Plan{}, err
	}
	plan.Digest = digest
	return plan, nil
}

func installActionKind(state DestinationState) (ActionKind, error) {
	switch state.Kind {
	case KindAbsent:
		return ActionCreateSymlink, nil
	case KindCorrectSymlink:
		return ActionNoOp, nil
	case KindDirectory, KindDivergentSymlink:
		return ActionReplaceWithSymlink, nil
	case KindUnsupported:
		return "", operationError(ErrUnsupportedFileType, state.Path, string(state.ID), fmt.Errorf("unsupported file type at %s", state.Path))
	default:
		return "", operationError(ErrUnsupportedFileType, state.Path, string(state.ID), fmt.Errorf("unsupported destination state %q", state.Kind))
	}
}

func isSupportedDestinationID(id DestinationID) bool {
	return id == DestinationClaude || id == DestinationAgents
}

func digestPlan(plan Plan) (Digest, error) {
	canonical := struct {
		Operation Operation `json:"operation"`
		Source    *Source   `json:"source,omitempty"`
		Actions   []Action  `json:"actions"`
	}{plan.Operation, plan.Source, plan.Actions}
	data, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return Digest("sha256:" + hex.EncodeToString(sum[:])), nil
}
