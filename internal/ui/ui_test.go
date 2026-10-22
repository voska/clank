package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/voska/clank/internal/detect"
	"github.com/voska/clank/internal/state"
	"github.com/voska/clank/internal/tmux"
)

// mockMux implements tmux.Mux for testing.
type mockMux struct {
	sessions []tmux.Session
	windows  map[string][]tmux.Window
	panes    []tmux.PaneInfo
	captures map[string]string
	switched []string
}

func (m *mockMux) ListSessions() ([]tmux.Session, error) { return m.sessions, nil }
func (m *mockMux) ListWindows(session string) ([]tmux.Window, error) {
	return m.windows[session], nil
}
func (m *mockMux) AllPanes() ([]tmux.PaneInfo, error) { return m.panes, nil }
func (m *mockMux) PaneCommand(_ string) (string, error) {
	return "", nil
}

func (m *mockMux) PaneCommands(_ string, _ int) ([]string, error) {
	return nil, nil
}

func (m *mockMux) CapturePane(target string) (string, error) {
	if m.captures != nil {
		if c, ok := m.captures[target]; ok {
			return c, nil
		}
	}
	return "", fmt.Errorf("no capture for %s", target)
}

func (m *mockMux) CapturePaneByIndex(session string, windowIndex, paneIndex int) (string, error) {
	return m.CapturePane(fmt.Sprintf("%s:%d.%d", session, windowIndex, paneIndex))
}

func (m *mockMux) SwitchWindow(session string, windowIndex int) error {
	m.switched = append(m.switched, fmt.Sprintf("%s:%d", session, windowIndex))
	return nil
}

func TestRenderSessionHeader(t *testing.T) {
	got := renderSessionHeader(1, "proj-a", false, "", false, 30)
	if !strings.Contains(got, "proj-a") {
		t.Errorf("expected name in header, got %q", got)
	}
}

func TestRenderWindowLine(t *testing.T) {
	win := state.WindowState{
		Name:   "dev",
		Index:  1,
		Active: false,
		Agent: &state.AgentInfo{
			Name:  "claude",
			State: detect.StateRunning,
		},
	}
	got := renderWindowLine(win, 30, false, 0)
	if !strings.Contains(got, "dev") {
		t.Errorf("expected window name, got %q", got)
	}
}

func TestRenderWindowLineActive(t *testing.T) {
	win := state.WindowState{
		Name:   "dev",
		Index:  1,
		Active: true,
	}
	got := renderWindowLine(win, 30, false, 0)
	if !strings.Contains(got, "dev") {
		t.Errorf("expected window name, got %q", got)
	}
}

func TestRenderWindowLineNoAgent(t *testing.T) {
	win := state.WindowState{
		Name:   "misc",
		Index:  2,
		Active: false,
	}
	got := renderWindowLine(win, 30, false, 0)
	if strings.Contains(got, glyphRunning) || strings.Contains(got, glyphDone) {
		t.Error("expected no agent indicator for plain window")
	}
}

func TestIdentifyAgentPanes(t *testing.T) {
	panes := []tmux.PaneInfo{
		{Session: "s1", Window: "dev", WinIndex: 1, PaneIndex: 0, Command: "claude", PanePID: 100},
		{Session: "s1", Window: "dev", WinIndex: 1, PaneIndex: 1, Command: "zsh", PanePID: 101},
		{Session: "s1", Window: "web", WinIndex: 2, PaneIndex: 0, Command: "droid", PanePID: 102},
		{Session: "s2", Window: "misc", WinIndex: 1, PaneIndex: 0, Command: "bash", PanePID: 103},
	}
	agents, claudePIDs := identifyAgentPanes(panes)
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}
	if agents[0].name != "claude" || agents[1].name != "droid" {
		t.Errorf("unexpected agents: %+v", agents)
	}
	if len(claudePIDs) != 1 || claudePIDs[0] != 100 {
		t.Errorf("unexpected claude PIDs: %v", claudePIDs)
	}
}

