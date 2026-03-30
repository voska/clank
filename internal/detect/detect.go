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

	tail := content

	// Running checks FIRST — active work is the strongest signal.
	// Old "? Allow" text can linger in visible output from previous turns,
	// but running indicators only appear during active work.

	// Running: Unicode ellipsis followed by timing — "Thinking… (1m 32s)"
	if strings.Contains(tail, "\u2026 (") {
		return StateRunning, ""
	}

	// Running: tool execution — "Running…"
	if strings.Contains(tail, "Running\u2026") {
		return StateRunning, ""
	}

	// Running: "esc to interrupt"
	if strings.Contains(tail, "esc to interrupt") {
		return StateRunning, ""
	}

	// Running: braille spinner characters
	for _, ch := range []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"} {
		if strings.Contains(tail, ch) {
			return StateRunning, ""
		}
	}

	// Waiting: permission prompt (only if NOT running).
	// Only check the last 8 lines to avoid matching stale "? Allow" text
	// from conversation history higher up in the visible output.
	bottom := lastNLines(tail, 8)
	if strings.Contains(bottom, "? Allow") || strings.Contains(bottom, "Yes, allow") {
		return StateWaiting, "permission"
	}
	if strings.Contains(bottom, "No, and tell Claude what to do differently") {
		return StateWaiting, "decision"
	}

	// Done: ❯ prompt visible with no running indicators after it
	if strings.Contains(tail, "❯") {
		return StateDone, ""
	}

	// Done: Droid idle — "? for help" in status bar or bare ">" prompt
	if strings.Contains(tail, "? for help") {
		return StateDone, ""
	}

	// Done: > prompt on its own line
	if strings.Contains(tail, "\n> \n") || strings.Contains(tail, "│ >") {
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

func lastNLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	start := len(lines) - n
	if start < 0 {
		start = 0
	}
	return strings.Join(lines[start:], "\n")
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
