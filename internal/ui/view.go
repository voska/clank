package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/voska/clank/internal/detect"
	"github.com/voska/clank/internal/state"
)

func renderSessionHeader(index int, name string, collapsed bool, highestState detect.State, selected bool) string {
	toggle := glyphExpanded
	style := sessionStyle
	if collapsed {
		toggle = glyphCollapsed
		style = sessionCollapsedStyle
	}

	prefix := fmt.Sprintf("%2d", index)
	if selected {
		prefix = cursorStyle.Render(prefix)
	}

	header := fmt.Sprintf("%s %s", prefix, style.Render(name))

	var stateStr string
	if collapsed && highestState != "" {
		stateStr = renderStateIndicator(highestState, 0)
	}

	if stateStr != "" {
		header = fmt.Sprintf("%s  %s", header, stateStr)
	}

	header = fmt.Sprintf("%s %s", header, dimStyle.Render(toggle))
	return header
}

func renderWindowLine(win state.WindowState, width int, selected bool, spinnerFrame int) string {
	var prefix string
	if win.Active {
		prefix = cursorStyle.Render(glyphActive + " ")
	} else {
		prefix = "    "
	}

	name := win.Name
	if selected {
		name = cursorStyle.Render(name)
	} else if win.Active {
		name = windowActiveStyle.Render(name)
	} else {
		name = windowStyle.Render(name)
	}

	var indicator string
	if win.Agent != nil {
		indicator = renderStateIndicator(win.Agent.State, spinnerFrame)
	}

	var unseenMark string
	if win.HasUnseen {
		unseenMark = indicatorUnseen.Render(glyphUnseen) + " "
	}

	line := fmt.Sprintf("%s%s%s", prefix, unseenMark, name)
	if indicator != "" {
		nameLen := lipgloss.Width(line)
		indicatorLen := lipgloss.Width(indicator)
		padding := width - nameLen - indicatorLen - 1
		if padding < 1 {
			padding = 1
		}
		line = line + strings.Repeat(" ", padding) + indicator
	}
	return line
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
