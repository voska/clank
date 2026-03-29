package detect

import (
	"regexp"
	"strings"
)

type State string

const (
	StateRunning State = "running"
	StateWaiting State = "waiting"
	StateDone    State = "done"
	StateError   State = "error"
	StateExited  State = "exited"
	StateUnknown State = "unknown"
)

var knownAgents = map[string]string{
	"claude": "claude",
	"droid":  "droid",
	"aider":  "aider",
	"codex":  "codex",
}

var versionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

func IdentifyAgent(command string) string {
	if agent, ok := knownAgents[command]; ok {
		return agent
	}
	if versionPattern.MatchString(command) {
		return "claude"
	}
	return ""
}

func DetectState(content string) (State, string) {
	if content == "" {
		return StateUnknown, ""
	}

	lines := strings.Split(content, "\n")
	start := len(lines) - 30
	if start < 0 {
		start = 0
	}
	tail := strings.Join(lines[start:], "\n")

	// Waiting: permission prompt (highest priority — check full content)
	if strings.Contains(tail, "? Allow") || strings.Contains(tail, "Yes, allow") {
		return StateWaiting, "permission"
	}
	if strings.Contains(tail, "No, and tell Claude what to do differently") {
		return StateWaiting, "decision"
	}

	// Find the LAST ❯ prompt — everything above it is history.
	// Only check for running indicators AFTER the last ❯.
	recent := tail
	if idx := strings.LastIndex(tail, "❯"); idx >= 0 {
		recent = tail[idx:]
	}

	// Running: Unicode ellipsis followed by timing — "Thinking… (1m 32s)"
	// Only in content AFTER the last ❯ prompt
	if strings.Contains(recent, "\u2026 (") {
		return StateRunning, ""
	}

	// Running: tool execution — "Running…"
	if strings.Contains(recent, "Running\u2026") {
		return StateRunning, ""
	}

	// Running: "esc to interrupt"
	if strings.Contains(recent, "esc to interrupt") {
		return StateRunning, ""
	}

	// Running: braille spinner characters
	for _, ch := range []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"} {
		if strings.Contains(recent, ch) {
			return StateRunning, ""
		}
	}

	// Done: ❯ prompt visible with no running indicators after it
	if strings.Contains(tail, "❯") {
		return StateDone, ""
	}

	// Done: > prompt on its own line
	if strings.Contains(tail, "\n> \n") {
		return StateDone, ""
	}

	// Exited: bare shell prompt
	lastLine := lastNonEmptyLine(tail)
	trimmedLine := strings.TrimSpace(lastLine)
	if trimmedLine == "$" || trimmedLine == "%" ||
		strings.HasSuffix(trimmedLine, "$ ") || strings.HasSuffix(trimmedLine, "$") {
		return StateExited, ""
	}

	return StateUnknown, ""
}

func lastNonEmptyLine(s string) string {
	lines := strings.Split(s, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			return lines[i]
		}
	}
	return ""
}
