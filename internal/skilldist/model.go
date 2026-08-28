package skilldist

import "fmt"

type Digest string

type ErrorCode string

const (
	ErrSourceNotCanonical    ErrorCode = "source_not_canonical"
	ErrLinkedWorktreeRefused ErrorCode = "linked_worktree_refused"
	ErrSourceDigestChanged   ErrorCode = "source_digest_changed"
	ErrPreimageDigestChanged ErrorCode = "preimage_digest_changed"
	ErrUnsupportedFileType   ErrorCode = "unsupported_file_type"
	ErrSymlinkPermission     ErrorCode = "symlink_permission_denied"
	ErrBackupFailed          ErrorCode = "backup_failed"
	ErrVerificationFailed    ErrorCode = "verification_failed"
	ErrRestoreConflict       ErrorCode = "restore_conflict"
)

type OperationError struct {
	Code        ErrorCode `json:"code"`
	Path        string    `json:"path,omitempty"`
	Destination string    `json:"destination,omitempty"`
	Message     string    `json:"message"`
	Err         error     `json:"-"`
}

func (e *OperationError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Code, e.Err)
}

func operationError(code ErrorCode, path, destination string, err error) *OperationError {
	return &OperationError{Code: code, Path: path, Destination: destination, Message: err.Error(), Err: err}
}

type Source struct {
	RepoRoot  string `json:"repo_root"`
	SkillPath string `json:"path"`
	Commit    string `json:"commit"`
	Digest    Digest `json:"digest"`
}

type DestinationID string

const (
	DestinationClaude DestinationID = "claude"
	DestinationAgents DestinationID = "agents"
)

type EntryKind string

const (
	KindAbsent           EntryKind = "absent"
	KindDirectory        EntryKind = "directory"
	KindCorrectSymlink   EntryKind = "correct_symlink"
	KindDivergentSymlink EntryKind = "divergent_symlink"
	KindUnsupported      EntryKind = "unsupported"
)

type Operation string

const (
	OperationInstall   Operation = "install"
	OperationStatus    Operation = "status"
	OperationUninstall Operation = "uninstall"
	OperationRestore   Operation = "restore"
)

type ActionKind string

const (
	ActionCreateSymlink        ActionKind = "create_symlink"
	ActionReplaceWithSymlink   ActionKind = "replace_with_symlink"
	ActionRemoveManagedSymlink ActionKind = "remove_managed_symlink"
	ActionRestorePreimage      ActionKind = "restore_preimage"
	ActionNoOp                 ActionKind = "no_op"
)

type Destination struct {
	ID   DestinationID `json:"id"`
	Path string        `json:"path"`
}

type DestinationState struct {
	ID              DestinationID `json:"id"`
	Path            string        `json:"path"`
	Kind            EntryKind     `json:"kind"`
	Digest          Digest        `json:"digest,omitempty"`
	LexicalTarget   string        `json:"lexical_target,omitempty"`
	CanonicalTarget string        `json:"canonical_target,omitempty"`
}

type Action struct {
	Kind        ActionKind       `json:"kind"`
	Destination DestinationState `json:"destination"`
}

type Plan struct {
	Operation Operation `json:"operation"`
	Source    *Source   `json:"source,omitempty"`
	Actions   []Action  `json:"actions"`
	Digest    Digest    `json:"digest"`
}
