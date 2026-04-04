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
	if selected {
		return cursorStyle.Render(name)
	}
	return sessionStyle.Render(name)
}

func renderWindowLine(win state.WindowState, width int, selected bool, spinnerFrame int) string {
	indent := "  "

	name := win.Name
	// Truncate long names to leave room for indicators
	maxNameLen := width - 12 // indent(2) + gap(1) + timer(4) + space(1) + icon(1) + margin(3)
	if maxNameLen < 8 {
		maxNameLen = 8
	}
	if len(name) > maxNameLen {
		name = name[:maxNameLen-1] + "…"
	}

	if selected {
		name = cursorStyle.Render(name)
	} else if win.Active {
		name = windowActiveStyle.Render(name)
	} else {
		name = dimStyle.Render(name)
	}

	left := indent + name

	if win.Agent == nil {
		return left
	}

	// Done + already seen = no indicator (idle, nothing to show)
	if win.Agent.State == detect.StateDone && !win.HasUnseen {
		return left
	}

	icon := renderStateIcon(win.Agent.State, spinnerFrame)

	var right string
	if win.Agent.State == detect.StateRunning || win.Agent.State == detect.StateWaiting {
		elapsed := time.Since(win.Agent.LastChanged)
		timer := formatDuration(elapsed)
		right = dimStyle.Render(timer) + " " + icon
	} else {
		right = icon
	}

	leftLen := lipgloss.Width(left)
	rightLen := lipgloss.Width(right)
	gap := width - leftLen - rightLen
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + right
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
