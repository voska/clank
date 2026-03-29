package tmux

import "testing"

func TestParseSessionList(t *testing.T) {
	raw := "1:sensai:4:0\n2:homelab:2:1\n3:onspot:3:0\n"
	sessions := ParseSessionList(raw)
	if len(sessions) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(sessions))
	}
	want := []Session{
		{Index: 1, Name: "sensai", WindowCount: 4, Attached: false},
		{Index: 2, Name: "homelab", WindowCount: 2, Attached: true},
		{Index: 3, Name: "onspot", WindowCount: 3, Attached: false},
	}
	for i, s := range sessions {
		if s != want[i] {
			t.Errorf("session %d: got %+v, want %+v", i, s, want[i])
		}
	}
}

func TestParseWindowList(t *testing.T) {
	raw := "1:dev:1\n2:droid:0\n3:web:0\n"
	windows := ParseWindowList(raw)
	if len(windows) != 3 {
		t.Fatalf("expected 3 windows, got %d", len(windows))
	}
	want := []Window{
		{Index: 1, Name: "dev", Active: true},
		{Index: 2, Name: "droid", Active: false},
		{Index: 3, Name: "web", Active: false},
	}
	for i, w := range windows {
		if w != want[i] {
			t.Errorf("window %d: got %+v, want %+v", i, w, want[i])
		}
	}
}

func TestParsePaneCommand(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"claude\n", "claude"},
		{"zsh\n", "zsh"},
		{"bash\n", "bash"},
		{"droid\n", "droid"},
		{"", ""},
	}
	for _, tt := range tests {
		got := ParsePaneCommand(tt.raw)
		if got != tt.want {
			t.Errorf("ParsePaneCommand(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}
