package skilldist

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

const receiptKind = "rootline/skill-receipt"

type Service struct {
	homeDir      string
	stateDir     string
	now          func() time.Time
	newReceiptID func() string
	store        *Store
	destinations []Destination
	executor     *installExecutor
}

type installExecutor struct {
	symlink       func(oldname, newname string) error
	beforeSymlink func(DestinationState) error
}

type publishOutcome struct {
	after      DestinationState
	rolledBack bool
}

func New(options Options) (*Service, error) {
	homeDir := filepath.Clean(options.HomeDir)
	if options.HomeDir == "" {
		defaultHome, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve home directory: %w", err)
		}
		homeDir = filepath.Clean(defaultHome)
	}
	if homeDir == "." || homeDir == string(filepath.Separator) || homeDir == "" {
		return nil, fmt.Errorf("home directory is required")
	}

	stateDir := filepath.Clean(options.StateDir)
	if options.StateDir == "" {
		stateDir = defaultStateDir(homeDir)
	}
	if stateDir == "." || stateDir == string(filepath.Separator) || stateDir == "" {
		return nil, fmt.Errorf("state directory is required")
	}

	now := options.Now
	if now == nil {
		now = time.Now
	}
	newReceiptID := options.NewReceiptID
	if newReceiptID == nil {
		newReceiptID = randomReceiptID
	}

	executor := &installExecutor{symlink: os.Symlink}
	return &Service{
		homeDir:      homeDir,
		stateDir:     stateDir,
		now:          now,
		newReceiptID: newReceiptID,
		store:        NewStore(stateDir),
		destinations: SupportedDestinations(homeDir),
		executor:     executor,
	}, nil
}

func defaultStateDir(homeDir string) string {
	if xdgStateHome := os.Getenv("XDG_STATE_HOME"); xdgStateHome != "" {
		return filepath.Clean(xdgStateHome)
	}
	return filepath.Join(homeDir, ".local", "state")
}

func randomReceiptID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("generate receipt ID: %v", err))
	}
	return hex.EncodeToString(b[:])
}

func (s *Service) Install(ctx context.Context, sourcePath string, approval Digest) Result {
	result := Result{Operation: OperationInstall, Errors: []OperationError{}}
	source, _, plan, ok := s.installPlan(ctx, sourcePath, &result)
	if !ok {
		return result
	}
	if approval == "" {
		return result
	}
	if approval != plan.Digest {
		addResultError(&result, operationError(ErrPreimageDigestChanged, source.SkillPath, "", fmt.Errorf("approval digest %q does not match current plan digest %q", approval, plan.Digest)))
		return result
	}

	result.Attempted = true
	receipt := Receipt{
		Version:    1,
		Kind:       receiptKind,
		ID:         s.newReceiptID(),
		Timestamp:  s.now(),
		Operation:  OperationInstall,
		Source:     copySourcePtr(source),
		PlanDigest: plan.Digest,
		Actions:    []ActionResult{},
		Backups:    []Backup{},
		Errors:     []OperationError{},
	}
	if err := s.store.Reserve(receipt.ID); err != nil {
		opErr := operationError(ErrBackupFailed, "", "", err)
		addResultError(&result, opErr)
		receipt.Errors = append(receipt.Errors, *opErr)
		result.Receipt = &receipt
		return result
	}

	complete := true
	for _, action := range plan.Actions {
		actionResult := ActionResult{Destination: action.Destination.ID, Action: action.Kind, Before: action.Destination}
		if action.Kind == ActionNoOp {
			actionResult.After = action.Destination
			actionResult.Complete = true
			receipt.Actions = append(receipt.Actions, actionResult)
			continue
		}

		backup, err := s.store.Backup(receipt.ID, action.Destination)
		if err != nil {
			opErr := coerceOperationError(ErrBackupFailed, action.Destination.Path, string(action.Destination.ID), err)
			actionResult.Error = opErr
			receipt.Actions = append(receipt.Actions, actionResult)
			addResultError(&result, opErr)
			receipt.Errors = append(receipt.Errors, *opErr)
			complete = false
			break
		}
		if backup.Kind != KindAbsent {
			receipt.Backups = append(receipt.Backups, backup)
		}

		outcome, err := s.executor.publishSymlink(action.Destination, source, backup)
		actionResult.After = outcome.after
		actionResult.RolledBack = outcome.rolledBack
		if err != nil {
			opErr := coerceOperationError(ErrVerificationFailed, action.Destination.Path, string(action.Destination.ID), err)
			actionResult.Error = opErr
			receipt.Actions = append(receipt.Actions, actionResult)
			addResultError(&result, opErr)
			receipt.Errors = append(receipt.Errors, *opErr)
			complete = false
			break
		}
		actionResult.Complete = true
		receipt.Actions = append(receipt.Actions, actionResult)
	}

	receipt.Complete = complete
	if err := s.store.Append(receipt); err != nil {
		opErr := operationError(ErrBackupFailed, "", "", fmt.Errorf("append receipt: %w", err))
		addResultError(&result, opErr)
		receipt.Errors = append(receipt.Errors, *opErr)
		receipt.Complete = false
	}
	result.Receipt = &receipt
	result.Backups = append([]Backup(nil), receipt.Backups...)
	result.Complete = receipt.Complete
	return result
}

