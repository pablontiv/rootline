package main

import (
	"testing"

	"github.com/pablontiv/picokit/autoupdate"
)

func TestWiring_NewWithTwoArgs(t *testing.T) {
	u := autoupdate.New("pablontiv/rootline", "rootline")
	if u.EnvDisable != "" {
		t.Fatalf("EnvDisable = %q, want empty", u.EnvDisable)
	}
	if u.Repo != "pablontiv/rootline" {
		t.Fatalf("Repo = %q, want %q", u.Repo, "pablontiv/rootline")
	}
	if u.Binary != "rootline" {
		t.Fatalf("Binary = %q, want %q", u.Binary, "rootline")
	}
}

func TestWiring_VersionDev_NoSideEffects(t *testing.T) {
	called := false
	runWithUpdater("dev", func() { called = true })
	if !called {
		t.Fatal("execute was not called by runWithUpdater")
	}
}
