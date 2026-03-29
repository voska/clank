package state

import (
	"testing"

	"github.com/voska/clank/internal/detect"
)

func TestUpdateAndQuery(t *testing.T) {
	s := New()
	s.UpdateTree([]SessionSnapshot{
		{Name: "proj-a", Index: 1, Windows: []WindowSnapshot{
			{Name: "dev", Index: 1, Active: true},
			{Name: "droid", Index: 2, Active: false},
		}},
	})
	sessions := s.Sessions()
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if len(sessions[0].Windows) != 2 {
		t.Fatalf("expected 2 windows, got %d", len(sessions[0].Windows))
	}
}

func TestSetAgentState(t *testing.T) {
	s := New()
	s.UpdateTree([]SessionSnapshot{
		{Name: "proj-a", Index: 1, Windows: []WindowSnapshot{
			{Name: "dev", Index: 1, Active: true},
		}},
	})
	s.SetAgentState("proj-a", "dev", AgentInfo{
		Name:  "claude",
		State: detect.StateRunning,
	})
	sessions := s.Sessions()
	agent := sessions[0].Windows[0].Agent
	if agent == nil {
		t.Fatal("expected agent, got nil")
	}
	if agent.State != detect.StateRunning {
		t.Errorf("expected running, got %s", agent.State)
	}
}

func TestUnseenMarker(t *testing.T) {
	s := New()
	s.UpdateTree([]SessionSnapshot{
		{Name: "proj-a", Index: 1, Windows: []WindowSnapshot{
			{Name: "dev", Index: 1, Active: false},
		}},
	})
	s.SetAgentState("proj-a", "dev", AgentInfo{
		Name:  "claude",
		State: detect.StateDone,
	})
	sessions := s.Sessions()
	if !sessions[0].Windows[0].HasUnseen {
		t.Error("expected unseen marker on non-active window")
	}
	s.ClearUnseen("proj-a", "dev")
	sessions = s.Sessions()
	if sessions[0].Windows[0].HasUnseen {
		t.Error("expected unseen marker to be cleared")
	}
}

func TestCollapseToggle(t *testing.T) {
	s := New()
	s.UpdateTree([]SessionSnapshot{
		{Name: "proj-a", Index: 1, Windows: []WindowSnapshot{
			{Name: "dev", Index: 1, Active: true},
		}},
	})
	s.ToggleCollapse("proj-a")
	sessions := s.Sessions()
	if !sessions[0].Collapsed {
		t.Error("expected collapsed after toggle")
	}
	s.ToggleCollapse("proj-a")
	sessions = s.Sessions()
	if sessions[0].Collapsed {
		t.Error("expected expanded after second toggle")
	}
}

func TestUnseenStaysCleared(t *testing.T) {
	s := New()
	s.UpdateTree([]SessionSnapshot{
		{Name: "proj-a", Index: 1, Windows: []WindowSnapshot{
			{Name: "dev", Index: 1, Active: false},
		}},
	})

	// Agent finishes -> unseen = true
	s.SetAgentState("proj-a", "dev", AgentInfo{Name: "claude", State: detect.StateDone})
	sessions := s.Sessions()
	if !sessions[0].Windows[0].HasUnseen {
		t.Fatal("expected unseen after state transition to done")
	}

	// User clicks -> clear unseen
	s.ClearUnseen("proj-a", "dev")
	sessions = s.Sessions()
	if sessions[0].Windows[0].HasUnseen {
		t.Fatal("expected unseen cleared after click")
	}

	// Scrape runs again with same state (done -> done) -> unseen should NOT reappear
	s.SetAgentState("proj-a", "dev", AgentInfo{Name: "claude", State: detect.StateDone})
	sessions = s.Sessions()
	if sessions[0].Windows[0].HasUnseen {
		t.Fatal("unseen should NOT reappear on repeated done state")
	}

	// Actual state change (done -> running) -> unseen should appear
	s.SetAgentState("proj-a", "dev", AgentInfo{Name: "claude", State: detect.StateRunning})
	s.SetAgentState("proj-a", "dev", AgentInfo{Name: "claude", State: detect.StateDone})
	sessions = s.Sessions()
	if !sessions[0].Windows[0].HasUnseen {
		t.Fatal("expected unseen after real state transition running->done")
	}
}

func TestHighestPriorityState(t *testing.T) {
	s := New()
	s.UpdateTree([]SessionSnapshot{
		{Name: "proj-a", Index: 1, Windows: []WindowSnapshot{
			{Name: "dev", Index: 1, Active: false},
			{Name: "droid", Index: 2, Active: false},
		}},
	})
	s.SetAgentState("proj-a", "dev", AgentInfo{Name: "claude", State: detect.StateDone})
	s.SetAgentState("proj-a", "droid", AgentInfo{Name: "droid", State: detect.StateWaiting})
	hp := s.HighestPriority("proj-a")
	if hp != detect.StateWaiting {
		t.Errorf("expected waiting (highest priority), got %s", hp)
	}
}
