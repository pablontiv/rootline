package migrate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLogAppendCreatesFile(t *testing.T) {
	dir := t.TempDir()
	ml := NewMigrationLog(dir)

	entry := MigrationEntry{
		Timestamp:     "2025-01-15T10:30:00Z",
		Type:          "rename",
		Details:       map[string]string{"old_field": "titulo", "new_field": "title"},
		FilesAffected: 3,
	}

	if err := ml.Append(entry); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// Verify file was created.
	logPath := filepath.Join(dir, LogFileName)
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("log file not created: %v", err)
	}

	// Verify content is valid JSON.
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	var parsed MigrationEntry
	if err := json.Unmarshal([]byte(lines[0]), &parsed); err != nil {
		t.Fatalf("invalid JSON line: %v", err)
	}

	if parsed.Type != "rename" {
		t.Errorf("type = %q, want %q", parsed.Type, "rename")
	}
	if parsed.FilesAffected != 3 {
		t.Errorf("files_affected = %d, want 3", parsed.FilesAffected)
	}
	if parsed.Details["old_field"] != "titulo" {
		t.Errorf("details.old_field = %q, want %q", parsed.Details["old_field"], "titulo")
	}
}

func TestLogAppendAppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	ml := NewMigrationLog(dir)

	entry1 := MigrationEntry{
		Timestamp:     "2025-01-15T10:30:00Z",
		Type:          "rename",
		Details:       map[string]string{"old_field": "a", "new_field": "b"},
		FilesAffected: 1,
	}
	entry2 := MigrationEntry{
		Timestamp:     "2025-01-15T11:00:00Z",
		Type:          "rename",
		Details:       map[string]string{"old_field": "c", "new_field": "d"},
		FilesAffected: 5,
	}

	if err := ml.Append(entry1); err != nil {
		t.Fatalf("first append: %v", err)
	}
	if err := ml.Append(entry2); err != nil {
		t.Fatalf("second append: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, LogFileName))
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
}

func TestLogReadParsesAllEntries(t *testing.T) {
	dir := t.TempDir()
	ml := NewMigrationLog(dir)

	entries := []MigrationEntry{
		{Timestamp: "2025-01-15T10:00:00Z", Type: "rename", Details: map[string]string{"old_field": "a", "new_field": "b"}, FilesAffected: 2},
		{Timestamp: "2025-01-15T11:00:00Z", Type: "rename", Details: map[string]string{"old_field": "c", "new_field": "d"}, FilesAffected: 4},
		{Timestamp: "2025-01-15T12:00:00Z", Type: "rename", Details: map[string]string{"old_field": "e", "new_field": "f"}, FilesAffected: 1},
	}

	for _, e := range entries {
		if err := ml.Append(e); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	read, err := ml.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if len(read) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(read))
	}

	for i, e := range read {
		if e.Timestamp != entries[i].Timestamp {
			t.Errorf("entry %d: timestamp = %q, want %q", i, e.Timestamp, entries[i].Timestamp)
		}
		if e.FilesAffected != entries[i].FilesAffected {
			t.Errorf("entry %d: files_affected = %d, want %d", i, e.FilesAffected, entries[i].FilesAffected)
		}
	}
}

func TestLogReadNonExistentFile(t *testing.T) {
	dir := t.TempDir()
	ml := NewMigrationLog(dir)

	entries, err := ml.Read()
	if err != nil {
		t.Fatalf("Read should not error for non-existent file: %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil entries, got %v", entries)
	}
}

func TestLogEntryHasRequiredFields(t *testing.T) {
	dir := t.TempDir()
	ml := NewMigrationLog(dir)

	entry := NewRenameEntry("old", "new", 7)
	if err := ml.Append(entry); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	read, err := ml.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if len(read) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(read))
	}

	e := read[0]

	// Verify timestamp is ISO 8601.
	if _, parseErr := time.Parse(time.RFC3339, e.Timestamp); parseErr != nil {
		t.Errorf("timestamp %q is not valid ISO 8601: %v", e.Timestamp, parseErr)
	}

	// Verify type.
	if e.Type != "rename" {
		t.Errorf("type = %q, want %q", e.Type, "rename")
	}

	// Verify details.
	if e.Details["old_field"] != "old" {
		t.Errorf("details.old_field = %q, want %q", e.Details["old_field"], "old")
	}
	if e.Details["new_field"] != "new" {
		t.Errorf("details.new_field = %q, want %q", e.Details["new_field"], "new")
	}

	// Verify files_affected.
	if e.FilesAffected != 7 {
		t.Errorf("files_affected = %d, want 7", e.FilesAffected)
	}
}

func TestLogJSONLinesFormat(t *testing.T) {
	dir := t.TempDir()
	ml := NewMigrationLog(dir)

	for i := 0; i < 3; i++ {
		entry := NewRenameEntry("old", "new", i)
		if err := ml.Append(entry); err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}

	content, err := os.ReadFile(filepath.Join(dir, LogFileName))
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}

	// Each line must be valid JSON (JSON Lines format).
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	for i, line := range lines {
		if !json.Valid([]byte(line)) {
			t.Errorf("line %d is not valid JSON: %s", i, line)
		}
		// Each line should be a single line (no embedded newlines).
		if strings.Contains(line, "\n") {
			t.Errorf("line %d contains embedded newline", i)
		}
	}
}
