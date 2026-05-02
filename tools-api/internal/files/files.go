// Package files implements the file-oriented HTTP endpoints of tools-api:
// listing, reading, writing, applying unified diffs, and substring search.
package files

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/coinbaseblock/opencodeforge/tools-api/internal/safety"
)

const (
	maxReadBytes   = 2 * 1024 * 1024  // 2 MiB cap for /read
	maxWriteBytes  = 4 * 1024 * 1024  // 4 MiB cap for /write
	maxPatchBytes  = 4 * 1024 * 1024  // 4 MiB cap for /patch
	maxSearchHits  = 500
	maxSearchBytes = 1 * 1024 * 1024 // skip files larger than this when searching
)

// Register installs the file routes onto mux.
func Register(mux *http.ServeMux, sandbox *safety.Sandbox) {
	h := &handler{sandbox: sandbox}
	mux.HandleFunc("/files", h.list)
	mux.HandleFunc("/read", h.read)
	mux.HandleFunc("/write", h.write)
	mux.HandleFunc("/patch", h.patch)
	mux.HandleFunc("/search", h.search)
}

type handler struct {
	sandbox *safety.Sandbox
}

type fileEntry struct {
	Path  string `json:"path"`
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	rel := r.URL.Query().Get("path")
	abs, err := h.sandbox.Resolve(rel)
	if err != nil {
		writeSandboxError(w, err)
		return
	}
	info, err := os.Stat(abs)
	if err != nil {
		writeFSError(w, err)
		return
	}
	if !info.IsDir() {
		writeError(w, http.StatusBadRequest, "path is not a directory")
		return
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		writeFSError(w, err)
		return
	}
	out := make([]fileEntry, 0, len(entries))
	for _, e := range entries {
		full := filepath.Join(abs, e.Name())
		relPath, relErr := h.sandbox.Rel(full)
		if relErr != nil {
			continue
		}
		fi, statErr := e.Info()
		if statErr != nil {
			continue
		}
		out = append(out, fileEntry{
			Path:  relPath,
			Name:  e.Name(),
			IsDir: fi.IsDir(),
			Size:  fi.Size(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return out[i].Name < out[j].Name
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"path":    strings.TrimPrefix(strings.TrimPrefix(abs, h.sandbox.Root()), "/"),
		"entries": out,
	})
}

func (h *handler) read(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	rel := r.URL.Query().Get("path")
	if rel == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}
	abs, err := h.sandbox.ResolveExisting(rel)
	if err != nil {
		writeSandboxError(w, err)
		return
	}
	info, err := os.Stat(abs)
	if err != nil {
		writeFSError(w, err)
		return
	}
	if info.IsDir() {
		writeError(w, http.StatusBadRequest, "path is a directory")
		return
	}
	if info.Size() > maxReadBytes {
		writeError(w, http.StatusRequestEntityTooLarge,
			fmt.Sprintf("file is %d bytes; max %d", info.Size(), maxReadBytes))
		return
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		writeFSError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"path":    rel,
		"size":    info.Size(),
		"content": string(data),
	})
}

type writeRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Mode    string `json:"mode,omitempty"` // "overwrite" (default) or "create"
}

func (h *handler) write(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req writeRequest
	if err := decodeJSON(r, &req, maxWriteBytes); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}
	abs, err := h.sandbox.Resolve(req.Path)
	if err != nil {
		writeSandboxError(w, err)
		return
	}
	if req.Mode == "create" {
		if _, err := os.Stat(abs); err == nil {
			writeError(w, http.StatusConflict, "file already exists")
			return
		}
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		writeFSError(w, err)
		return
	}
	if err := os.WriteFile(abs, []byte(req.Content), 0o644); err != nil {
		writeFSError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"path":  req.Path,
		"bytes": len(req.Content),
	})
}

type patchRequest struct {
	Diff    string `json:"diff"`
	Strip   int    `json:"strip,omitempty"` // -p value, defaults to 1
	Reverse bool   `json:"reverse,omitempty"`
	Cwd     string `json:"cwd,omitempty"` // relative directory to apply in
}

func (h *handler) patch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req patchRequest
	if err := decodeJSON(r, &req, maxPatchBytes); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.Diff) == "" {
		writeError(w, http.StatusBadRequest, "diff is required")
		return
	}
	cwd, err := h.sandbox.ResolveExisting(req.Cwd)
	if err != nil {
		writeSandboxError(w, err)
		return
	}
	strip := req.Strip
	if strip <= 0 {
		strip = 1
	}
	args := []string{fmt.Sprintf("-p%d", strip), "--batch", "--forward"}
	if req.Reverse {
		args = append(args, "-R")
	}
	cmd := exec.Command("patch", args...)
	cmd.Dir = cwd
	cmd.Stdin = strings.NewReader(req.Diff)
	out, runErr := cmd.CombinedOutput()
	status := http.StatusOK
	ok := runErr == nil
	if !ok {
		status = http.StatusUnprocessableEntity
	}
	writeJSON(w, status, map[string]any{
		"ok":     ok,
		"output": string(out),
	})
}

type searchHit struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Snippet string `json:"snippet"`
}

func (h *handler) search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, "q is required")
		return
	}
	pathParam := r.URL.Query().Get("path")
	startAbs, err := h.sandbox.Resolve(pathParam)
	if err != nil {
		writeSandboxError(w, err)
		return
	}
	hits, truncated, err := searchTree(h.sandbox, startAbs, q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"query":     q,
		"hits":      hits,
		"truncated": truncated,
	})
}

var ignoredDirs = map[string]struct{}{
	".git":         {},
	"node_modules": {},
	"vendor":       {},
	"dist":         {},
	"build":        {},
	".venv":        {},
	"__pycache__":  {},
	".next":        {},
	".cache":       {},
	"target":       {},
}

func searchTree(sb *safety.Sandbox, root, query string) ([]searchHit, bool, error) {
	var hits []searchHit
	truncated := false

	walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable paths
		}
		if d.IsDir() {
			if _, skip := ignoredDirs[d.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		fi, statErr := d.Info()
		if statErr != nil || fi.Size() > maxSearchBytes {
			return nil
		}
		f, openErr := os.Open(path)
		if openErr != nil {
			return nil
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		lineNo := 0
		for scanner.Scan() {
			lineNo++
			line := scanner.Text()
			if strings.Contains(line, query) {
				rel, relErr := sb.Rel(path)
				if relErr != nil {
					continue
				}
				hits = append(hits, searchHit{
					Path:    rel,
					Line:    lineNo,
					Snippet: trimSnippet(line),
				})
				if len(hits) >= maxSearchHits {
					truncated = true
					return io.EOF
				}
			}
		}
		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, io.EOF) {
		return nil, truncated, walkErr
	}
	return hits, truncated, nil
}

func trimSnippet(s string) string {
	const max = 240
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

// --- HTTP helpers ------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeSandboxError(w http.ResponseWriter, err error) {
	if errors.Is(err, safety.ErrOutsideSandbox) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	writeFSError(w, err)
}

func writeFSError(w http.ResponseWriter, err error) {
	switch {
	case os.IsNotExist(err):
		writeError(w, http.StatusNotFound, err.Error())
	case os.IsPermission(err):
		writeError(w, http.StatusForbidden, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func decodeJSON(r *http.Request, v any, maxBytes int64) error {
	r.Body = http.MaxBytesReader(nil, r.Body, maxBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	return nil
}
