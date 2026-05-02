package safety

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSandboxResolve(t *testing.T) {
	dir := t.TempDir()
	s, err := NewSandbox(dir)
	if err != nil {
		t.Fatalf("new sandbox: %v", err)
	}

	good := []string{".", "file.txt", "sub/dir/file.go", "deep/./nested"}
	for _, p := range good {
		if _, err := s.Resolve(p); err != nil {
			t.Errorf("expected %q to resolve, got %v", p, err)
		}
	}

	bad := []string{
		"../escape",
		"/etc/passwd",
		`C:\Windows`,
		`\\server\share`,
		"sub/../../escape",
	}
	for _, p := range bad {
		if _, err := s.Resolve(p); !errors.Is(err, ErrOutsideSandbox) {
			t.Errorf("expected %q to be rejected, got %v", p, err)
		}
	}
}

func TestSandboxRejectsSymlinkEscape(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(dir, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	s, err := NewSandbox(dir)
	if err != nil {
		t.Fatalf("new sandbox: %v", err)
	}
	if _, err := s.Resolve("escape"); !errors.Is(err, ErrOutsideSandbox) {
		t.Fatalf("expected symlink escape to be rejected, got %v", err)
	}
}