func (s *Service) Status(ctx context.Context, sourcePath string) Result {
	result := Result{Operation: OperationStatus, Errors: []OperationError{}}
	source, err := ResolveSource(ctx, sourcePath)
	if err != nil {
		addResultError(&result, coerceOperationError(ErrSourceNotCanonical, sourcePath, "", err))
		return result
	}
	result.Source = copySourcePtr(source)

	states, err := InventoryDestinations(s.homeDir, source)
	if err != nil {
		addResultError(&result, coerceOperationError(ErrVerificationFailed, "", "", err))
		return result
	}
	result.Destinations = append([]DestinationState(nil), states...)
	result.Complete = allDestinationsExact(states)

	latest, ok, err := s.store.Latest()
	if err != nil {
		addResultError(&result, coerceOperationError(ErrBackupFailed, "", "", err))
		return result
	}
	if ok {
		result.Receipt = &latest
		result.Backups = append([]Backup(nil), latest.Backups...)
		result.ReceiptDrift = receiptDrifted(latest, source, states)
	}
	return result
}

func (s *Service) installPlan(ctx context.Context, sourcePath string, result *Result) (Source, []DestinationState, Plan, bool) {
	source, err := ResolveSource(ctx, sourcePath)
	if err != nil {
		addResultError(result, coerceOperationError(ErrSourceNotCanonical, sourcePath, "", err))
		return Source{}, nil, Plan{}, false
	}
	result.Source = copySourcePtr(source)

	states, err := InventoryDestinations(s.homeDir, source)
	if err != nil {
		addResultError(result, coerceOperationError(ErrVerificationFailed, "", "", err))
		return source, nil, Plan{}, false
	}
	result.Destinations = append([]DestinationState(nil), states...)

	plan, err := BuildInstallPlan(source, states)
	if err != nil {
		addResultError(result, coerceOperationError(ErrUnsupportedFileType, "", "", err))
		return source, states, Plan{}, false
	}
	result.Plan = &plan
	return source, states, plan, true
}

