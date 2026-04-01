package transcript

import (
	"encoding/json"
	"io"
	"os"
	"strings"


	"github.com/voska/clank/internal/detect"
)

// rawLine is the minimal JSON structure we need from a JSONL line.
type rawLine struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
	Message *struct {
		StopReason *string `json:"stop_reason"`
		Content    json.RawMessage `json:"content"`
	} `json:"message"`
	// For system error messages
	Level string `json:"level"`
}

// stateRelevant returns true if this line type indicates agent state.
func (r rawLine) stateRelevant() bool {
	switch r.Type {
	case "user", "assistant", "system":
		return true
	}
	return false
}

// DeriveState determines the agent state from a parsed JSONL line.
func DeriveState(line rawLine) detect.State {
	switch line.Type {
	case "user":
		return detect.StateRunning

	case "assistant":
		if line.Message == nil || line.Message.StopReason == nil {
			return detect.StateRunning // streaming, incomplete
		}
		switch *line.Message.StopReason {
		case "end_turn":
			return detect.StateDone
		case "tool_use":
			return detect.StateRunning
		default:
			return detect.StateRunning
		}

	case "system":
		switch line.Subtype {
		case "turn_duration", "stop_hook_summary":
			return detect.StateDone
		}
		// Check for rate limit / API error system messages
		if line.Level == "error" {
			content := string(line.Message.Content)
			if strings.Contains(content, "rate") || strings.Contains(content, "overloaded") ||
				strings.Contains(content, "529") || strings.Contains(content, "limit") {
				return detect.StateError
			}
		}
	}

	return detect.StateUnknown
}

// ReadLastState reads the tail of a JSONL file and returns the state
// derived from the last state-relevant line.
func ReadLastState(path string) (detect.State, error) {
	f, err := os.Open(path)
	if err != nil {
		return detect.StateUnknown, err
	}
	defer f.Close()

	// Seek to last 16KB — enough for several JSONL lines
	const tailSize = 16384
	stat, err := f.Stat()
	if err != nil {
		return detect.StateUnknown, err
	}
	offset := stat.Size() - tailSize
	if offset < 0 {
		offset = 0
	}
	f.Seek(offset, io.SeekStart)

	data, err := io.ReadAll(f)
	if err != nil {
		return detect.StateUnknown, err
	}

	lines := strings.Split(string(data), "\n")

	// Walk backward to find the last state-relevant line
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		var rl rawLine
		if err := json.Unmarshal([]byte(line), &rl); err != nil {
			continue
		}

		if !rl.stateRelevant() {
			continue
		}

		state := DeriveState(rl)
		if state != detect.StateUnknown {
			return state, nil
		}
	}

	return detect.StateUnknown, nil
}
