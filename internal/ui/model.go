package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/voska/clank/internal/detect"
	"github.com/voska/clank/internal/logger"
	"github.com/voska/clank/internal/socket"
	"github.com/voska/clank/internal/state"
	"github.com/voska/clank/internal/tmux"
)

type Config struct {
	Width          int
	PollInterval   time.Duration
	ScrapeInterval time.Duration
}

type Model struct {
	config       Config
	store        *state.Store
	mux          tmux.Mux
	events       <-chan socket.Event
	cursor       int
	spinnerFrame int
	width        int
	height       int
	items        []item
	screenRows   []int
}

type item struct {
	isSession bool
	session   string
	window    string
	sessIndex int
	winIndex  int
}

type tickMsg struct{}
type scrapeMsg struct{}
type spinnerMsg struct{}
type socketEventMsg socket.Event

func NewModel(store *state.Store, mux tmux.Mux, events <-chan socket.Event, config Config) Model {
	return Model{
		config: config,
		store:  store,
		mux:    mux,
		events: events,
		width:  config.Width,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.pollTmux(),
		m.tickCmd(),
		m.scrapeCmd(),
		m.spinnerCmd(),
		m.listenSocket(),
	)
}

// rebuildItems flattens the session/window tree into a navigable list
// and maps screen rows to item indices. Must be called after any state change.
func (m *Model) rebuildItems() {
	m.items = nil
	m.screenRows = nil

	sessions := m.store.Sessions()

	// Top padding
	m.screenRows = append(m.screenRows, -1)

	for si, sess := range sessions {
		m.screenRows = append(m.screenRows, len(m.items))
		m.items = append(m.items, item{isSession: true, session: sess.Name, sessIndex: sess.Index})

		for _, win := range sess.Windows {
			m.screenRows = append(m.screenRows, len(m.items))
			m.items = append(m.items, item{
				isSession: false,
				session:   sess.Name,
				window:    win.Name,
				sessIndex: sess.Index,
				winIndex:  win.Index,
			})
		}

		if si < len(sessions)-1 {
			m.screenRows = append(m.screenRows, -1)
		}
	}

	// Clamp cursor
	if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.MouseMsg:
		return m.handleMouse(msg)
	case tea.WindowSizeMsg:
		if m.config.Width > 0 {
			m.width = m.config.Width
		} else {
			m.width = msg.Width
		}
		m.height = msg.Height
		m.rebuildItems()
		return m, nil
	case tickMsg:
		m.rebuildItems()
		return m, tea.Batch(m.pollTmux(), m.tickCmd())
	case scrapeMsg:
		m.rebuildItems()
		return m, tea.Batch(m.scrapeAgents(), m.scrapeCmd())
	case spinnerMsg:
		m.spinnerFrame++
		return m, m.spinnerCmd()
	case socketEventMsg:
		m.handleSocketEvent(socket.Event(msg))
		m.rebuildItems()
		return m, m.listenSocket()
	}
	return m, nil
}

func (m Model) View() string {
	sessions := m.store.Sessions()

	var b strings.Builder
	b.WriteString("\n") // top padding

	itemIdx := 0
	for si, sess := range sessions {
		selected := m.cursor == itemIdx
		b.WriteString(renderSessionHeader(sess.Index, sess.Name, false, "", selected, m.width))
		b.WriteString("\n")
		itemIdx++

		for _, win := range sess.Windows {
			selected = m.cursor == itemIdx
			b.WriteString(renderWindowLine(win, m.width, selected, m.spinnerFrame))
			b.WriteString("\n")
			itemIdx++
		}

		if si < len(sessions)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "enter", " ":
		if m.cursor >= 0 && m.cursor < len(m.items) {
			it := m.items[m.cursor]
			if it.isSession {
				m.mux.SwitchWindow(it.session, 0)
			} else {
				m.mux.SwitchWindow(it.session, it.winIndex)
				m.store.ClearUnseen(it.session, it.winIndex)
			}
		}
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		idx := int(msg.String()[0] - '0')
		for i, it := range m.items {
			if it.isSession && it.sessIndex == idx {
				m.cursor = i
				break
			}
		}
	}
	return m, nil
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	row := msg.Y
	if row < 0 || row >= len(m.screenRows) {
		return m, nil
	}
	idx := m.screenRows[row]
	if idx < 0 || idx >= len(m.items) {
		return m, nil
	}
	m.cursor = idx
	it := m.items[idx]
	if it.isSession {
		m.mux.SwitchWindow(it.session, 0)
	} else {
		m.mux.SwitchWindow(it.session, it.winIndex)
		m.store.ClearUnseen(it.session, it.winIndex)
	}
	return m, nil
}

