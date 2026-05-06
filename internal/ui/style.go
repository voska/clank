package ui

import "github.com/charmbracelet/lipgloss"

// Catppuccin palette — Latte for light terminals, Mocha for dark.
var (
	colorRosewater = lipgloss.AdaptiveColor{Light: "#dc8a78", Dark: "#f5e0dc"}
	colorBlue      = lipgloss.AdaptiveColor{Light: "#1e66f5", Dark: "#89b4fa"}
	colorGreen     = lipgloss.AdaptiveColor{Light: "#40a02b", Dark: "#a6e3a1"}
	colorYellow    = lipgloss.AdaptiveColor{Light: "#df8e1d", Dark: "#f9e2af"}
	colorRed       = lipgloss.AdaptiveColor{Light: "#d20f39", Dark: "#f38ba8"}
	colorText      = lipgloss.AdaptiveColor{Light: "#4c4f69", Dark: "#cdd6f4"}
	colorSubtext   = lipgloss.AdaptiveColor{Light: "#6c6f85", Dark: "#a6adc8"}
	colorOverlay   = lipgloss.AdaptiveColor{Light: "#9ca0b0", Dark: "#6c7086"}
	colorSurface   = lipgloss.AdaptiveColor{Light: "#ccd0da", Dark: "#313244"}
	colorBase      = lipgloss.AdaptiveColor{Light: "#eff1f5", Dark: "#1e1e2e"}
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
