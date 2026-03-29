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

type Mux interface {
	ListSessions() ([]Session, error)
	ListWindows(session string) ([]Window, error)
	PaneCommand(target string) (string, error)
	CapturePane(target string, lines int) (string, error)
	SwitchWindow(session string, windowIndex int) error
}

type Live struct{}

func (l *Live) ListSessions() ([]Session, error) {
	out, err := run("list-sessions", "-F", "#{session_id}:#{session_name}:#{session_windows}:#{session_attached}")
	if err != nil {
		return nil, fmt.Errorf("list-sessions: %w", err)
	}
	return ParseSessionList(out), nil
}

func (l *Live) ListWindows(session string) ([]Window, error) {
	out, err := run("list-windows", "-t", session, "-F", "#{window_index}:#{window_name}:#{window_active}")
	if err != nil {
		return nil, fmt.Errorf("list-windows %s: %w", session, err)
	}
	return ParseWindowList(out), nil
}

func (l *Live) PaneCommand(target string) (string, error) {
	out, err := run("display-message", "-t", target, "-p", "#{pane_current_command}")
	if err != nil {
		return "", fmt.Errorf("pane-command %s: %w", target, err)
	}
	return ParsePaneCommand(out), nil
}

func (l *Live) CapturePane(target string, lines int) (string, error) {
	start := fmt.Sprintf("-%d", lines)
	out, err := run("capture-pane", "-t", target, "-p", "-S", start)
	if err != nil {
		return "", fmt.Errorf("capture-pane %s: %w", target, err)
	}
	return out, nil
}

func (l *Live) SwitchWindow(session string, windowIndex int) error {
	target := fmt.Sprintf("%s:%d", session, windowIndex)
	_, err := run("select-window", "-t", target)
	if err != nil {
		return fmt.Errorf("select-window %s: %w", target, err)
	}
	_, err = run("switch-client", "-t", session)
	return err
}

func run(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	out, err := cmd.Output()
	return string(out), err
}

func ParseSessionList(raw string) []Session {
	var sessions []Session
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 4)
		if len(parts) < 4 {
			continue
		}
		idx, _ := strconv.Atoi(parts[0])
		wc, _ := strconv.Atoi(parts[2])
		attached := parts[3] == "1"
		sessions = append(sessions, Session{
			Index:       idx,
			Name:        parts[1],
			WindowCount: wc,
			Attached:    attached,
		})
	}
	return sessions
}

func ParseWindowList(raw string) []Window {
	var windows []Window
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		idx, _ := strconv.Atoi(parts[0])
		active := parts[2] == "1"
		windows = append(windows, Window{
			Index:  idx,
			Name:   parts[1],
			Active: active,
		})
	}
	return windows
}

func ParsePaneCommand(raw string) string {
	return strings.TrimSpace(raw)
}
