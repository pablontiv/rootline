//go:build darwin || linux || windows

package skilldist

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicPublishNoReplacePublishesDirectoryWhenDestinationAbsent(t *testing.T) {
	parent := t.TempDir()
	candidate := filepath.Join(parent, "candidate")
	destination := filepath.Join(parent, "destination")
	mustWriteSkillFile(t, candidate, "SKILL.md", "candidate")

	if err := atomicPublishNoReplace(candidate, destination); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(candidate); !os.IsNotExist(err) {
		t.Fatalf("candidate remained after publish: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(destination, "SKILL.md"))
	if err != nil || string(data) != "candidate" {
		t.Fatalf("published data = %q, err=%v", data, err)
	}
}

func TestAtomicPublishNoReplaceReportsConflictWithoutOverwrite(t *testing.T) {
	parent := t.TempDir()
	candidate := filepath.Join(parent, "candidate")
	destination := filepath.Join(parent, "destination")
	mustWriteSkillFile(t, candidate, "SKILL.md", "candidate")
	mustWriteSkillFile(t, destination, "SKILL.md", "external")

	err := atomicPublishNoReplace(candidate, destination)
	if !errors.Is(err, errAtomicPublishDestinationExists) {
		t.Fatalf("publish error = %v, want destination-exists sentinel", err)
	}
	candidateData, candidateErr := os.ReadFile(filepath.Join(candidate, "SKILL.md"))
	if candidateErr != nil || string(candidateData) != "candidate" {
		t.Fatalf("candidate data = %q, err=%v", candidateData, candidateErr)
	}
	destinationData, destinationErr := os.ReadFile(filepath.Join(destination, "SKILL.md"))
	if destinationErr != nil || string(destinationData) != "external" {
		t.Fatalf("destination data = %q, err=%v", destinationData, destinationErr)
	}
}
