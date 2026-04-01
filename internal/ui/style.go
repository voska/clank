package ui

import "github.com/charmbracelet/lipgloss"

// Catppuccin Mocha palette
var (
	colorRosewater = lipgloss.Color("#f5e0dc")
	colorBlue      = lipgloss.Color("#89b4fa")
	colorGreen     = lipgloss.Color("#a6e3a1")
	colorYellow    = lipgloss.Color("#f9e2af")
	colorRed       = lipgloss.Color("#f38ba8")
	colorText      = lipgloss.Color("#cdd6f4")
	colorSubtext   = lipgloss.Color("#a6adc8")
	colorOverlay   = lipgloss.Color("#6c7086")
	colorSurface   = lipgloss.Color("#313244")
	colorBase      = lipgloss.Color("#1e1e2e")
)

var (
	sessionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorText)

	sessionCollapsedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorSubtext)

	windowStyle = lipgloss.NewStyle().
			Foreground(colorSubtext)

	windowActiveStyle = lipgloss.NewStyle().
				Foreground(colorText)

	cursorStyle = lipgloss.NewStyle().
			Foreground(colorRosewater)

	indicatorRunning = lipgloss.NewStyle().Foreground(colorBlue)
	indicatorWaiting = lipgloss.NewStyle().Foreground(colorYellow)
	indicatorDone    = lipgloss.NewStyle().Foreground(colorGreen)
	indicatorError   = lipgloss.NewStyle().Foreground(colorRed)
	indicatorUnseen  = lipgloss.NewStyle().Foreground(colorYellow)

	dimStyle = lipgloss.NewStyle().Foreground(colorOverlay)
)

const (
	glyphRunning   = "●"
	glyphWaiting   = "⏸"
	glyphDone      = "✓"
	glyphError     = "✗"
	glyphExpanded  = "▾"
	glyphCollapsed = "▸"
	glyphActive    = "▸"
	glyphUnseen    = "·"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
