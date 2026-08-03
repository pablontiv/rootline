package fix

import (
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// absPrefix is the platform's root prefix so that constructed test paths are
// genuinely absolute on every OS (filepath.IsAbs(`\root`) is false on Windows).
var absPrefix = func() string {
	if runtime.GOOS == "windows" {
		return `C:\`
	}
	return "/"
}()

// abs builds an absolute, platform-native path from slash-separated segments.
func abs(parts ...string) string {
	return filepath.Join(absPrefix, filepath.Join(parts...))
}

func TestContainPath(t *testing.T) {
	root := abs("root")
	nested := abs("root", "dir")

	tests := []struct {
		name       string
		root       string
		path       string
		policy     ContainmentPolicy
		wantErr    bool
		wantMsg    string // substring expected in the error message
		wantResult string // expected resolved path (only checked when wantErr is false)
	}{
		// --- relative paths that stay inside root ---
		{
			name:       "RejectAbsolute/relative file",
			root:       root,
			path:       "docs/page.md",
			policy:     PolicyRejectAbsolute,
			wantResult: abs("root", "docs", "page.md"),
		},
		{
			name:       "RejectAbsolute/relative nested directories",
			root:       root,
			path:       "a/b/c.md",
			policy:     PolicyRejectAbsolute,
			wantResult: abs("root", "a", "b", "c.md"),
		},
		{
			name:       "RejectAbsolute/dot resolves to root",
			root:       root,
			path:       ".",
			policy:     PolicyRejectAbsolute,
			wantResult: root,
		},
		{
			name:       "RejectAbsolute/empty path resolves to root",
			root:       root,
			path:       "",
			policy:     PolicyRejectAbsolute,
			wantResult: root,
		},
		{
			name:       "RejectAbsolute/trailing separator is cleaned",
			root:       root,
			path:       "docs/",
			policy:     PolicyRejectAbsolute,
			wantResult: abs("root", "docs"),
		},
		{
			name:       "RejectAbsolute/nested dotdot staying inside",
			root:       root,
			path:       "a/../b/file.md",
			policy:     PolicyRejectAbsolute,
			wantResult: abs("root", "b", "file.md"),
		},
		{
			// Spec Scenario 8: a/../../dir/file.md under /root/dir resolves back
			// to /root/dir/file.md, which is still inside the root.
			name:       "RejectAbsolute/scenario8 dotdot returns inside root",
			root:       nested,
			path:       "a/../../dir/file.md",
			policy:     PolicyRejectAbsolute,
			wantResult: abs("root", "dir", "file.md"),
		},
		{
			name:       "RejectAbsolute/non-existent target is allowed",
			root:       root,
			path:       "does/not/exist.md",
			policy:     PolicyRejectAbsolute,
			wantResult: abs("root", "does", "not", "exist.md"),
		},
		{
			name:       "AcceptAbsolute/relative path still allowed",
			root:       root,
			path:       "docs/page.md",
			policy:     PolicyAcceptAbsolute,
			wantResult: abs("root", "docs", "page.md"),
		},

		// --- relative paths that escape root ---
		{
			name:    "RejectAbsolute/parent escape",
			root:    root,
			path:    "../outside.md",
			policy:  PolicyRejectAbsolute,
			wantErr: true,
			wantMsg: "escapes root",
		},
		{
			name:    "RejectAbsolute/deep parent escape",
			root:    root,
			path:    "../../../etc/shadow",
			policy:  PolicyRejectAbsolute,
			wantErr: true,
			wantMsg: "escapes root",
		},
		{
			name:    "RejectAbsolute/bare dotdot",
			root:    root,
			path:    "..",
			policy:  PolicyRejectAbsolute,
			wantErr: true,
			wantMsg: "escapes root",
		},
		{
			// Spec Scenario 9: ../.. under /root/dir resolves above the root.
			name:    "RejectAbsolute/scenario9 double dotdot from nested root",
			root:    nested,
			path:    "../..",
			policy:  PolicyRejectAbsolute,
			wantErr: true,
			wantMsg: "escapes root",
		},
		{
			name:    "RejectAbsolute/escape hidden behind a subdirectory",
			root:    root,
			path:    "docs/../../outside/.stem",
			policy:  PolicyRejectAbsolute,
			wantErr: true,
			wantMsg: "escapes root",
		},
		{
			name:    "AcceptAbsolute/relative escape still rejected",
			root:    root,
			path:    "../outside.md",
			policy:  PolicyAcceptAbsolute,
			wantErr: true,
			wantMsg: "escapes root",
		},

		// --- absolute paths under PolicyRejectAbsolute ---
		{
			// Spec Scenario 3: absolute input is a policy violation, not an I/O error.
			name:    "RejectAbsolute/absolute outside root",
			root:    root,
			path:    abs("etc", "passwd"),
			policy:  PolicyRejectAbsolute,
			wantErr: true,
			wantMsg: "absolute paths are not allowed",
		},
		{
			// Even a contained absolute path is refused: the policy is about the
			// shape of the input, not only about where it lands.
			name:    "RejectAbsolute/absolute inside root",
			root:    root,
			path:    abs("root", "docs", "page.md"),
			policy:  PolicyRejectAbsolute,
			wantErr: true,
			wantMsg: "absolute paths are not allowed",
		},

		// --- absolute paths under PolicyAcceptAbsolute ---
		{
			// Spec Scenario 6: schema propose emits absolute targets; they must survive.
			name:       "AcceptAbsolute/absolute inside root",
			root:       root,
			path:       abs("root", "docs", ".stem"),
			policy:     PolicyAcceptAbsolute,
			wantResult: abs("root", "docs", ".stem"),
		},
		{
			name:       "AcceptAbsolute/absolute equal to root",
			root:       root,
			path:       root,
			policy:     PolicyAcceptAbsolute,
			wantResult: root,
		},
		{
			// Spec Scenario 7: absolute target above the root.
			name:    "AcceptAbsolute/absolute outside root",
			root:    root,
			path:    abs("tmp", ".stem"),
			policy:  PolicyAcceptAbsolute,
			wantErr: true,
			wantMsg: "escapes root",
		},
		{
			// Spec Scenario 5: the absolute target only escapes after cleaning, which
			// is exactly the case filepath.Join(root, path) would have hidden.
			name:    "AcceptAbsolute/absolute escaping after clean",
			root:    root,
			path:    abs("root", "..", "etc", ".stem"),
			policy:  PolicyAcceptAbsolute,
			wantErr: true,
			wantMsg: "escapes root",
		},
		{
			// String-prefix containment checks accept this; filepath.Rel does not.
			name:    "AcceptAbsolute/sibling directory sharing the root prefix",
			root:    root,
			path:    abs("rootother", "file.md"),
			policy:  PolicyAcceptAbsolute,
			wantErr: true,
			wantMsg: "escapes root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ContainPath(tt.root, tt.path, tt.policy)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("ContainPath(%q, %q) = %q, want error", tt.root, tt.path, got)
				}
				if !strings.Contains(err.Error(), tt.wantMsg) {
					t.Errorf("error = %q, want it to contain %q", err.Error(), tt.wantMsg)
				}
				if got != "" {
					t.Errorf("resolved path = %q, want empty string on error", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("ContainPath(%q, %q) returned unexpected error: %v", tt.root, tt.path, err)
			}
			if got != tt.wantResult {
				t.Errorf("resolved path = %q, want %q", got, tt.wantResult)
			}
		})
	}
}

func TestContainPathResolvesRelativeRoot(t *testing.T) {
	// A relative root must be resolved against the working directory before the
	// containment check, otherwise the returned path is not usable for a write.
	got, err := ContainPath(".", "docs/page.md", PolicyRejectAbsolute)
	if err != nil {
		t.Fatalf("ContainPath returned unexpected error: %v", err)
	}

	cwd, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("resolving working directory: %v", err)
	}
	want := filepath.Join(cwd, "docs", "page.md")
	if got != want {
		t.Errorf("resolved path = %q, want %q", got, want)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("resolved path %q is not absolute", got)
	}
}

func TestContainPathErrorCarriesContext(t *testing.T) {
	root := abs("root")
	_, err := ContainPath(root, "../../../etc/shadow", PolicyRejectAbsolute)
	if err == nil {
		t.Fatal("expected a containment error")
	}

	var cerr *ContainmentError
	if !errors.As(err, &cerr) {
		t.Fatalf("error type = %T, want *ContainmentError", err)
	}
	if cerr.Requested != "../../../etc/shadow" {
		t.Errorf("Requested = %q, want the original proposal path", cerr.Requested)
	}
	if cerr.Root != root {
		t.Errorf("Root = %q, want %q", cerr.Root, root)
	}
	if cerr.Message != "escapes root" {
		t.Errorf("Message = %q, want %q", cerr.Message, "escapes root")
	}

	// The message goes straight into JSON output, so it must stay single-line
	// and must name both the offending path and the root.
	msg := err.Error()
	if strings.ContainsAny(msg, "\n\r") {
		t.Errorf("error message %q contains a line break", msg)
	}
	if !strings.Contains(msg, "../../../etc/shadow") {
		t.Errorf("error message %q does not name the requested path", msg)
	}
	if !strings.Contains(msg, root) {
		t.Errorf("error message %q does not name the root", msg)
	}
}

func TestContainPathWindowsVolumes(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("drive letters and UNC shares are only absolute paths on Windows")
	}

	t.Run("volume mismatch is a containment error", func(t *testing.T) {
		// Spec Scenario 11: filepath.Rel cannot relate two different volumes and
		// its error must surface as a containment violation, never be swallowed.
		if _, err := ContainPath(`C:\root`, `D:\file.md`, PolicyAcceptAbsolute); err == nil {
			t.Fatal("expected a containment error across volumes")
		}
	})

	t.Run("drive letter case is normalized", func(t *testing.T) {
		got, err := ContainPath(`C:\root`, `c:\root\docs\page.md`, PolicyAcceptAbsolute)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == "" {
			t.Error("expected a resolved path")
		}
	})

	t.Run("UNC share is contained", func(t *testing.T) {
		got, err := ContainPath(`\\server\share\root`, `docs\page.md`, PolicyRejectAbsolute)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if want := `\\server\share\root\docs\page.md`; got != want {
			t.Errorf("resolved path = %q, want %q", got, want)
		}
	})

	t.Run("UNC escape is rejected", func(t *testing.T) {
		if _, err := ContainPath(`\\server\share\root`, `..\..\other\file.md`, PolicyRejectAbsolute); err == nil {
			t.Fatal("expected a containment error")
		}
	})

	t.Run("mixed separators are contained", func(t *testing.T) {
		got, err := ContainPath(`C:\root`, `docs/page.md`, PolicyRejectAbsolute)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if want := `C:\root\docs\page.md`; got != want {
			t.Errorf("resolved path = %q, want %q", got, want)
		}
	})
}
