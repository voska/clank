package ui

import (
	"fmt"
	"strings"
	"time"

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

	var stateStr string
	if collapsed && highestState != "" {
		stateStr = renderStateIcon(highestState, 0)
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

	indicator := renderStateIcon(win.Agent.State, spinnerFrame)

	// Add timer for running/waiting states
	if win.Agent.State == detect.StateRunning || win.Agent.State == detect.StateWaiting {
		elapsed := time.Since(win.Agent.LastChanged)
		timer := formatDuration(elapsed)
		indicator = indicator + " " + dimStyle.Render(timer)
	}

	leftLen := lipgloss.Width(left)
	rightLen := lipgloss.Width(indicator)
	gap := width - leftLen - rightLen
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + indicator
}

func renderStateIcon(s detect.State, spinnerFrame int) string {
	switch s {
	case detect.StateRunning:
		glyph := spinnerFrames[spinnerFrame%len(spinnerFrames)]
		return indicatorRunning.Render(glyph)
	case detect.StateWaiting:
		return indicatorWaiting.Render(glyphWaiting)
	case detect.StateDone:
		return indicatorDone.Render(glyphDone)
	case detect.StateError:
		return indicatorError.Render(glyphError)
	case detect.StateExited:
		return dimStyle.Render("·")
	default:
		return ""
	}
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}
