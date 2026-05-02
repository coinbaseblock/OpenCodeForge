// Package shell exposes a guarded shell-execution endpoint.
package shell

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os/exec"
	"time"

	"github.com/coinbaseblock/opencodeforge/tools-api/internal/safety"
)

const (
	defaultTimeout = 5 * time.Minute
	maxTimeout     = 15 * time.Minute
)

// Register installs the /run route on mux.
func Register(mux *http.ServeMux, sandbox *safety.Sandbox, guard *safety.CommandGuard) {
	h := &handler{sandbox: sandbox, guard: guard}
	mux.HandleFunc("/run", h.run)
	mux.HandleFunc("/run/policy", h.policy)
}

type handler struct {
	sandbox *safety.Sandbox
	guard   *safety.CommandGuard
}

type runRequest struct {
	Cmd       string `json:"cmd"`
	Cwd       string `json:"cwd,omitempty"`
	TimeoutMs int    `json:"timeout_ms,omitempty"`
}

type runResponse struct {
	Ok       bool   `json:"ok"`
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	TimedOut bool   `json:"timed_out,omitempty"`
}

func (h *handler) run(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req runRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Cmd == "" {
		writeError(w, http.StatusBadRequest, "cmd is required")
		return
	}
	if err := h.guard.Check(req.Cmd); err != nil {
		switch {
		case errors.Is(err, safety.ErrCommandDenied):
			writeError(w, http.StatusForbidden, "command denied by safety policy")
		case errors.Is(err, safety.ErrCommandNotAllowed):
			writeError(w, http.StatusForbidden, "command not in allowlist")
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	cwd, err := h.sandbox.ResolveExisting(req.Cwd)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	timeout := defaultTimeout
	if req.TimeoutMs > 0 {
		t := time.Duration(req.TimeoutMs) * time.Millisecond
		if t > maxTimeout {
			t = maxTimeout
		}
		timeout = t
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", req.Cmd)
	cmd.Dir = cwd
	stdout, stderr, exitCode, runErr := capture(cmd)

	resp := runResponse{
		Ok:       runErr == nil,
		ExitCode: exitCode,
		Stdout:   stdout,
		Stderr:   stderr,
		TimedOut: errors.Is(ctx.Err(), context.DeadlineExceeded),
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *handler) policy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"safe_mode":         h.guard.SafeMode(),
		"allowed_prefixes":  h.guard.AllowedPrefixes(),
		"default_timeout_s": int(defaultTimeout.Seconds()),
		"max_timeout_s":     int(maxTimeout.Seconds()),
	})
}

func capture(cmd *exec.Cmd) (string, string, int, error) {
	var stdout, stderr capBuf
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return stdout.String(), stderr.String(), exitCode, err
}

// capBuf is a fixed-size byte buffer that drops overflow.
type capBuf struct {
	buf []byte
}

const maxCapture = 256 * 1024

func (b *capBuf) Write(p []byte) (int, error) {
	remaining := maxCapture - len(b.buf)
	if remaining <= 0 {
		return len(p), nil
	}
	if len(p) > remaining {
		b.buf = append(b.buf, p[:remaining]...)
		b.buf = append(b.buf, []byte("\n[truncated]\n")...)
		return len(p), nil
	}
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *capBuf) String() string { return string(b.buf) }

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
