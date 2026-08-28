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