func TestIdentifyAgentPanesDedup(t *testing.T) {
	panes := []tmux.PaneInfo{
		{Session: "s1", Window: "dev", WinIndex: 1, PaneIndex: 0, Command: "claude", PanePID: 100},
		{Session: "s1", Window: "dev", WinIndex: 1, PaneIndex: 1, Command: "claude", PanePID: 101},
	}
	agents, _ := identifyAgentPanes(panes)
	if len(agents) != 1 {
		t.Errorf("expected dedup to 1 agent, got %d", len(agents))
	}
}

func TestSessionFilter(t *testing.T) {
	store := state.New()
	store.UpdateTree([]state.SessionSnapshot{
		{Name: "proj-a", Index: 1, Windows: []state.WindowSnapshot{
			{Name: "dev", Index: 1, Active: true},
		}},
		{Name: "proj-b", Index: 2, Windows: []state.WindowSnapshot{
			{Name: "misc", Index: 1, Active: true},
		}},
		{Name: "proj-c", Index: 3, Windows: []state.WindowSnapshot{
			{Name: "api", Index: 1, Active: false},
		}},
		{Name: "idle", Index: 4, Windows: []state.WindowSnapshot{
			{Name: "shell", Index: 1, Active: true},
		}},
	})
	// proj-a: running -> active
	store.SetAgentState("proj-a", 1, state.AgentInfo{Name: "claude", State: detect.StateRunning})
	// proj-b: done + seen -> NOT active
	store.SetAgentState("proj-b", 1, state.AgentInfo{Name: "claude", State: detect.StateDone})
	store.ClearUnseen("proj-b", 1)
	// proj-c: done + unseen -> active (needs attention)
	store.SetAgentState("proj-c", 1, state.AgentInfo{Name: "droid", State: detect.StateDone})
	// idle: no agent -> NOT active

	m := NewModel(store, &mockMux{}, nil, Config{Width: 30}, nil)

	// Without filter: all 4 sessions visible
	m.filterActive = false
	m.rebuildItems()
	sessionCount := 0
	for _, it := range m.items {
		if it.isSession {
			sessionCount++
		}
	}
	if sessionCount != 4 {
		t.Errorf("unfiltered: expected 4 sessions, got %d", sessionCount)
	}

	// With filter: only proj-a (running) + proj-c (done+unseen)
	m.filterActive = true
	m.rebuildItems()
	sessionCount = 0
	var sessionNames []string
	for _, it := range m.items {
		if it.isSession {
			sessionCount++
			sessionNames = append(sessionNames, it.session)
		}
	}
	if sessionCount != 2 {
		t.Errorf("filtered: expected 2 sessions (proj-a, proj-c), got %d (%v)", sessionCount, sessionNames)
	}
}

func TestSessionFilterEmpty(t *testing.T) {
	store := state.New()
	store.UpdateTree([]state.SessionSnapshot{
		{Name: "shell", Index: 1, Windows: []state.WindowSnapshot{
			{Name: "zsh", Index: 1, Active: true},
		}},
	})

	m := NewModel(store, &mockMux{}, nil, Config{Width: 30}, nil)
	m.filterActive = true
	m.rebuildItems()

	if len(m.items) != 0 {
		t.Errorf("expected 0 items when all sessions filtered, got %d", len(m.items))
	}
}

func TestDetectAgentStateTitle(t *testing.T) {
	store := state.New()
	store.UpdateTree([]state.SessionSnapshot{
		{Name: "s", Index: 1, Windows: []state.WindowSnapshot{
			{Name: "w", Index: 1, Active: true},
		}},
	})

	mux := &mockMux{}
	m := NewModel(store, mux, nil, Config{Width: 30}, nil)

	// Running title -> running, no scrape needed
	s, _ := m.detectAgentState(agentPane{
		name: "claude", session: "s", winIndex: 1, title: "\u2802 fix-bug",
	})
	if s != detect.StateRunning {
		t.Errorf("expected running from busy title, got %s", s)
	}

	// Idle title + waiting content -> waiting
	mux.captures = map[string]string{
		"s:1.0": "? Allow Read tool on file.go?\n  Yes, allow once",
	}
	s, reason := m.detectAgentState(agentPane{
		name: "claude", session: "s", winIndex: 1, paneIndex: 0, title: "\u2733 Claude Code",
	})
	if s != detect.StateWaiting || reason != "permission" {
		t.Errorf("expected waiting/permission, got %s/%s", s, reason)
	}

	// Idle title + no waiting content -> done
	mux.captures = map[string]string{
		"s:1.0": "All done.\n\u276f \nstatus bar",
	}
	s, _ = m.detectAgentState(agentPane{
		name: "claude", session: "s", winIndex: 1, paneIndex: 0, title: "\u2733 Claude Code",
	})
	if s != detect.StateDone {
		t.Errorf("expected done from idle title, got %s", s)
	}
}

