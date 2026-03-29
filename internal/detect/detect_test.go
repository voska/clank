package detect

import "testing"

func TestIdentifyAgent(t *testing.T) {
	tests := []struct {
		command string
		want    string
	}{
		{"claude", "claude"},
		{"droid", "droid"},
		{"aider", "aider"},
		{"codex", "codex"},
		{"zsh", ""},
		{"bash", ""},
		{"vim", ""},
		{"node", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := IdentifyAgent(tt.command)
		if got != tt.want {
			t.Errorf("IdentifyAgent(%q) = %q, want %q", tt.command, got, tt.want)
		}
	}
}

func TestDetectState(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    State
		reason  string
	}{
		{
			name:    "running with spinner text",
			content: "some output\n⠙ Thinking...\n\x1b[2m  esc to interrupt\x1b[0m",
			want:    StateRunning,
		},
		{
			name:    "running with esc to interrupt",
			content: "some output\nDoing stuff\n  esc to interrupt",
			want:    StateRunning,
		},
		{
			name:    "waiting for permission",
			content: "some output\n? Allow Read tool on file.go?\n  Yes, allow once",
			want:    StateWaiting,
			reason:  "permission",
		},
		{
			name:    "waiting for user decision",
			content: "some output\nDo you want to proceed?\n  No, and tell Claude what to do differently",
			want:    StateWaiting,
			reason:  "decision",
		},
		{
			name:    "idle prompt",
			content: "some output\nDone with the task.\n\n> ",
			want:    StateDone,
		},
		{
			name:    "shell prompt zsh",
			content: "some output\n❯ ",
			want:    StateExited,
		},
		{
			name:    "shell prompt bash",
			content: "some output\n$ ",
			want:    StateExited,
		},
		{
			name:    "empty content",
			content: "",
			want:    StateUnknown,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reason := DetectState(tt.content)
			if got != tt.want {
				t.Errorf("state: got %q, want %q", got, tt.want)
			}
			if tt.reason != "" && reason != tt.reason {
				t.Errorf("reason: got %q, want %q", reason, tt.reason)
			}
		})
	}
}
