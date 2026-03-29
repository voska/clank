package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/voska/clank/internal/detect"
	"github.com/voska/clank/internal/state"
)

func renderSessionHeader(index int, name string, collapsed bool, highestState detect.State, selected bool, width int) string {
	toggle := glyphExpanded
	if collapsed {
		toggle = glyphCollapsed
	}

	prefix := fmt.Sprintf("%d", index)
	styledName := sessionStyle.Render(name)
	if collapsed {
		styledName = sessionCollapsedStyle.Render(name)
	}
	if selected {
		prefix = cursorStyle.Render(prefix)
		styledName = cursorStyle.Render(name)
	}

	left := fmt.Sprintf(" %s %s", prefix, styledName)

	// If collapsed and there's agent activity, show rolled-up state
	var stateStr string
	if collapsed && highestState != "" {
		stateStr = renderStateIndicator(highestState, 0)
	}

	right := dimStyle.Render(toggle)
	if stateStr != "" {
		right = stateStr + " " + right
	}

	leftLen := lipgloss.Width(left)
	rightLen := lipgloss.Width(right)
	gap := width - leftLen - rightLen
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + right
}

func renderWindowLine(win state.WindowState, width int, selected bool, spinnerFrame int) string {
	indent := "    "

	var unseenMark string
	if win.HasUnseen {
		unseenMark = indicatorUnseen.Render(glyphUnseen) + " "
	}

	name := win.Name
	if selected {
		name = cursorStyle.Render(name)
	} else if win.Active {
		name = windowActiveStyle.Render(name)
	} else {
		name = dimStyle.Render(name)
	}

	left := indent + unseenMark + name

	if win.Agent == nil {
		return left
	}

	indicator := renderStateIndicator(win.Agent.State, spinnerFrame)
	leftLen := lipgloss.Width(left)
	rightLen := lipgloss.Width(indicator)
	gap := width - leftLen - rightLen
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + indicator
}

func renderStateIndicator(s detect.State, spinnerFrame int) string {
	switch s {
	case detect.StateRunning:
		glyph := spinnerFrames[spinnerFrame%len(spinnerFrames)]
		return indicatorRunning.Render(glyph + " running")
	case detect.StateWaiting:
		return indicatorWaiting.Render(glyphWaiting + " waiting")
	case detect.StateDone:
		return indicatorDone.Render(glyphDone + " done")
	case detect.StateError:
		return indicatorError.Render(glyphError + " error")
	case detect.StateExited:
		return dimStyle.Render("exited")
	default:
		return ""
	}
}
