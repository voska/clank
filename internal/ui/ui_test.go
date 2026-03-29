package ui

import (
	"strings"
	"testing"

	"github.com/voska/clank/internal/detect"
	"github.com/voska/clank/internal/state"
)

func TestRenderSessionHeader(t *testing.T) {
	got := renderSessionHeader(1, "proj-a", false, "", false)
	if !strings.Contains(got, "1") || !strings.Contains(got, "proj-a") {
		t.Errorf("expected index and name in header, got %q", got)
	}
	if !strings.Contains(got, glyphExpanded) {
		t.Errorf("expected expanded glyph, got %q", got)
	}
}

func TestRenderSessionHeaderCollapsed(t *testing.T) {
	got := renderSessionHeader(1, "proj-a", true, "", false)
	if !strings.Contains(got, glyphCollapsed) {
		t.Errorf("expected collapsed glyph, got %q", got)
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
	if !strings.Contains(got, glyphActive) {
		t.Errorf("expected active glyph, got %q", got)
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
