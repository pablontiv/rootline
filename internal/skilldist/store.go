package skilldist

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	receiptsFileName = "receipts.jsonl"
	backupsDirName   = "backups"
)

type Store struct {
	root string
}

func NewStore(stateRoot string) *Store {
	return &Store{root: filepath.Join(stateRoot, "rootline", "skill")}
}

func (s *Store) Reserve(receiptID string) error {
	if err := validateStoreChildName(receiptID, "receipt ID"); err != nil {
		return err
	}
	if err := os.MkdirAll(s.backupsRoot(), 0o700); err != nil {
		return err
	}
	return os.Mkdir(s.receiptBackupDir(receiptID), 0o700)
}

func (s *Store) Append(receipt Receipt) error {
	if receipt.ID == "" {
		return fmt.Errorf("receipt ID is required")
	}
	if _, err := s.loadReceipt(receipt.ID); err == nil {
		return fmt.Errorf("duplicate receipt ID %q", receipt.ID)
	} else if !errors.Is(err, errReceiptNotFound) {
		return err
	}

	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(s.receiptsPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	line, err := json.Marshal(normalizeReceipt(receipt))
	if err != nil {
		return errors.Join(err, file.Close())
	}
	line = append(line, '\n')
	n, writeErr := file.Write(line)
	if writeErr == nil && n != len(line) {
		writeErr = io.ErrShortWrite
	}
	syncErr := file.Sync()
	closeErr := file.Close()
	return errors.Join(writeErr, syncErr, closeErr)
}

func (s *Store) Load(id string) (Receipt, error) {
	return s.loadReceipt(id)
}

func (s *Store) Latest() (Receipt, bool, error) {
	receipts, err := s.scanReceipts()
	if err != nil {
		return Receipt{}, false, err
	}
	if len(receipts) == 0 {
		return Receipt{}, false, nil
	}
	return receipts[len(receipts)-1], true, nil
}

func (s *Store) Backup(receiptID string, state DestinationState) (Backup, error) {
	if err := validateStoreChildName(receiptID, "receipt ID"); err != nil {
		return Backup{}, err
	}
	if err := validateStoreChildName(string(state.ID), "destination ID"); err != nil {
		return Backup{}, err
	}
	if _, err := os.Stat(s.receiptBackupDir(receiptID)); err != nil {
		if os.IsNotExist(err) {
			return Backup{}, fmt.Errorf("reserved receipt backup directory %q does not exist", receiptID)
		}
		return Backup{}, err
	}

	backup := Backup{
		Destination:  state.ID,
		OriginalPath: state.Path,
		Kind:         state.Kind,
		Digest:       state.Digest,
	}

	switch state.Kind {
	case KindAbsent:
		if _, err := os.Lstat(state.Path); err == nil {
			return Backup{}, operationError(ErrPreimageDigestChanged, state.Path, string(state.ID), fmt.Errorf("destination exists but preimage was recorded absent"))
		} else if !os.IsNotExist(err) {
			return Backup{}, err
		}
		return backup, nil
	case KindDirectory:
		return s.backupDirectory(receiptID, state, backup)
	case KindCorrectSymlink, KindDivergentSymlink:
		return s.backupSymlink(receiptID, state, backup)
	default:
		return Backup{}, operationError(ErrUnsupportedFileType, state.Path, string(state.ID), fmt.Errorf("unsupported backup kind %q", state.Kind))
	}
}

func (s *Store) VerifyBackup(backup Backup) error {
	switch backup.Kind {
	case KindAbsent:
		return nil
	case KindDirectory:
		if backup.StoredPath == "" {
			return operationError(ErrVerificationFailed, backup.OriginalPath, string(backup.Destination), fmt.Errorf("directory backup has no stored path"))
		}
		digest, err := DigestTree(backup.StoredPath)
		if err != nil {
			return err
		}
		if digest != backup.Digest {
			return operationError(ErrVerificationFailed, backup.StoredPath, string(backup.Destination), fmt.Errorf("backup digest %q does not match preimage digest %q", digest, backup.Digest))
		}
		return nil
	case KindCorrectSymlink, KindDivergentSymlink:
		if backup.LinkTarget == "" {
			return operationError(ErrVerificationFailed, backup.OriginalPath, string(backup.Destination), fmt.Errorf("symlink backup has no link target"))
		}
		if backup.StoredPath != "" {
			return operationError(ErrVerificationFailed, backup.StoredPath, string(backup.Destination), fmt.Errorf("symlink backup unexpectedly has a stored tree"))
		}
		return nil
	default:
		return operationError(ErrUnsupportedFileType, backup.OriginalPath, string(backup.Destination), fmt.Errorf("unsupported backup kind %q", backup.Kind))
	}
}

func (s *Store) RestoreBackup(backup Backup, destination string) error {
	if err := ensureDestinationAbsent(destination, backup.Destination); err != nil {
		return err
	}

	switch backup.Kind {
	case KindAbsent:
		return nil
	case KindDirectory:
		if err := copyDirectory(backup.StoredPath, destination); err != nil {
			return err
		}
		digest, err := DigestTree(destination)
		if err != nil {
			return err
		}
		if digest != backup.Digest {
			return operationError(ErrVerificationFailed, destination, string(backup.Destination), fmt.Errorf("restored digest %q does not match backup digest %q", digest, backup.Digest))
		}
		return nil
	case KindCorrectSymlink, KindDivergentSymlink:
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return err
		}
		if err := os.Symlink(backup.LinkTarget, destination); err != nil {
			return err
		}
		linkTarget, err := os.Readlink(destination)
		if err != nil {
			return err
		}
		if linkTarget != backup.LinkTarget {
			return operationError(ErrVerificationFailed, destination, string(backup.Destination), fmt.Errorf("restored symlink target %q does not match backup target %q", linkTarget, backup.LinkTarget))
		}
		return nil
	default:
		return operationError(ErrUnsupportedFileType, destination, string(backup.Destination), fmt.Errorf("unsupported restore kind %q", backup.Kind))
	}
}

