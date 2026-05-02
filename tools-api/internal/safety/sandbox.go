// Package safety enforces the OpenCodeForge security model:
//   - every filesystem path is resolved relative to a single workspace root,
//   - shell commands must match an allowlist and never a deny pattern.
package safety

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Sandbox confines all file paths to a single root directory.
type Sandbox struct {
	root string
}

// ErrOutsideSandbox is returned when a request resolves outside the workspace.
var ErrOutsideSandbox = errors.New("path resolves outside workspace sandbox")

// NewSandbox validates that root exists and returns a Sandbox bound to it.
func NewSandbox(root string) (*Sandbox, error) {
	if root == "" {
		return nil, errors.New("workspace root must be set")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// Allow the directory to not exist yet; create it.
		if os.IsNotExist(err) {
			if mkErr := os.MkdirAll(abs, 0o755); mkErr != nil {
				return nil, fmt.Errorf("create root: %w", mkErr)
			}
			resolved = abs
		} else {
			return nil, fmt.Errorf("eval root: %w", err)
		}
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, fmt.Errorf("stat root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("workspace root %q is not a directory", resolved)
	}
	return &Sandbox{root: resolved}, nil
}

// Root returns the absolute, symlink-resolved sandbox root.
func (s *Sandbox) Root() string { return s.root }

// Resolve maps a user-supplied relative path to an absolute path inside the
// sandbox. It rejects absolute inputs, parent traversal, and symlinks that
// escape the root. The returned path may not exist yet (useful for writes).
func (s *Sandbox) Resolve(rel string) (string, error) {
	clean := strings.TrimSpace(rel)
	if clean == "" || clean == "." {
		return s.root, nil
	}
	if filepath.IsAbs(clean) {
		return "", fmt.Errorf("%w: absolute paths not allowed", ErrOutsideSandbox)
	}
	// Reject Windows-style drive letters and UNC paths even on Linux hosts.
	if len(clean) >= 2 && clean[1] == ':' {
		return "", fmt.Errorf("%w: drive-letter paths not allowed", ErrOutsideSandbox)
	}
	if strings.HasPrefix(clean, `\\`) {
		return "", fmt.Errorf("%w: UNC paths not allowed", ErrOutsideSandbox)
	}

	joined := filepath.Join(s.root, clean)
	cleaned := filepath.Clean(joined)
	if !s.contains(cleaned) {
		return "", fmt.Errorf("%w: %s", ErrOutsideSandbox, rel)
	}

	// If the path exists, follow symlinks and re-check containment.
	if resolved, err := filepath.EvalSymlinks(cleaned); err == nil {
		if !s.contains(resolved) {
			return "", fmt.Errorf("%w: symlink escapes sandbox", ErrOutsideSandbox)
		}
		return resolved, nil
	}
	return cleaned, nil
}

// ResolveExisting is like Resolve but requires the path to exist.
func (s *Sandbox) ResolveExisting(rel string) (string, error) {
	abs, err := s.Resolve(rel)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(abs); err != nil {
		return "", err
	}
	return abs, nil
}

// Rel returns the workspace-relative path for an absolute file inside the
// sandbox.
func (s *Sandbox) Rel(abs string) (string, error) {
	rel, err := filepath.Rel(s.root, abs)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(rel, "..") {
		return "", ErrOutsideSandbox
	}
	if rel == "." {
		return "", nil
	}
	return filepath.ToSlash(rel), nil
}

func (s *Sandbox) contains(abs string) bool {
	rel, err := filepath.Rel(s.root, abs)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, "..")
}