func TestDetectAgentStateFallback(t *testing.T) {
	store := state.New()
	store.UpdateTree([]state.SessionSnapshot{
		{Name: "s", Index: 1, Windows: []state.WindowSnapshot{
			{Name: "w", Index: 1, Active: true},
		}},
	})

	mux := &mockMux{
		captures: map[string]string{
			"s:1.0": "Executing...  (Press ESC to stop)\n\u2502 > deploy \u2502",
		},
	}
	m := NewModel(store, mux, nil, Config{Width: 30}, nil)

	// No title (droid) -> full scrape
	s, _ := m.detectAgentState(agentPane{
		name: "droid", session: "s", winIndex: 1, paneIndex: 0, title: "",
	})
	if s != detect.StateRunning {
		t.Errorf("expected running from droid scrape, got %s", s)
	}
}

func TestApplyScrapeResult(t *testing.T) {
	store := state.New()
	store.UpdateTree([]state.SessionSnapshot{
		{Name: "s", Index: 1, Windows: []state.WindowSnapshot{
			{Name: "a", Index: 1, Active: false},
			{Name: "b", Index: 2, Active: false},
		}},
	})

	m := NewModel(store, &mockMux{}, nil, Config{Width: 30}, nil)

	msg := scrapeResultMsg{
		updates: []scrapedAgent{
			{session: "s", winIndex: 1, info: state.AgentInfo{Name: "claude", State: detect.StateRunning}},
			{session: "s", winIndex: 2, info: state.AgentInfo{Name: "droid", State: detect.StateDone}},
		},
	}
	m.applyScrapeResult(msg)

	a1 := store.GetAgent("s", 1)
	if a1 == nil || a1.State != detect.StateRunning {
		t.Errorf("window 1: expected running, got %+v", a1)
	}
	a2 := store.GetAgent("s", 2)
	if a2 == nil || a2.State != detect.StateDone {
		t.Errorf("window 2: expected done, got %+v", a2)
	}
}

func TestApplyScrapeResultSkipsFreshHook(t *testing.T) {
	store := state.New()
	store.UpdateTree([]state.SessionSnapshot{
		{Name: "s", Index: 1, Windows: []state.WindowSnapshot{
			{Name: "a", Index: 1, Active: false},
		}},
	})

	// Set agent state from hook (recent)
	store.SetAgentState("s", 1, state.AgentInfo{
		Name:     "claude",
		State:    detect.StateWaiting,
		FromHook: true,
	})

	m := NewModel(store, &mockMux{}, nil, Config{Width: 30}, nil)

	// Scrape says "done" but hook data is fresh — should be ignored
	msg := scrapeResultMsg{
		updates: []scrapedAgent{
			{session: "s", winIndex: 1, info: state.AgentInfo{Name: "claude", State: detect.StateDone}},
		},
	}
	m.applyScrapeResult(msg)

	agent := store.GetAgent("s", 1)
	if agent.State != detect.StateWaiting {
		t.Errorf("expected hook state preserved, got %s", agent.State)
	}
}

func TestApplyPollResult(t *testing.T) {
	store := state.New()
	m := NewModel(store, &mockMux{}, nil, Config{Width: 30}, nil)

	msg := pollResultMsg{
		snapshots: []state.SessionSnapshot{
			{Name: "s1", Index: 1, Windows: []state.WindowSnapshot{
				{Name: "dev", Index: 1, Active: true},
				{Name: "test", Index: 2, Active: false},
			}},
		},
	}
	m.applyPollResult(msg)

	sessions := store.Sessions()
	if len(sessions) != 1 || len(sessions[0].Windows) != 2 {
		t.Errorf("unexpected tree: %+v", sessions)
	}
	if !m.pruned {
		t.Error("expected pruned=true after first poll")
	}
}
