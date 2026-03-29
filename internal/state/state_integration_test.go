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
		{Name: "homelab", Index: 1, Windows: []WindowSnapshot{
			{Name: "hass", Index: 1, Active: false},
			{Name: "opensessions", Index: 2, Active: true},
		}},
		{Name: "onspot", Index: 2, Windows: []WindowSnapshot{
			{Name: "cf", Index: 1, Active: false},
		}},
	})

	// Scrape cycle 1: detect agents
	s.SetAgentState("homelab", "hass", AgentInfo{Name: "claude", State: detect.StateDone})
	s.SetAgentState("homelab", "opensessions", AgentInfo{Name: "claude", State: detect.StateRunning})
	s.SetAgentState("onspot", "cf", AgentInfo{Name: "droid", State: detect.StateDone})

	// Verify initial unseen state
	sessions := s.Sessions()
	if !sessions[0].Windows[0].HasUnseen { // hass (not active)
		t.Fatal("hass: expected unseen after initial done")
	}
	if sessions[0].Windows[1].HasUnseen { // opensessions (active window)
		t.Fatal("opensessions: should NOT be unseen (it's the active window)")
	}
	if !sessions[1].Windows[0].HasUnseen { // cf (not active)
		t.Fatal("cf: expected unseen after initial done")
	}

	// User clicks hass in sidebar -> clear unseen
	s.ClearUnseen("homelab", "hass")
	sessions = s.Sessions()
	if sessions[0].Windows[0].HasUnseen {
		t.Fatal("hass: unseen should be cleared after click")
	}

	// Scrape cycle 2: same state (done -> done)
	s.SetAgentState("homelab", "hass", AgentInfo{Name: "claude", State: detect.StateDone})

	sessions = s.Sessions()
	if sessions[0].Windows[0].HasUnseen {
		t.Fatal("hass: unseen should NOT reappear on repeated done scrape")
	}

	// Scrape cycle 3, 4, 5: keep scraping same state
	for i := 0; i < 3; i++ {
		s.SetAgentState("homelab", "hass", AgentInfo{Name: "claude", State: detect.StateDone})
	}
	sessions = s.Sessions()
	if sessions[0].Windows[0].HasUnseen {
		t.Fatal("hass: unseen still reappeared after multiple scrape cycles")
	}

	// Now agent ACTUALLY changes state (user sends prompt -> running -> done)
	s.SetAgentState("homelab", "hass", AgentInfo{Name: "claude", State: detect.StateRunning})
	sessions = s.Sessions()
	if !sessions[0].Windows[0].HasUnseen {
		t.Fatal("hass: expected unseen after running transition")
	}

	s.ClearUnseen("homelab", "hass")

	s.SetAgentState("homelab", "hass", AgentInfo{Name: "claude", State: detect.StateDone})
	sessions = s.Sessions()
	if !sessions[0].Windows[0].HasUnseen {
		t.Fatal("hass: expected unseen after real running->done transition")
	}

	// User clicks again -> clear
	s.ClearUnseen("homelab", "hass")

	// Multiple scrapes -> stays cleared
	for i := 0; i < 5; i++ {
		s.SetAgentState("homelab", "hass", AgentInfo{Name: "claude", State: detect.StateDone})
	}
	sessions = s.Sessions()
	if sessions[0].Windows[0].HasUnseen {
		t.Fatal("hass: unseen reappeared after clear + repeated done scrapes")
	}
}

// TestUnseenClearsOnActiveWindow verifies that when pollTmux reports
// a window as active, unseen gets cleared automatically.
func TestUnseenClearsOnActiveWindow(t *testing.T) {
	s := New()

	s.UpdateTree([]SessionSnapshot{
		{Name: "homelab", Index: 1, Windows: []WindowSnapshot{
			{Name: "hass", Index: 1, Active: false},
		}},
	})

	// Agent finishes
	s.SetAgentState("homelab", "hass", AgentInfo{Name: "claude", State: detect.StateDone})
	sessions := s.Sessions()
	if !sessions[0].Windows[0].HasUnseen {
		t.Fatal("expected unseen")
	}

	// User switches to that window (tmux reports it active)
	s.UpdateTree([]SessionSnapshot{
		{Name: "homelab", Index: 1, Windows: []WindowSnapshot{
			{Name: "hass", Index: 1, Active: true},
		}},
	})
	// Simulate what pollTmux does: clear unseen on active windows
	s.ClearUnseen("homelab", "hass")

	sessions = s.Sessions()
	if sessions[0].Windows[0].HasUnseen {
		t.Fatal("unseen should clear when window becomes active")
	}

	// Scrape runs again
	s.SetAgentState("homelab", "hass", AgentInfo{Name: "claude", State: detect.StateDone})
	sessions = s.Sessions()
	if sessions[0].Windows[0].HasUnseen {
		t.Fatal("unseen should NOT reappear after clear + repeated done")
	}
}
