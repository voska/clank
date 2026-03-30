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
		{"", ""},
		{"2.1.87", "claude"},
		{"1.0.0", "claude"},
		{"10.22.345", "claude"},
		{"2.1", ""},
		{"v2.1.87", ""},
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
			name:    "running: thinking with timer after prompt",
			content: "old output\n───\n❯ what should I do?\n───\n✢ Thinking\u2026 (1m 32s · ↓ 790 tokens)\n  ⎿  Tip: ...\n",
			want:    StateRunning,
		},
		{
			name:    "running: random verb with timer after prompt",
			content: "old output\n───\n❯ fix the bug\n───\n✢ Sautéeing\u2026 (5s)\n",
			want:    StateRunning,
		},
		{
			name:    "running: tool execution after prompt",
			content: "old output\n───\n❯ do it\n───\n⎿  Running\u2026\n",
			want:    StateRunning,
		},
		{
			name:    "running: esc to interrupt",
			content: "some output\n  esc to interrupt",
			want:    StateRunning,
		},
		{
			name:    "running beats stale permission prompt",
			content: "old output\n? Allow Read tool on file.go?\n  Yes, allow once\nmore output\n✢ Thinking\u2026 (5s)\n",
			want:    StateRunning,
		},
		{
			name:    "done: past tense completed",
			content: "✻ Sautéed for 33s\n\n───\n❯ \n───\n  proj | Opus 4.6\n  ⏸ plan mode\n",
			want:    StateDone,
		},
		{
			name:    "done: cogitated past tense",
			content: "✻ Cogitated for 33s\n\n───\n❯ \n───\n  proj | Opus 4.6\n",
			want:    StateDone,
		},
		{
			name:    "done: bare chevron prompt",
			content: "some output\n❯ \nstatus bar",
			want:    StateDone,
		},
		{
			name:    "waiting: permission",
			content: "some output\n? Allow Read tool on file.go?\n  Yes, allow once",
			want:    StateWaiting,
			reason:  "permission",
		},
		{
			name:    "waiting: user decision",
			content: "Do you want?\n  No, and tell Claude what to do differently",
			want:    StateWaiting,
			reason:  "decision",
		},
		{
			name:    "done not waiting: stale permission text high up in output",
			content: "? Allow Read tool on file.go?\n  Yes, allow once\n\nDid the thing.\n\n✻ Cogitated for 33s\n\n───\n❯ \n───\n  proj | Opus 4.6\n  ⏵⏵ bypass\n",
			want:    StateDone,
		},
		{
			name:    "exited: shell prompt",
			content: "some output\n$ ",
			want:    StateExited,
		},
		{
			name:    "unknown: empty",
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
