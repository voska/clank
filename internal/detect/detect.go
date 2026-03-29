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

// Claude Code active indicators — present tense with ellipsis means working
var runningPatterns = []string{
	"Running…",
	"Thinking…",
	"Clauding…",
	"Pondering…",
	"Befuddling…",
	"Reasoning…",
	"Planning…",
	"Analyzing…",
	"Searching…",
	"Reading…",
	"Writing…",
	"Editing…",
	"Executing…",
	"Installing…",
	"Compiling…",
	"Building…",
	"Testing…",
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

	// Waiting: permission prompt (check first — highest priority)
	if strings.Contains(tail, "? Allow") || strings.Contains(tail, "Yes, allow") {
		return StateWaiting, "permission"
	}
	if strings.Contains(tail, "No, and tell Claude what to do differently") {
		return StateWaiting, "decision"
	}

	// Running: "esc to interrupt" (original pattern, may work in some terminals)
	if strings.Contains(tail, "esc to interrupt") {
		return StateRunning, ""
	}

	// Running: active spinner text (present tense with ellipsis)
	for _, pattern := range runningPatterns {
		if strings.Contains(tail, pattern) {
			return StateRunning, ""
		}
	}

	// Running: braille spinner characters
	for _, ch := range []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"} {
		if strings.Contains(tail, ch) {
			return StateRunning, ""
		}
	}

	// Running: tool execution indicator (⎿ prefix with Running)
	if strings.Contains(tail, "⎿") && strings.Contains(tail, "Running") {
		return StateRunning, ""
	}

	// Done: Claude's idle prompt (❯ or >) visible with no running indicators
	lastLine := lastNonEmptyLine(tail)
	trimmedLine := strings.TrimSpace(lastLine)
	// The status bar is always the last non-empty content, so check lines above it
	if strings.Contains(tail, "❯") || strings.Contains(tail, "\n> \n") {
		return StateDone, ""
	}

	// Exited: bare shell prompt
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
