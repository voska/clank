package state

import (
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
	s.SetAgentState("proj-b", "backend", AgentInfo{Name: "claude", State: detect.StateDone})
	s.SetAgentState("proj-b", "editor", AgentInfo{Name: "claude", State: detect.StateRunning})
	s.SetAgentState("proj-c", "cf", AgentInfo{Name: "droid", State: detect.StateDone})

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
	s.ClearUnseen("proj-b", "backend")
	sessions = s.Sessions()
	if sessions[0].Windows[0].HasUnseen {
		t.Fatal("backend: unseen should be cleared after click")
	}

	// Scrape cycle 2: same state (done -> done)
	s.SetAgentState("proj-b", "backend", AgentInfo{Name: "claude", State: detect.StateDone})

	sessions = s.Sessions()
	if sessions[0].Windows[0].HasUnseen {
		t.Fatal("backend: unseen should NOT reappear on repeated done scrape")
	}

	// Scrape cycle 3, 4, 5: keep scraping same state
	for i := 0; i < 3; i++ {
		s.SetAgentState("proj-b", "backend", AgentInfo{Name: "claude", State: detect.StateDone})
	}
	sessions = s.Sessions()
	if sessions[0].Windows[0].HasUnseen {
		t.Fatal("backend: unseen still reappeared after multiple scrape cycles")
	}

	// Now agent ACTUALLY changes state (user sends prompt -> running -> done)
	s.SetAgentState("proj-b", "backend", AgentInfo{Name: "claude", State: detect.StateRunning})
	sessions = s.Sessions()
	if !sessions[0].Windows[0].HasUnseen {
		t.Fatal("backend: expected unseen after running transition")
	}

	s.ClearUnseen("proj-b", "backend")

	s.SetAgentState("proj-b", "backend", AgentInfo{Name: "claude", State: detect.StateDone})
	sessions = s.Sessions()
	if !sessions[0].Windows[0].HasUnseen {
		t.Fatal("backend: expected unseen after real running->done transition")
	}

	// User clicks again -> clear
	s.ClearUnseen("proj-b", "backend")

	// Multiple scrapes -> stays cleared
	for i := 0; i < 5; i++ {
		s.SetAgentState("proj-b", "backend", AgentInfo{Name: "claude", State: detect.StateDone})
	}
	sessions = s.Sessions()
	if sessions[0].Windows[0].HasUnseen {
		t.Fatal("backend: unseen reappeared after clear + repeated done scrapes")
	}
}

// TestUnseenClearsOnActiveWindow verifies that when pollTmux reports
// a window as active, unseen gets cleared automatically.
func TestUnseenClearsOnActiveWindow(t *testing.T) {
	s := New()

	s.UpdateTree([]SessionSnapshot{
		{Name: "proj-b", Index: 1, Windows: []WindowSnapshot{
			{Name: "backend", Index: 1, Active: false},
		}},
	})

	// Agent finishes
	s.SetAgentState("proj-b", "backend", AgentInfo{Name: "claude", State: detect.StateDone})
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
	s.ClearUnseen("proj-b", "backend")

	sessions = s.Sessions()
	if sessions[0].Windows[0].HasUnseen {
		t.Fatal("unseen should clear when window becomes active")
	}

	// Scrape runs again
	s.SetAgentState("proj-b", "backend", AgentInfo{Name: "claude", State: detect.StateDone})
	sessions = s.Sessions()
	if sessions[0].Windows[0].HasUnseen {
		t.Fatal("unseen should NOT reappear after clear + repeated done")
	}
}
