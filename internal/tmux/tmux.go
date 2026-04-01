package tmux

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type Session struct {
	Index       int
	Name        string
	WindowCount int
	Attached    bool
}

type Window struct {
	Index  int
	Name   string
	Active bool
}

// PaneInfo holds the result of a bulk pane query.
type PaneInfo struct {
	Session   string
	Window    string
	WinIndex  int
	PaneIndex int
	Command   string
	PanePID   int
	Title     string
}

type Mux interface {
	ListSessions() ([]Session, error)
	ListWindows(session string) ([]Window, error)
	AllPanes() ([]PaneInfo, error)
	PaneCommand(target string) (string, error)
	PaneCommands(session string, windowIndex int) ([]string, error)
	CapturePane(target string, lines int) (string, error)
	CapturePaneByIndex(session string, windowIndex, paneIndex, lines int) (string, error)
	SwitchWindow(session string, windowIndex int) error
}

type Live struct{}

func (l *Live) ListSessions() ([]Session, error) {
	out, err := run("list-sessions", "-F", "#{session_name}\t#{session_windows}\t#{session_attached}")
	if err != nil {
		return nil, fmt.Errorf("list-sessions: %w", err)
	}
	return ParseSessionList(out), nil
}

func (l *Live) ListWindows(session string) ([]Window, error) {
	out, err := run("list-windows", "-t", session, "-F", "#{window_index}\t#{window_name}\t#{window_active}")
	if err != nil {
		return nil, fmt.Errorf("list-windows %s: %w", session, err)
	}
	return ParseWindowList(out), nil
}

// AllPanes returns every pane across all sessions in a single tmux call.
// Uses tab separator to handle colons in session/window names.
func (l *Live) AllPanes() ([]PaneInfo, error) {
	out, err := run("list-panes", "-a", "-F", "#{session_name}\t#{window_name}\t#{window_index}\t#{pane_index}\t#{pane_current_command}\t#{pane_pid}\t#{pane_title}")
	if err != nil {
		return nil, fmt.Errorf("list-panes -a: %w", err)
	}
	var panes []PaneInfo
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 7)
		if len(parts) < 6 {
			continue
		}
		winIdx, _ := strconv.Atoi(parts[2])
		paneIdx, _ := strconv.Atoi(parts[3])
		panePID, _ := strconv.Atoi(parts[5])
		var title string
		if len(parts) >= 7 {
			title = parts[6]
		}
		panes = append(panes, PaneInfo{
			Session:   parts[0],
			Window:    parts[1],
			WinIndex:  winIdx,
			PaneIndex: paneIdx,
			Command:   parts[4],
			PanePID:   panePID,
			Title:     title,
		})
	}
	return panes, nil
}

func (l *Live) PaneCommand(target string) (string, error) {
	out, err := run("display-message", "-t", target, "-p", "#{pane_current_command}")
	if err != nil {
		return "", fmt.Errorf("pane-command %s: %w", target, err)
	}
	return ParsePaneCommand(out), nil
}

// PaneCommands returns the foreground command of every pane in a window.
func (l *Live) PaneCommands(session string, windowIndex int) ([]string, error) {
	target := fmt.Sprintf("%s:%d", session, windowIndex)
	out, err := run("list-panes", "-t", target, "-F", "#{pane_current_command}")
	if err != nil {
		return nil, fmt.Errorf("pane-commands %s: %w", target, err)
	}
	var cmds []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line != "" {
			cmds = append(cmds, line)
		}
	}
	return cmds, nil
}

func (l *Live) CapturePane(target string, lines int) (string, error) {
	// Capture visible pane content only (no scrollback history)
	out, err := run("capture-pane", "-t", target, "-p")
	if err != nil {
		return "", fmt.Errorf("capture-pane %s: %w", target, err)
	}
	return out, nil
}

// CapturePaneByIndex captures a specific pane by session:window.pane index.
func (l *Live) CapturePaneByIndex(session string, windowIndex, paneIndex, lines int) (string, error) {
	target := fmt.Sprintf("%s:%d.%d", session, windowIndex, paneIndex)
	return l.CapturePane(target, lines)
}

func (l *Live) SwitchWindow(session string, windowIndex int) error {
	target := fmt.Sprintf("%s:%d", session, windowIndex)
	_, err := run("select-window", "-t", target)
	if err != nil {
		return fmt.Errorf("select-window %s: %w", target, err)
	}
	_, err = run("switch-client", "-t", session)
	if err != nil {
		return err
	}
	// Focus the main pane (not clank) — find the last non-clank pane
	out, _ := run("list-panes", "-t", target, "-F", "#{pane_id}\t#{pane_current_command}")
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 && parts[1] != "clank" && parts[1] != "cat" {
			run("select-pane", "-t", parts[0])
			break
		}
	}
	return nil
}

func run(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	out, err := cmd.Output()
	return string(out), err
}

// ParseSessionList parses tmux list-sessions output.
// Format: "name:window_count:attached\n"
// Assigns sequential 1-based indices.
func ParseSessionList(raw string) []Session {
	var sessions []Session
	idx := 1
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		wc, _ := strconv.Atoi(parts[1])
		attached := parts[2] == "1"
		sessions = append(sessions, Session{
			Index:       idx,
			Name:        parts[0],
			WindowCount: wc,
			Attached:    attached,
		})
		idx++
	}
	return sessions
}

// ParseWindowList parses tmux list-windows output.
// Format: "index:name:active\n"
func ParseWindowList(raw string) []Window {
	var windows []Window
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		i, _ := strconv.Atoi(parts[0])
		active := parts[2] == "1"
		windows = append(windows, Window{
			Index:  i,
			Name:   parts[1],
			Active: active,
		})
	}
	return windows
}

func ParsePaneCommand(raw string) string {
	return strings.TrimSpace(raw)
}
