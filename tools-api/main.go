// Command tools-api is the OpenCodeForge sandboxed HTTP API.
//
// It exposes file, git, and shell operations scoped to a single workspace
// directory and a configurable command allowlist.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/coinbaseblock/opencodeforge/tools-api/internal/files"
	"github.com/coinbaseblock/opencodeforge/tools-api/internal/git"
	"github.com/coinbaseblock/opencodeforge/tools-api/internal/safety"
	"github.com/coinbaseblock/opencodeforge/tools-api/internal/shell"
)

func main() {
	cfg := loadConfig()

	sandbox, err := safety.NewSandbox(cfg.WorkspaceDir)
	if err != nil {
		log.Fatalf("init sandbox: %v", err)
	}

	guard := safety.NewCommandGuard(cfg.AllowedCommands, cfg.SafeMode)

	mux := http.NewServeMux()
	files.Register(mux, sandbox)
	git.Register(mux, sandbox)
	shell.Register(mux, sandbox, guard)

	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"tools-api"}`))
	})

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           withLogging(mux),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("tools-api listening on %s (workspace=%s, safe_mode=%t)",
			cfg.ListenAddr, sandbox.Root(), cfg.SafeMode)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Println("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

type config struct {
	WorkspaceDir    string
	ListenAddr      string
	SafeMode        bool
	AllowedCommands []string
}

func loadConfig() config {
	cfg := config{
		WorkspaceDir: getenv("WORKSPACE_DIR", "/workspace"),
		ListenAddr:   getenv("LISTEN_ADDR", ":8088"),
		SafeMode:     getenv("SAFE_MODE", "true") != "false",
	}
	raw := getenv("ALLOWED_COMMANDS",
		"go test ./...,go build ./...,npm test,npm run build,python -m pytest,docker compose config")
	for _, c := range strings.Split(raw, ",") {
		if c = strings.TrimSpace(c); c != "" {
			cfg.AllowedCommands = append(cfg.AllowedCommands, c)
		}
	}
	return cfg
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, rw.status, time.Since(start))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}
