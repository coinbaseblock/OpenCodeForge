package safety

import (
	"errors"
	"regexp"
	"strings"
)

// CommandGuard validates shell command strings against a deny list and an
// optional allowlist of prefixes.
type CommandGuard struct {
	allowedPrefixes []string
	safeMode        bool
}

// ErrCommandDenied is returned when a command matches a hard deny pattern.
var ErrCommandDenied = errors.New("command denied by safety policy")

// ErrCommandNotAllowed is returned when SafeMode is on and the command does
// not match any allowlist prefix.
var ErrCommandNotAllowed = errors.New("command not in allowlist")

// NewCommandGuard creates a guard. allowedPrefixes are matched as literal
// prefixes against the normalized command line.
func NewCommandGuard(allowedPrefixes []string, safeMode bool) *CommandGuard {
	cleaned := make([]string, 0, len(allowedPrefixes))
	for _, p := range allowedPrefixes {
		if p = strings.TrimSpace(p); p != "" {
			cleaned = append(cleaned, normalizeSpaces(p))
		}
	}
	return &CommandGuard{allowedPrefixes: cleaned, safeMode: safeMode}
}

// denyPatterns match obviously destructive or evasive commands. They are
// enforced regardless of SAFE_MODE.
var denyPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\brm\s+(-[a-z]*\s+)?(/|~)`),
	regexp.MustCompile(`(?i)\bmkfs\b`),
	regexp.MustCompile(`(?i)\bdd\s+if=.*of=/dev/`),
	regexp.MustCompile(`(?i)\bformat\b`),
	regexp.MustCompile(`(?i)\bdel\s+/[sfq]\b`),
	regexp.MustCompile(`(?i)\bshutdown\b`),
	regexp.MustCompile(`(?i)\breboot\b`),
	regexp.MustCompile(`(?i)\bhalt\b`),
	regexp.MustCompile(`(?i)\bpoweroff\b`),
	regexp.MustCompile(`(?i)\bpowershell\b.*-encodedcommand\b`),
	regexp.MustCompile(`(?i)\bcurl\b[^|]*\|\s*(sh|bash|zsh)\b`),
	regexp.MustCompile(`(?i)\bwget\b[^|]*\|\s*(sh|bash|zsh)\b`),
	regexp.MustCompile(`(?i)\b(iwr|irm|invoke-webrequest|invoke-restmethod)\b[^|]*\|\s*iex\b`),
	regexp.MustCompile(`(?i):\(\)\s*\{\s*:\|:&\s*\}\s*;:`), // fork bomb
	regexp.MustCompile(`(?i)/etc/passwd`),
	regexp.MustCompile(`(?i)/etc/shadow`),
}

// Check validates cmd. It always enforces the deny list; the allowlist is
// only enforced when SAFE_MODE is on.
func (g *CommandGuard) Check(cmd string) error {
	norm := normalizeSpaces(cmd)
	if norm == "" {
		return errors.New("empty command")
	}
	for _, re := range denyPatterns {
		if re.MatchString(norm) {
			return ErrCommandDenied
		}
	}
	if !g.safeMode {
		return nil
	}
	for _, prefix := range g.allowedPrefixes {
		if matchesPrefix(norm, prefix) {
			return nil
		}
	}
	return ErrCommandNotAllowed
}

// AllowedPrefixes returns a copy of the configured allowlist for diagnostics.
func (g *CommandGuard) AllowedPrefixes() []string {
	out := make([]string, len(g.allowedPrefixes))
	copy(out, g.allowedPrefixes)
	return out
}

// SafeMode reports whether the allowlist is enforced.
func (g *CommandGuard) SafeMode() bool { return g.safeMode }

// matchesPrefix returns true when cmd starts with prefix on whitespace
// boundaries (so "go test" does not match "gotest-runner").
func matchesPrefix(cmd, prefix string) bool {
	if !strings.HasPrefix(cmd, prefix) {
		return false
	}
	if len(cmd) == len(prefix) {
		return true
	}
	next := cmd[len(prefix)]
	return next == ' ' || next == '\t'
}

var spaceRegexp = regexp.MustCompile(`\s+`)

func normalizeSpaces(s string) string {
	return strings.TrimSpace(spaceRegexp.ReplaceAllString(s, " "))
}
