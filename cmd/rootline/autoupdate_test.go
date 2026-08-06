package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/pablontiv/picokit/autoupdate"
)

type updaterStub struct {
	applyErr error
	fetchErr error
}

func (u updaterStub) ApplyStagedIfAvailable() error { return u.applyErr }

func (u updaterStub) FetchAndStage(string) error { return u.fetchErr }

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

func TestRootlineUpdaterUsesSameMajorPolicy(t *testing.T) {
	tests := []struct {
		name      string
		current   string
		candidate string
		want      bool
	}{
		{name: "same major allowed", current: "v5.2.0", candidate: "v5.3.0", want: true},
		{name: "cross major withheld", current: "v5.2.0", candidate: "v6.0.0", want: false},
		{name: "same pre-one minor allowed", current: "v0.5.2", candidate: "v0.5.3", want: true},
		{name: "cross pre-one minor withheld", current: "v0.5.2", candidate: "v0.6.0", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := newRootlineUpdater(tt.current)
			if got := u.VersionPolicy(u.CurrentVersion, tt.candidate); got != tt.want {
				t.Fatalf("VersionPolicy(%q, %q) = %v, want %v", u.CurrentVersion, tt.candidate, got, tt.want)
			}
		})
	}
}

func TestRunWithUpdaterWritesNoticeExactlyOnceOnlyWhenWithheld(t *testing.T) {
	withheld := &autoupdate.UpdateWithheldError{
		CurrentVersion:   "v5.2.0",
		CandidateVersion: "v6.0.0",
	}

	tests := []struct {
		name     string
		updater  updaterStub
		wantLine string
	}{
		{
			name:    "no notice without withheld update",
			updater: updaterStub{fetchErr: errors.New("network unavailable")},
		},
		{
			name:     "one notice when both updater paths withhold",
			updater:  updaterStub{applyErr: withheld, fetchErr: withheld},
			wantLine: "rootline: incompatible update withheld: current v5.2.0, available v6.0.0; run the installer to upgrade deliberately\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			executed := false

			runWithUpdaterUsing("v5.2.0", func() { executed = true }, tt.updater, &stderr)

			if !executed {
				t.Fatal("execute was not called")
			}
			if got := stderr.String(); got != tt.wantLine {
				t.Fatalf("stderr = %q, want %q", got, tt.wantLine)
			}
			if strings.Count(stderr.String(), "incompatible update withheld") > 1 {
				t.Fatalf("withheld notice emitted more than once: %q", stderr.String())
			}
		})
	}
}
