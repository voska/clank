package tmux

import "testing"

func TestParseSessionList(t *testing.T) {
	raw := "proj-a\t4\t0\nproj-b\t2\t1\nproj-c\t3\t0\n"
	sessions := ParseSessionList(raw)
	if len(sessions) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(sessions))
	}
	want := []Session{
		{Index: 1, Name: "proj-a", WindowCount: 4, Attached: false},
		{Index: 2, Name: "proj-b", WindowCount: 2, Attached: true},
		{Index: 3, Name: "proj-c", WindowCount: 3, Attached: false},
	}
	for i, s := range sessions {
		if s != want[i] {
			t.Errorf("session %d: got %+v, want %+v", i, s, want[i])
		}
	}
}

func TestParseWindowList(t *testing.T) {
	raw := "1\tdev\t1\n2\tdroid\t0\n3\tweb\t0\n"
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
