package state

import (
	"path/filepath"
	"testing"

	"github.com/voska/clank/internal/detect"
)

// TestFullScrapeUnseenCycle simulates what happens in the real app:
// 1. Tree gets populated
// 2. Agent detected as done (scrape calls SetAgentState)
// 3. User clicks window (ClearUnseen)
// 4. Next scrape cycle calls SetAgentState(done) again
// 5. Unseen should NOT reappear
func TestFullScrapeUnseenCycle(t *testing.T) {
	s := New()

	// Simulate pollTmux populating the tree
	s.UpdateTree([]SessionSnapshot{
		{Name: "proj-b", Index: 1, Windows: []WindowSnapshot{
			{Name: "backend", Index: 1, Active: false},
			{Name: "editor", Index: 2, Active: true},
		}},
		{Name: "proj-c", Index: 2, Windows: []WindowSnapshot{
			{Name: "cf", Index: 1, Active: false},
		}},
	})

	// Scrape cycle 1: detect agents
	s.SetAgentState("proj-b", 1, AgentInfo{Name: "claude", State: detect.StateDone})
	s.SetAgentState("proj-b", 2, AgentInfo{Name: "claude", State: detect.StateRunning})
	s.SetAgentState("proj-c", 1, AgentInfo{Name: "droid", State: detect.StateDone})

	// Verify initial unseen state
	sessions := s.Sessions()
	if !sessions[0].Windows[0].HasUnseen { // backend (not active)
		t.Fatal("backend: expected unseen after initial done")
	}
	if sessions[0].Windows[1].HasUnseen { // editor (active window)
		t.Fatal("editor: should NOT be unseen (it's the active window)")
	}
	if !sessions[1].Windows[0].HasUnseen { // cf (not active)
		t.Fatal("cf: expected unseen after initial done")
	}

	// User clicks backend in sidebar -> clear unseen
	s.ClearUnseen("proj-b", 1)
	sessions = s.Sessions()
	if sessions[0].Windows[0].HasUnseen {
		t.Fatal("backend: unseen should be cleared after click")
	}

	// Scrape cycle 2: same state (done -> done)
	s.SetAgentState("proj-b", 1, AgentInfo{Name: "claude", State: detect.StateDone})

	sessions = s.Sessions()
	if sessions[0].Windows[0].HasUnseen {
		t.Fatal("backend: unseen should NOT reappear on repeated done scrape")
	}

	// Scrape cycle 3, 4, 5: keep scraping same state
	for i := 0; i < 3; i++ {
		s.SetAgentState("proj-b", 1, AgentInfo{Name: "claude", State: detect.StateDone})
	}
	sessions = s.Sessions()
	if sessions[0].Windows[0].HasUnseen {
		t.Fatal("backend: unseen still reappeared after multiple scrape cycles")
	}

	// Now agent ACTUALLY changes state (user sends prompt -> running -> done)
	s.SetAgentState("proj-b", 1, AgentInfo{Name: "claude", State: detect.StateRunning})
	sessions = s.Sessions()
	if !sessions[0].Windows[0].HasUnseen {
		t.Fatal("backend: expected unseen after running transition")
	}

	s.ClearUnseen("proj-b", 1)

	s.SetAgentState("proj-b", 1, AgentInfo{Name: "claude", State: detect.StateDone})
	sessions = s.Sessions()
	if !sessions[0].Windows[0].HasUnseen {
		t.Fatal("backend: expected unseen after real running->done transition")
	}

	// User clicks again -> clear
	s.ClearUnseen("proj-b", 1)

	// Multiple scrapes -> stays cleared
	for i := 0; i < 5; i++ {
		s.SetAgentState("proj-b", 1, AgentInfo{Name: "claude", State: detect.StateDone})
	}
	sessions = s.Sessions()
	if sessions[0].Windows[0].HasUnseen {
		t.Fatal("backend: unseen reappeared after clear + repeated done scrapes")
	}
}

// TestUnseenClearsOnActiveWindow verifies that when pollTmux reports
// a window as active, unseen gets cleared automatically.
func TestStatePersistence(t *testing.T) {
	s := New()
	s.UpdateTree([]SessionSnapshot{
		{Name: "proj-b", Index: 1, Windows: []WindowSnapshot{
			{Name: "dev", Index: 1, Active: false},
		}},
	})

	// Set agent state with a specific time
	s.SetAgentState("proj-b", 1, AgentInfo{
		Name:  "claude",
		State: detect.StateRunning,
	})

	// Save to temp file
	path := filepath.Join(t.TempDir(), "state.json")
	if err := s.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Create a new store, load from file
	s2 := New()
	if err := s2.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Populate tree — should restore agent from persisted state
	s2.UpdateTree([]SessionSnapshot{
		{Name: "proj-b", Index: 1, Windows: []WindowSnapshot{
			{Name: "dev", Index: 1, Active: false},
		}},
	})

	sessions := s2.Sessions()
	agent := sessions[0].Windows[0].Agent
	if agent == nil {
		t.Fatal("expected restored agent, got nil")
	}
	if agent.State != detect.StateRunning {
		t.Errorf("expected running, got %s", agent.State)
	}
	if agent.Name != "claude" {
		t.Errorf("expected claude, got %s", agent.Name)
	}
	// Timer should be preserved (not reset to now)
	if agent.LastChanged.IsZero() {
		t.Error("expected non-zero LastChanged after restore")
	}
}

func TestUnseenClearsOnActiveWindow(t *testing.T) {
	s := New()

	s.UpdateTree([]SessionSnapshot{
		{Name: "proj-b", Index: 1, Windows: []WindowSnapshot{
			{Name: "backend", Index: 1, Active: false},
		}},
	})

	// Agent finishes
	s.SetAgentState("proj-b", 1, AgentInfo{Name: "claude", State: detect.StateDone})
	sessions := s.Sessions()
	if !sessions[0].Windows[0].HasUnseen {
		t.Fatal("expected unseen")
	}

	// User switches to that window (tmux reports it active)
	s.UpdateTree([]SessionSnapshot{
		{Name: "proj-b", Index: 1, Windows: []WindowSnapshot{
			{Name: "backend", Index: 1, Active: true},
		}},
	})
	// Simulate what pollTmux does: clear unseen on active windows
	s.ClearUnseen("proj-b", 1)

	sessions = s.Sessions()
	if sessions[0].Windows[0].HasUnseen {
		t.Fatal("unseen should clear when window becomes active")
	}

	// Scrape runs again
	s.SetAgentState("proj-b", 1, AgentInfo{Name: "claude", State: detect.StateDone})
	sessions = s.Sessions()
	if sessions[0].Windows[0].HasUnseen {
		t.Fatal("unseen should NOT reappear after clear + repeated done")
	}
}