// resolveWindowIndex finds the window index for a session:window name pair.
func (m Model) resolveWindowIndex(session, window string) int {
	for _, sess := range m.store.Sessions() {
		if sess.Name != session {
			continue
		}
		for _, win := range sess.Windows {
			if win.Name == window {
				return win.Index
			}
		}
	}
	return -1
}

func (m Model) handleSocketEvent(ev socket.Event) {
	logger.Log("hook  %s:%s agent=%s state=%s reason=%s model=%s ctx=%.0f", ev.Session, ev.Window, ev.Agent, ev.State, ev.Reason, ev.Model, ev.ContextPct)

	winIdx := m.resolveWindowIndex(ev.Session, ev.Window)
	if winIdx < 0 {
		return
	}

	// Metadata-only event (statusline): update model/context without changing state
	if ev.State == "" {
		for _, sess := range m.store.Sessions() {
			if sess.Name != ev.Session {
				continue
			}
			for _, win := range sess.Windows {
				if win.Index == winIdx && win.Agent != nil {
					info := *win.Agent
					if ev.Model != "" {
						info.Model = ev.Model
					}
					if ev.ContextPct > 0 {
						info.ContextPct = ev.ContextPct
					}
					m.store.SetAgentState(ev.Session, winIdx, info)
					return
				}
			}
		}
		return
	}

	info := state.AgentInfo{
		Name:       ev.Agent,
		State:      detect.State(ev.State),
		WaitReason: ev.Reason,
		Model:      ev.Model,
		ContextPct: ev.ContextPct,
		FromHook:   true,
	}
	m.store.SetAgentState(ev.Session, winIdx, info)
}

func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(m.config.PollInterval, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m Model) scrapeCmd() tea.Cmd {
	return tea.Tick(m.config.ScrapeInterval, func(time.Time) tea.Msg {
		return scrapeMsg{}
	})
}

func (m Model) spinnerCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
		return spinnerMsg{}
	})
}

func (m Model) listenSocket() tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-m.events
		if !ok {
			return nil
		}
		return socketEventMsg(ev)
	}
}

func (m Model) pollTmux() tea.Cmd {
	return func() tea.Msg {
		sessions, err := m.mux.ListSessions()
		if err != nil {
			logger.Log("poll  error listing sessions: %v", err)
			return nil
		}

		var snapshots []state.SessionSnapshot
		for _, sess := range sessions {
			if sess.Name == "_clank" {
				continue
			}
			windows, err := m.mux.ListWindows(sess.Name)
			if err != nil {
				logger.Log("poll  error listing windows for %s: %v", sess.Name, err)
				continue
			}
			snap := state.SessionSnapshot{
				Name:  sess.Name,
				Index: sess.Index,
			}
			for _, win := range windows {
				snap.Windows = append(snap.Windows, state.WindowSnapshot{
					Name:   win.Name,
					Index:  win.Index,
					Active: win.Active,
				})
			}
			snapshots = append(snapshots, snap)
		}
		m.store.UpdateTree(snapshots)

		// Auto-clear unseen when user visits a window (active = they're looking at it)
		for _, snap := range snapshots {
			for _, win := range snap.Windows {
				if win.Active {
					m.store.ClearUnseen(snap.Name, win.Index)
				}
			}
		}

		return nil
	}
}

