package detect

import "strings"

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

func IdentifyAgent(command string) string {
	return knownAgents[command]
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

	trimmed := strings.TrimRight(tail, " \t\n")
	if strings.HasSuffix(trimmed, ">") || strings.HasSuffix(trimmed, "> ") {
		lastLine := lastNonEmptyLine(tail)
		if strings.TrimSpace(lastLine) == ">" {
			return StateDone, ""
		}
	}

	lastLine := lastNonEmptyLine(tail)
	trimmedLine := strings.TrimSpace(lastLine)
	if trimmedLine == "$" || trimmedLine == "%" || trimmedLine == "❯" ||
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