func (s *Store) receiptsPath() string {
	return filepath.Join(s.root, receiptsFileName)
}

func (s *Store) backupsRoot() string {
	return filepath.Join(s.root, backupsDirName)
}

func (s *Store) receiptBackupDir(receiptID string) string {
	return filepath.Join(s.backupsRoot(), receiptID)
}

func (s *Store) destinationBackupPath(receiptID string, destination DestinationID) string {
	return filepath.Join(s.receiptBackupDir(receiptID), string(destination))
}

var errReceiptNotFound = errors.New("receipt not found")

func (s *Store) loadReceipt(id string) (Receipt, error) {
	receipts, err := s.scanReceipts()
	if err != nil {
		return Receipt{}, err
	}
	for _, receipt := range receipts {
		if receipt.ID == id {
			return receipt, nil
		}
	}
	return Receipt{}, errReceiptNotFound
}

func (s *Store) scanReceipts() ([]Receipt, error) {
	file, err := os.Open(s.receiptsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	seen := make(map[string]struct{})
	receipts := []Receipt{}
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			return nil, fmt.Errorf("malformed receipt JSONL at line %d: empty line", lineNumber)
		}
		var receipt Receipt
		if err := json.Unmarshal(line, &receipt); err != nil {
			return nil, fmt.Errorf("malformed receipt JSONL at line %d: %w", lineNumber, err)
		}
		if receipt.ID == "" {
			return nil, fmt.Errorf("malformed receipt JSONL at line %d: missing receipt ID", lineNumber)
		}
		if _, ok := seen[receipt.ID]; ok {
			return nil, fmt.Errorf("duplicate receipt ID %q", receipt.ID)
		}
		seen[receipt.ID] = struct{}{}
		receipts = append(receipts, normalizeReceipt(receipt))
	}
	scanErr := scanner.Err()
	closeErr := file.Close()
	if scanErr != nil || closeErr != nil {
		return nil, errors.Join(scanErr, closeErr)
	}
	return receipts, nil
}

func (s *Store) backupDirectory(receiptID string, state DestinationState, backup Backup) (Backup, error) {
	info, err := os.Lstat(state.Path)
	if err != nil {
		return Backup{}, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return Backup{}, operationError(ErrPreimageDigestChanged, state.Path, string(state.ID), fmt.Errorf("destination is no longer a directory"))
	}
	observedDigest, err := DigestTree(state.Path)
	if err != nil {
		return Backup{}, err
	}
	if state.Digest != "" && observedDigest != state.Digest {
		return Backup{}, operationError(ErrPreimageDigestChanged, state.Path, string(state.ID), fmt.Errorf("preimage digest %q does not match observed digest %q", state.Digest, observedDigest))
	}
	backup.Digest = observedDigest
	backup.StoredPath = s.destinationBackupPath(receiptID, state.ID)
	if err := copyDirectory(state.Path, backup.StoredPath); err != nil {
		if copyDirectoryCreatedDestination(err) {
			_ = os.RemoveAll(backup.StoredPath)
		}
		return Backup{}, err
	}
	if err := s.VerifyBackup(backup); err != nil {
		return Backup{}, err
	}
	return backup, nil
}

