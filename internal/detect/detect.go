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

// versionPattern matches version strings like "2.1.87" that Claude Code
// reports via pane_current_command (tmux resolves the symlink to the version dir).
var versionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// IdentifyAgent returns the agent name from a process/command name.
// Handles both direct names ("claude", "droid") and Claude Code's
// version-as-command-name quirk ("2.1.87").
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
	start := len(lines) - 20
	if start < 0 {
		start = 0
	}
	tail := strings.Join(lines[start:], "\n")

	if strings.Contains(tail, "esc to interrupt") {
		return StateRunning, ""
	}
	for _, ch := range []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"} {
		if strings.Contains(tail, ch) {
			return StateRunning, ""
		}
	}

	if strings.Contains(tail, "? Allow") || strings.Contains(tail, "Yes, allow") {
		return StateWaiting, "permission"
	}

	if strings.Contains(tail, "No, and tell Claude what to do differently") {
		return StateWaiting, "decision"
	}

	// Done/idle: Claude Code's prompt is ">" or "❯" at the end
	lastLine := lastNonEmptyLine(tail)
	trimmedLine := strings.TrimSpace(lastLine)
	if trimmedLine == ">" || trimmedLine == "❯" {
		return StateDone, ""
	}

	// Exited: bare shell prompt (only "$" or "%" without ">" or "❯")
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
