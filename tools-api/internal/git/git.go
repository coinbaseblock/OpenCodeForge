// Package git exposes read-only git inspection endpoints for the workspace.
package git

import (
	"encoding/json"
	"errors"
	"net/http"
	"os/exec"

	"github.com/coinbaseblock/opencodeforge/tools-api/internal/safety"
)

// Register installs the git routes onto mux.
func Register(mux *http.ServeMux, sandbox *safety.Sandbox) {
	h := &handler{sandbox: sandbox}
	mux.HandleFunc("/git/status", h.status)
	mux.HandleFunc("/git/diff", h.diff)
}

type handler struct {
	sandbox *safety.Sandbox
}

func (h *handler) status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	cwd, err := h.resolveCwd(r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	out, runErr := runGit(cwd, "status", "--porcelain=v1", "-b")
	writeGitResult(w, out, runErr)
}

func (h *handler) diff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	cwd, err := h.resolveCwd(r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	args := []string{"diff", "--no-color"}
	if r.URL.Query().Get("staged") == "true" {
		args = append(args, "--staged")
	}
	if file := r.URL.Query().Get("file"); file != "" {
		// Validate file is inside the sandbox too.
		if _, err := h.sandbox.Resolve(file); err != nil {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
		args = append(args, "--", file)
	}
	out, runErr := runGit(cwd, args...)
	writeGitResult(w, out, runErr)
}

func (h *handler) resolveCwd(rel string) (string, error) {
	cwd, err := h.sandbox.ResolveExisting(rel)
	if err != nil {
		return "", err
	}
	return cwd, nil
}

func runGit(cwd string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func writeGitResult(w http.ResponseWriter, out string, runErr error) {
	status := http.StatusOK
	ok := runErr == nil
	if !ok {
		status = http.StatusUnprocessableEntity
	}
	writeJSON(w, status, map[string]any{
		"ok":     ok,
		"output": out,
	})
}

func statusFor(err error) int {
	if errors.Is(err, safety.ErrOutsideSandbox) {
		return http.StatusForbidden
	}
	return http.StatusBadRequest
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