func (e *installExecutor) publishSymlink(destination DestinationState, source Source, backup Backup) (publishOutcome, error) {
	outcome := publishOutcome{after: destination}
	sourceCanonical, err := filepath.EvalSymlinks(source.SkillPath)
	if err != nil {
		return outcome, err
	}

	stagedPath := ""
	symlinkCreated := false
	if destination.Kind != KindAbsent {
		stagedPath, err = uniqueSiblingPath(destination.Path)
		if err != nil {
			return outcome, err
		}
		if err := os.Rename(destination.Path, stagedPath); err != nil {
			return outcome, coerceOperationError(ErrPreimageDigestChanged, destination.Path, string(destination.ID), err)
		}
		stagedState, err := inventoryDestination(Destination{ID: destination.ID, Path: stagedPath}, source, sourceCanonical)
		if err != nil {
			rolledBack := e.restoreStaged(destination, stagedPath, backup)
			outcome.rolledBack = rolledBack
			return outcome, coerceOperationError(ErrVerificationFailed, destination.Path, string(destination.ID), err)
		}
		if !sameDestinationEvidence(destination, stagedState) {
			rolledBack := e.restoreStaged(destination, stagedPath, backup)
			outcome.rolledBack = rolledBack
			return outcome, operationError(ErrVerificationFailed, destination.Path, string(destination.ID), fmt.Errorf("staged preimage evidence changed"))
		}
	}

	if e.beforeSymlink != nil {
		if err := e.beforeSymlink(destination); err != nil {
			rolledBack := e.rollbackAfterFailure(destination, stagedPath, backup, symlinkCreated)
			outcome.rolledBack = rolledBack
			outcome.after = inventoryAfterFailure(destination, source, sourceCanonical)
			return outcome, err
		}
	}
	if err := verifySourceDigest(source); err != nil {
		rolledBack := e.rollbackAfterFailure(destination, stagedPath, backup, symlinkCreated)
		outcome.rolledBack = rolledBack
		outcome.after = inventoryAfterFailure(destination, source, sourceCanonical)
		return outcome, err
	}

	if err := os.MkdirAll(filepath.Dir(destination.Path), 0o700); err != nil {
		rolledBack := e.rollbackAfterFailure(destination, stagedPath, backup, symlinkCreated)
		outcome.rolledBack = rolledBack
		return outcome, err
	}
	if err := e.symlink(source.SkillPath, destination.Path); err != nil {
		rolledBack := e.rollbackAfterFailure(destination, stagedPath, backup, symlinkCreated)
		outcome.rolledBack = rolledBack
		outcome.after = inventoryAfterFailure(destination, source, sourceCanonical)
		return outcome, mapSymlinkError(err, destination)
	}
	symlinkCreated = true

	if err := verifySourceDigest(source); err != nil {
		_ = os.Remove(destination.Path)
		rolledBack := e.rollbackAfterFailure(destination, stagedPath, backup, symlinkCreated)
		outcome.rolledBack = rolledBack
		outcome.after = inventoryAfterFailure(destination, source, sourceCanonical)
		return outcome, err
	}
	after, err := inventoryDestination(Destination{ID: destination.ID, Path: destination.Path}, source, sourceCanonical)
	if err != nil {
		_ = os.Remove(destination.Path)
		rolledBack := e.rollbackAfterFailure(destination, stagedPath, backup, symlinkCreated)
		outcome.rolledBack = rolledBack
		return outcome, coerceOperationError(ErrVerificationFailed, destination.Path, string(destination.ID), err)
	}
	outcome.after = after
	if after.Kind != KindCorrectSymlink {
		_ = os.Remove(destination.Path)
		rolledBack := e.rollbackAfterFailure(destination, stagedPath, backup, symlinkCreated)
		outcome.rolledBack = rolledBack
		outcome.after = inventoryAfterFailure(destination, source, sourceCanonical)
		return outcome, operationError(ErrVerificationFailed, destination.Path, string(destination.ID), fmt.Errorf("published symlink is not exact"))
	}

	if stagedPath != "" {
		if err := os.RemoveAll(stagedPath); err != nil {
			return outcome, coerceOperationError(ErrVerificationFailed, stagedPath, string(destination.ID), err)
		}
	}
	return outcome, nil
}

func (e *installExecutor) rollbackAfterFailure(destination DestinationState, stagedPath string, backup Backup, symlinkCreated bool) bool {
	if symlinkCreated {
		_ = os.Remove(destination.Path)
	}
	if stagedPath == "" {
		return true
	}
	return e.restoreStaged(destination, stagedPath, backup)
}

func (e *installExecutor) restoreStaged(destination DestinationState, stagedPath string, backup Backup) bool {
	if _, err := os.Lstat(destination.Path); err == nil {
		if backup.Kind != KindAbsent {
			if verifyErr := verifyStoredBackup(backup); verifyErr != nil {
				return false
			}
		}
		_ = os.RemoveAll(stagedPath)
		return false
	} else if !os.IsNotExist(err) {
		return false
	}
	return os.Rename(stagedPath, destination.Path) == nil
}

