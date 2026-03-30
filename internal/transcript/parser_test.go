package transcript

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/voska/clank/internal/detect"
)

func TestDeriveState(t *testing.T) {
	endTurn := "end_turn"
	toolUse := "tool_use"

	tests := []struct {
		name string
		line rawLine
		want detect.State
	}{
		{"user message", rawLine{Type: "user"}, detect.StateRunning},
		{"user tool result", rawLine{Type: "user"}, detect.StateRunning},
		{"assistant end_turn", rawLine{Type: "assistant", Message: &struct {
			StopReason *string `json:"stop_reason"`
		}{StopReason: &endTurn}}, detect.StateDone},
		{"assistant tool_use", rawLine{Type: "assistant", Message: &struct {
			StopReason *string `json:"stop_reason"`
		}{StopReason: &toolUse}}, detect.StateRunning},
		{"assistant streaming", rawLine{Type: "assistant", Message: &struct {
			StopReason *string `json:"stop_reason"`
		}{StopReason: nil}}, detect.StateRunning},
		{"assistant no message", rawLine{Type: "assistant"}, detect.StateRunning},
		{"system turn_duration", rawLine{Type: "system", Subtype: "turn_duration"}, detect.StateDone},
		{"system stop_hook_summary", rawLine{Type: "system", Subtype: "stop_hook_summary"}, detect.StateDone},
		{"system other", rawLine{Type: "system", Subtype: "something_else"}, detect.StateUnknown},
		{"file-history-snapshot", rawLine{Type: "file-history-snapshot"}, detect.StateUnknown},
		{"last-prompt", rawLine{Type: "last-prompt"}, detect.StateUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeriveState(tt.line)
			if got != tt.want {
				t.Errorf("DeriveState(%s) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestReadLastState(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    detect.State
	}{
		{
			name: "idle after end_turn",
			content: `{"type":"user","message":{"role":"user","content":"hello"}}
{"type":"assistant","message":{"stop_reason":"end_turn","content":[{"type":"text","text":"hi"}]}}
{"type":"system","subtype":"turn_duration","durationMs":1000}
`,
			want: detect.StateDone,
		},
		{
			name: "running mid-tool",
			content: `{"type":"user","message":{"role":"user","content":"fix it"}}
{"type":"assistant","message":{"stop_reason":"tool_use","content":[{"type":"tool_use"}]}}
{"type":"user","message":{"content":[{"type":"tool_result"}]}}
`,
			want: detect.StateRunning,
		},
		{
			name: "running after user submit",
			content: `{"type":"assistant","message":{"stop_reason":"end_turn"}}
{"type":"system","subtype":"turn_duration"}
{"type":"user","message":{"role":"user","content":"do something"}}
`,
			want: detect.StateRunning,
		},
		{
			name: "noise lines ignored",
			content: `{"type":"assistant","message":{"stop_reason":"end_turn"}}
{"type":"file-history-snapshot","snapshot":{}}
{"type":"last-prompt","lastPrompt":"hello"}
`,
			want: detect.StateDone,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "test.jsonl")
			os.WriteFile(path, []byte(tt.content), 0644)

			got, err := ReadLastState(path)
			if err != nil {
				t.Fatalf("ReadLastState: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
