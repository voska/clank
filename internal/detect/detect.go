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

// Claude Code title prefixes (from source: REPL.tsx)
// Busy: animated braille frames ⠂ / ⠐ (alternating at 960ms)
// Idle/waiting: static ✳
const (
	titlePrefixIdle  = "✳"
	titlePrefixBusy1 = "⠂"
	titlePrefixBusy2 = "⠐"
)

func IdentifyAgent(command string) string {
	if agent, ok := knownAgents[command]; ok {
		return agent
	}
	if versionPattern.MatchString(command) {
		return "claude"
	}
	return ""
}

// DetectStateFromTitle uses the Claude Code pane title prefix to determine
// busy vs not-busy. Returns StateRunning for animated prefixes, StateDone
// for the static prefix, or StateUnknown if the title doesn't match.
func DetectStateFromTitle(title string) State {
	if title == "" {
		return StateUnknown
	}
	// Title format: "{prefix} {session name}" e.g. "✳ Claude Code" or "⠂ fix-auth-bug"
	if strings.HasPrefix(title, titlePrefixBusy1) || strings.HasPrefix(title, titlePrefixBusy2) {
		return StateRunning
	}
	if strings.HasPrefix(title, titlePrefixIdle) {
		return StateDone // could be idle or waiting — caller should check content for waiting
	}
	return StateUnknown
}

// DetectWaiting checks pane content (bottom lines only) for permission/decision
// prompts. Use this as a secondary check when the title indicates not-busy.
func DetectWaiting(content string) (bool, string) {
	if content == "" {
		return false, ""
	}
	bottom := lastNLines(content, 8)

	if strings.Contains(bottom, "? Allow") || strings.Contains(bottom, "Yes, allow") {
		return true, "permission"
	}
	if strings.Contains(bottom, "No, and tell Claude what to do differently") {
		return true, "decision"
	}
	return false, ""
}

// DetectState detects Claude Code state from pane content. This is the
// fallback path used when title-based detection isn't available (e.g. for
// droid or other agents that don't set OSC titles).
func DetectState(content string) (State, string) {
	if content == "" {
		return StateUnknown, ""
	}

	tail := content
	bottom := lastNLines(tail, 8)

	// Rate limit / API error — highest priority, shows ✗
	if strings.Contains(tail, "rate limited") ||
		strings.Contains(tail, "usage limit reached") ||
		strings.Contains(tail, "Rate limit") ||
		strings.Contains(tail, "Resets at") ||
		strings.Contains(tail, "overloaded_error") ||
		strings.Contains(tail, "529") && strings.Contains(tail, "overloaded") ||
		strings.Contains(tail, "api_error") {
		return StateError, "rate_limit"
	}

	// Droid active execution — always takes priority.
	// The mission prompt (│ >) is visible even during execution, so we must
	// check for explicit running indicators before idle patterns.
	if strings.Contains(tail, "Executing...") && strings.Contains(tail, "Press ESC to stop") {
		return StateRunning, ""
	}
	if strings.Contains(tail, "sub-agents running") {
		return StateRunning, ""
	}

	// Droid idle: only when NOT executing
	if strings.Contains(bottom, "? for help") ||
		strings.Contains(bottom, "↑/↓ Navigate") ||
		strings.Contains(bottom, "Enter Select") {
		return StateDone, ""
	}

	// Droid idle: empty mission prompt box (│ > │ or │ >  │)
	if strings.Contains(bottom, "│ >") && strings.Contains(bottom, "│") {
		trimmed := ""
		for _, line := range strings.Split(bottom, "\n") {
			if strings.Contains(line, "│ >") {
				trimmed = strings.TrimSpace(line)
				break
			}
		}
		// Only match if the prompt content is empty or very short (user hasn't typed a long command)
		if strings.Contains(trimmed, "│ >") && !strings.Contains(trimmed, "Invoking") {
			return StateDone, ""
		}
	}

	// Claude waiting: permission prompt (bottom of screen only)
	if strings.Contains(bottom, "? Allow") || strings.Contains(bottom, "Yes, allow") {
		return StateWaiting, "permission"
	}
	if strings.Contains(bottom, "No, and tell Claude what to do differently") {
		return StateWaiting, "decision"
	}

	// Running checks — active work indicators anywhere in visible content.

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

	// Running: braille spinner characters (but only if no idle indicator at bottom)
	for _, ch := range []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"} {
		if strings.Contains(tail, ch) {
			return StateRunning, ""
		}
	}

	// Done: ❯ prompt visible (Claude idle)
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
