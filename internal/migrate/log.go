package migrate

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LogFileName is the name of the migration log file.
const LogFileName = ".rootline-migrations"

// MigrationEntry represents a single entry in the migration log.
type MigrationEntry struct {
	Timestamp     string            `json:"timestamp"`
	Type          string            `json:"type"`
	Details       map[string]string `json:"details"`
	FilesAffected int               `json:"files_affected"`
}

// MigrationLog provides append-only access to the migration log file.
type MigrationLog struct {
	rootPath string
}

// NewMigrationLog creates a MigrationLog rooted at the given directory.
func NewMigrationLog(rootPath string) *MigrationLog {
	return &MigrationLog{rootPath: rootPath}
}

// logPath returns the absolute path to the migration log file.
func (ml *MigrationLog) logPath() string {
	return filepath.Join(ml.rootPath, LogFileName)
}

// Append adds a new entry to the migration log.
// It creates the file if it does not exist.
func (ml *MigrationLog) Append(entry MigrationEntry) error {
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling log entry: %w", err)
	}

	f, err := os.OpenFile(ml.logPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("opening migration log: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		_ = f.Close()
		return fmt.Errorf("writing log entry: %w", err)
	}

	return f.Close()
}

// Read parses all entries from the migration log.
// Returns an empty slice if the file does not exist.
func (ml *MigrationLog) Read() ([]MigrationEntry, error) {
	f, err := os.Open(ml.logPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("opening migration log: %w", err)
	}
	defer func() { _ = f.Close() }()

	var entries []MigrationEntry
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line == "" {
			continue
		}
		var entry MigrationEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("parsing line %d: %w", lineNum, err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading migration log: %w", err)
	}

	return entries, nil
}

// NewRenameEntry creates a MigrationEntry for a rename operation.
func NewRenameEntry(oldField, newField string, filesAffected int) MigrationEntry {
	return MigrationEntry{
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Type:          "rename",
		Details:       map[string]string{"old_field": oldField, "new_field": newField},
		FilesAffected: filesAffected,
	}
}