func (m Model) scrapeAgents() tea.Cmd {
	return func() tea.Msg {
		// Single tmux call to get ALL panes across all sessions
		allPanes, err := m.mux.AllPanes()
		if err != nil {
			logger.Log("scrape error: %v", err)
			return nil
		}

		// Group by session:window, find agent panes
		type agentPane struct {
			name      string
			session   string
			window    string
			winIndex  int
			paneIndex int
		}
		var agents []agentPane
		seen := make(map[string]bool) // "session:winIndex" -> has agent

		for _, p := range allPanes {
			key := fmt.Sprintf("%s:%d", p.Session, p.WinIndex)
			if seen[key] {
				continue
			}
			agentName := detect.IdentifyAgent(p.Command)
			if agentName != "" {
				seen[key] = true
				agents = append(agents, agentPane{
					name:      agentName,
					session:   p.Session,
					window:    p.Window,
					winIndex:  p.WinIndex,
					paneIndex: p.PaneIndex,
				})
			}
		}

		// Clear agents from windows that no longer have one
		sessions := m.store.Sessions()
		for _, sess := range sessions {
			for _, win := range sess.Windows {
				key := fmt.Sprintf("%s:%d", sess.Name, win.Index)
				if !seen[key] && win.Agent != nil && win.Agent.State != detect.StateDone {
					logger.Log("scrape %s agent gone, clearing", key)
					m.store.ClearAgent(sess.Name, win.Index)
				}
			}
		}

		// Scrape content for each agent pane (only the ones that need it)
		for _, ap := range agents {
			// Skip if hook data is fresh
			sessions := m.store.Sessions()
			for _, sess := range sessions {
				if sess.Name != ap.session {
					continue
				}
				for _, win := range sess.Windows {
					if win.Index != ap.winIndex {
						continue
					}
					if win.Agent != nil && win.Agent.FromHook && time.Since(win.Agent.LastChanged) < 10*time.Second {
						logger.Log("scrape %s:%s skipping — hook fresh (%s)", ap.session, ap.window, time.Since(win.Agent.LastChanged).Round(time.Millisecond))
						goto nextAgent
					}
				}
			}

			{
				content, err := m.mux.CapturePaneByIndex(ap.session, ap.winIndex, ap.paneIndex, 30)
				if err != nil {
					continue
				}
				agentState, reason := detect.DetectState(content)

				// Look up existing state to avoid flipping done→running on unknown
				var existingState detect.State
				for _, sess := range m.store.Sessions() {
					if sess.Name != ap.session {
						continue
					}
					for _, win := range sess.Windows {
						if win.Index == ap.winIndex && win.Agent != nil {
							existingState = win.Agent.State
						}
					}
				}

				switch agentState {
				case detect.StateUnknown:
					if existingState != "" {
						agentState = existingState // preserve existing state
					} else {
						agentState = detect.StateRunning // new agent, assume running
					}
				case detect.StateExited:
					agentState = detect.StateDone
				}

				logger.Log("scrape %s:%s agent=%s state=%s", ap.session, ap.window, ap.name, agentState)

				// Re-check hook freshness before writing — a hook may have fired
				// during capture-pane (race condition)
				var model string
				var ctxPct float64
				skipWrite := false
				for _, sess := range m.store.Sessions() {
					if sess.Name != ap.session {
						continue
					}
					for _, win := range sess.Windows {
						if win.Index == ap.winIndex && win.Agent != nil {
							model = win.Agent.Model
							ctxPct = win.Agent.ContextPct
							if win.Agent.FromHook && time.Since(win.Agent.LastChanged) < 10*time.Second {
								skipWrite = true
							}
						}
					}
				}
				if skipWrite {
					logger.Log("scrape %s:%s skipping write — hook arrived during scrape", ap.session, ap.window)
					goto nextAgent
				}

				m.store.SetAgentState(ap.session, ap.winIndex, state.AgentInfo{
					Name:       ap.name,
					State:      agentState,
					WaitReason: reason,
					Model:      model,
					ContextPct: ctxPct,
				})
			}
		nextAgent:
		}
		return nil
	}
}