func verifyStoredBackup(backup Backup) error {
	store := Store{}
	return store.VerifyBackup(backup)
}

func verifySourceDigest(source Source) error {
	current, err := digestCanonicalTree(source.SkillPath)
	if err != nil {
		return coerceOperationError(ErrSourceDigestChanged, source.SkillPath, "", err)
	}
	if current != source.Digest {
		return operationError(ErrSourceDigestChanged, source.SkillPath, "", fmt.Errorf("source digest %q does not match approved digest %q", current, source.Digest))
	}
	return nil
}

func mapSymlinkError(err error, destination DestinationState) error {
	if errors.Is(err, os.ErrPermission) || errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
		return operationError(ErrSymlinkPermission, destination.Path, string(destination.ID), err)
	}
	if errors.Is(err, os.ErrExist) {
		return operationError(ErrRestoreConflict, destination.Path, string(destination.ID), err)
	}
	return operationError(ErrVerificationFailed, destination.Path, string(destination.ID), err)
}

func uniqueSiblingPath(path string) (string, error) {
	parent := filepath.Dir(path)
	base := filepath.Base(path)
	for i := 0; i < 100; i++ {
		candidate := filepath.Join(parent, fmt.Sprintf(".%s.rootline-stage.%d.%d", base, os.Getpid(), time.Now().UnixNano()+int64(i)))
		if _, err := os.Lstat(candidate); os.IsNotExist(err) {
			return candidate, nil
		} else if err != nil {
			return "", err
		}
	}
	return "", fmt.Errorf("could not allocate staging path for %s", path)
}

func inventoryAfterFailure(destination DestinationState, source Source, sourceCanonical string) DestinationState {
	state, err := inventoryDestination(Destination{ID: destination.ID, Path: destination.Path}, source, sourceCanonical)
	if err != nil {
		return DestinationState{ID: destination.ID, Path: destination.Path, Kind: KindUnsupported}
	}
	return state
}

func sameDestinationEvidence(expected, observed DestinationState) bool {
	return expected.ID == observed.ID &&
		expected.Kind == observed.Kind &&
		expected.Digest == observed.Digest &&
		filepath.Clean(expected.LexicalTarget) == filepath.Clean(observed.LexicalTarget) &&
		filepath.Clean(expected.CanonicalTarget) == filepath.Clean(observed.CanonicalTarget)
}

func allDestinationsExact(states []DestinationState) bool {
	if len(states) != 2 {
		return false
	}
	for _, state := range states {
		if state.Kind != KindCorrectSymlink {
			return false
		}
	}
	return true
}

func receiptDrifted(receipt Receipt, source Source, states []DestinationState) bool {
	if receipt.Source == nil || receipt.Source.Digest != source.Digest || receipt.Source.Commit != source.Commit || filepath.Clean(receipt.Source.SkillPath) != filepath.Clean(source.SkillPath) {
		return true
	}
	afterByDestination := make(map[DestinationID]DestinationState, len(receipt.Actions))
	for _, action := range receipt.Actions {
		afterByDestination[action.Destination] = action.After
	}
	for _, state := range states {
		recorded, ok := afterByDestination[state.ID]
		if !ok || !sameDestinationEvidence(recorded, state) || filepath.Clean(recorded.Path) != filepath.Clean(state.Path) {
			return true
		}
	}
	return false
}

func copySourcePtr(source Source) *Source {
	sourceCopy := source
	return &sourceCopy
}

func addResultError(result *Result, err *OperationError) {
	if err == nil {
		return
	}
	result.Errors = append(result.Errors, *err)
}

func coerceOperationError(defaultCode ErrorCode, path, destination string, err error) *OperationError {
	if err == nil {
		return nil
	}
	var opErr *OperationError
	if errors.As(err, &opErr) {
		copy := *opErr
		if copy.Path == "" {
			copy.Path = path
		}
		if copy.Destination == "" {
			copy.Destination = destination
		}
		return &copy
	}
	return operationError(defaultCode, path, destination, err)
}
