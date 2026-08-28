package skilldist

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
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

func BuildUninstallPlan(receipt Receipt, states []DestinationState) (Plan, error) {
	source, err := receiptPlanSource(receipt)
	if err != nil {
		return Plan{}, err
	}
	actionsByDestination := receiptActionsByDestination(receipt)
	actions := make([]Action, 0, len(states))
	for _, state := range states {
		if !isSupportedDestinationID(state.ID) {
			return Plan{}, operationError(ErrUnsupportedFileType, state.Path, string(state.ID), fmt.Errorf("unsupported destination %q", state.ID))
		}
		if state.Kind != KindCorrectSymlink {
			return Plan{}, operationError(ErrRestoreConflict, state.Path, string(state.ID), fmt.Errorf("destination is not an intact managed symlink"))
		}
		recorded, ok := actionsByDestination[state.ID]
		if !ok || !recorded.Complete || recorded.After.Kind != KindCorrectSymlink || filepathClean(recorded.After.Path) != filepathClean(state.Path) || !sameDestinationEvidence(recorded.After, state) {
			return Plan{}, operationError(ErrRestoreConflict, state.Path, string(state.ID), fmt.Errorf("destination does not match receipt evidence"))
		}
		if filepathClean(state.LexicalTarget) != filepathClean(source.SkillPath) || state.Digest != source.Digest {
			return Plan{}, operationError(ErrRestoreConflict, state.Path, string(state.ID), fmt.Errorf("destination does not target the receipted source"))
		}
		actions = append(actions, Action{Kind: ActionRemoveManagedSymlink, Destination: state})
	}

	plan := Plan{Operation: OperationUninstall, Source: copySourcePtr(source), Actions: actions}
	digest, err := digestReceiptBoundPlan(plan, receipt.ID, nil)
	if err != nil {
		return Plan{}, err
	}
	plan.Digest = digest
	return plan, nil
}

func BuildRestorePlan(receipt Receipt, states []DestinationState) (Plan, error) {
	source, err := receiptPlanSource(receipt)
	if err != nil {
		return Plan{}, err
	}
	actionsByDestination := receiptActionsByDestination(receipt)
	actions := make([]Action, 0, len(states))
	preimages := make([]DestinationState, 0, len(states))
	for _, state := range states {
		if !isSupportedDestinationID(state.ID) {
			return Plan{}, operationError(ErrUnsupportedFileType, state.Path, string(state.ID), fmt.Errorf("unsupported destination %q", state.ID))
		}
		recorded, ok := actionsByDestination[state.ID]
		if !ok || !recorded.Complete || filepathClean(recorded.After.Path) != filepathClean(state.Path) || !currentRestorableStateMatches(recorded.After, state) {
			return Plan{}, operationError(ErrRestoreConflict, state.Path, string(state.ID), fmt.Errorf("destination does not match receipt evidence"))
		}
		kind, err := restoreActionKind(recorded.Before)
		if err != nil {
			return Plan{}, err
		}
		if recorded.Action == ActionNoOp && sameDestinationEvidence(recorded.Before, state) && filepathClean(recorded.Before.Path) == filepathClean(state.Path) {
			kind = ActionNoOp
		}
		actions = append(actions, Action{Kind: kind, Destination: state})
		preimages = append(preimages, recorded.Before)
	}

	plan := Plan{Operation: OperationRestore, Source: copySourcePtr(source), Actions: actions}
	digest, err := digestReceiptBoundPlan(plan, receipt.ID, preimages)
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

func restoreActionKind(preimage DestinationState) (ActionKind, error) {
	switch preimage.Kind {
	case KindAbsent:
		return ActionRemoveManagedSymlink, nil
	case KindDirectory, KindCorrectSymlink, KindDivergentSymlink:
		return ActionRestorePreimage, nil
	case KindUnsupported:
		return "", operationError(ErrUnsupportedFileType, preimage.Path, string(preimage.ID), fmt.Errorf("unsupported preimage file type at %s", preimage.Path))
	default:
		return "", operationError(ErrUnsupportedFileType, preimage.Path, string(preimage.ID), fmt.Errorf("unsupported preimage state %q", preimage.Kind))
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
	return digestCanonicalPlan(canonical)
}

func digestReceiptBoundPlan(plan Plan, receiptID string, preimages []DestinationState) (Digest, error) {
	canonical := struct {
		Operation Operation          `json:"operation"`
		ReceiptID string             `json:"receipt_id"`
		Source    *Source            `json:"source,omitempty"`
		Actions   []Action           `json:"actions"`
		Preimages []DestinationState `json:"preimages,omitempty"`
	}{plan.Operation, receiptID, plan.Source, plan.Actions, preimages}
	return digestCanonicalPlan(canonical)
}

func digestCanonicalPlan(canonical any) (Digest, error) {
	data, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return Digest("sha256:" + hex.EncodeToString(sum[:])), nil
}

func receiptPlanSource(receipt Receipt) (Source, error) {
	if receipt.ID == "" {
		return Source{}, operationError(ErrRestoreConflict, "", "", fmt.Errorf("receipt ID is required"))
	}
	if !receipt.Complete {
		return Source{}, operationError(ErrRestoreConflict, "", "", fmt.Errorf("receipt %q is incomplete", receipt.ID))
	}
	if receipt.Source == nil {
		return Source{}, operationError(ErrRestoreConflict, "", "", fmt.Errorf("receipt %q has no source evidence", receipt.ID))
	}
	return *receipt.Source, nil
}

func receiptActionsByDestination(receipt Receipt) map[DestinationID]ActionResult {
	actions := make(map[DestinationID]ActionResult, len(receipt.Actions))
	for _, action := range receipt.Actions {
		actions[action.Destination] = action
	}
	return actions
}

func currentRestorableStateMatches(recorded, current DestinationState) bool {
	if currentReceiptedLinkMatches(recorded, current) {
		return true
	}
	return current.Kind == KindAbsent &&
		receiptRecordedManagedLink(recorded) &&
		recorded.ID == current.ID &&
		filepathClean(recorded.Path) == filepathClean(current.Path)
}

func currentReceiptedLinkMatches(recorded, current DestinationState) bool {
	if current.Kind != KindCorrectSymlink && current.Kind != KindDivergentSymlink {
		return false
	}
	if !receiptRecordedManagedLink(recorded) {
		return false
	}
	return recorded.ID == current.ID &&
		filepathClean(recorded.Path) == filepathClean(current.Path) &&
		filepathClean(recorded.LexicalTarget) == filepathClean(current.LexicalTarget) &&
		filepathClean(recorded.CanonicalTarget) == filepathClean(current.CanonicalTarget)
}

func receiptRecordedManagedLink(recorded DestinationState) bool {
	return recorded.Kind == KindCorrectSymlink || recorded.Kind == KindDivergentSymlink
}

func filepathClean(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}