func (s *Store) backupSymlink(receiptID string, state DestinationState, backup Backup) (Backup, error) {
	linkTarget, err := os.Readlink(state.Path)
	if err != nil {
		return Backup{}, err
	}
	if state.LexicalTarget != "" && linkTarget != state.LexicalTarget {
		return Backup{}, operationError(ErrPreimageDigestChanged, state.Path, string(state.ID), fmt.Errorf("preimage symlink target %q does not match observed target %q", state.LexicalTarget, linkTarget))
	}
	backupSlot := s.destinationBackupPath(receiptID, state.ID)
	if err := os.Mkdir(backupSlot, 0o700); err != nil {
		return Backup{}, err
	}
	backup.LinkTarget = linkTarget
	backup.StoredPath = ""
	if err := s.VerifyBackup(backup); err != nil {
		return Backup{}, err
	}
	return backup, nil
}

func normalizeReceipt(receipt Receipt) Receipt {
	if receipt.Actions == nil {
		receipt.Actions = []ActionResult{}
	}
	if receipt.Backups == nil {
		receipt.Backups = []Backup{}
	}
	if receipt.Errors == nil {
		receipt.Errors = []OperationError{}
	}
	return receipt
}

func validateStoreChildName(name, label string) error {
	if name == "" {
		return fmt.Errorf("%s is required", label)
	}
	if strings.ContainsAny(name, `/\\`) || filepath.Clean(name) != name || name == "." || name == ".." {
		return fmt.Errorf("%s %q is not a safe store child name", label, name)
	}
	return nil
}

func ensureDestinationAbsent(destination string, id DestinationID) error {
	if _, err := os.Lstat(destination); err == nil {
		return operationError(ErrRestoreConflict, destination, string(id), fmt.Errorf("destination already exists"))
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func copyDirectory(source, destination string) error {
	if source == "" || destination == "" {
		return fmt.Errorf("source and destination directories are required")
	}
	rootInfo, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("source %q is not a directory", source)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return err
	}
	if err := os.Mkdir(destination, 0o700); err != nil {
		return err
	}

	sourceRoot, err := os.OpenRoot(source)
	if err != nil {
		return copyDirectoryFailure{err: err, createdDestination: true}
	}
	destinationRoot, err := os.OpenRoot(destination)
	if err != nil {
		return copyDirectoryFailure{err: errors.Join(err, sourceRoot.Close()), createdDestination: true}
	}

	type dirMode struct {
		rel  string
		mode fs.FileMode
	}
	dirs := []dirMode{{rel: ".", mode: rootInfo.Mode().Perm()}}

	walkErr := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		mode := info.Mode()
		switch {
		case mode&os.ModeSymlink != 0:
			target, err := sourceRoot.Readlink(rel)
			if err != nil {
				return err
			}
			return destinationRoot.Symlink(target, rel)
		case entry.IsDir():
			if err := destinationRoot.Mkdir(rel, 0o700); err != nil {
				return err
			}
			dirs = append(dirs, dirMode{rel: rel, mode: mode.Perm()})
			return nil
		case mode.IsRegular():
			return copyRegularFile(sourceRoot, destinationRoot, rel, mode.Perm())
		default:
			return operationError(ErrUnsupportedFileType, path, "", fmt.Errorf("unsupported file type %s", mode.Type()))
		}
	})
	var chmodErr error
	if walkErr == nil {
		for i := len(dirs) - 1; i >= 0; i-- {
			if err := destinationRoot.Chmod(dirs[i].rel, dirs[i].mode); err != nil {
				chmodErr = err
				break
			}
		}
	}
	closeDestinationErr := destinationRoot.Close()
	closeSourceErr := sourceRoot.Close()
	if err := errors.Join(walkErr, chmodErr, closeDestinationErr, closeSourceErr); err != nil {
		return copyDirectoryFailure{err: err, createdDestination: true}
	}
	return nil
}

type copyDirectoryFailure struct {
	err                error
	createdDestination bool
}

func (e copyDirectoryFailure) Error() string {
	return e.err.Error()
}

func (e copyDirectoryFailure) Unwrap() error {
	return e.err
}

func copyDirectoryCreatedDestination(err error) bool {
	var failure copyDirectoryFailure
	return errors.As(err, &failure) && failure.createdDestination
}

func copyRegularFile(sourceRoot, destinationRoot *os.Root, rel string, mode fs.FileMode) error {
	input, err := sourceRoot.Open(rel)
	if err != nil {
		return err
	}
	output, err := destinationRoot.OpenFile(rel, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return errors.Join(err, input.Close())
	}
	_, copyErr := io.Copy(output, input)
	chmodErr := output.Chmod(mode)
	closeOutputErr := output.Close()
	closeInputErr := input.Close()
	return errors.Join(copyErr, chmodErr, closeOutputErr, closeInputErr)
}
