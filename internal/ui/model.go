package ui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/voska/clank/internal/detect"
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

		if !sess.Collapsed {
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
		hp := m.store.HighestPriority(sess.Name)
		selected := m.cursor == itemIdx
		b.WriteString(renderSessionHeader(sess.Index, sess.Name, sess.Collapsed, hp, selected, m.width))
		b.WriteString("\n")
		itemIdx++

		if !sess.Collapsed {
			for _, win := range sess.Windows {
				selected = m.cursor == itemIdx
				b.WriteString(renderWindowLine(win, m.width, selected, m.spinnerFrame))
				b.WriteString("\n")
				itemIdx++
			}
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
	case "enter":
		if m.cursor >= 0 && m.cursor < len(m.items) {
			it := m.items[m.cursor]
			if it.isSession {
				m.store.ToggleCollapse(it.session)
				m.rebuildItems()
			} else {
				m.mux.SwitchWindow(it.session, it.winIndex)
				m.store.ClearUnseen(it.session, it.window)
			}
		}
	case " ":
		if m.cursor >= 0 && m.cursor < len(m.items) {
			it := m.items[m.cursor]
			if it.isSession {
				m.store.ToggleCollapse(it.session)
				m.rebuildItems()
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
		m.store.ToggleCollapse(it.session)
		m.rebuildItems()
	} else {
		m.mux.SwitchWindow(it.session, it.winIndex)
		m.store.ClearUnseen(it.session, it.window)
	}
	return m, nil
}

func (m Model) handleSocketEvent(ev socket.Event) {
	info := state.AgentInfo{
		Name:       ev.Agent,
		State:      detect.State(ev.State),
		WaitReason: ev.Reason,
		Model:      ev.Model,
		ContextPct: ev.ContextPct,
	}
	m.store.SetAgentState(ev.Session, ev.Window, info)
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
			return nil
		}

		var snapshots []state.SessionSnapshot
		for _, sess := range sessions {
			if sess.Name == "_clank" {
				continue
			}
			windows, err := m.mux.ListWindows(sess.Name)
			if err != nil {
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
		return nil
	}
}

func (m Model) scrapeAgents() tea.Cmd {
	return func() tea.Msg {
		sessions := m.store.Sessions()
		for _, sess := range sessions {
			for _, win := range sess.Windows {
				cmds, err := m.mux.PaneCommands(sess.Name, win.Index)
				if err != nil {
					continue
				}

				agentName := ""
				agentPaneIdx := -1
				for i, cmd := range cmds {
					name := detect.IdentifyAgent(cmd)
					if name != "" {
						agentName = name
						agentPaneIdx = i
						break
					}
				}

				if agentName == "" {
					if win.Agent != nil && win.Agent.State != detect.StateDone {
						m.store.ClearAgent(sess.Name, win.Name)
					}
					continue
				}

				content, err := m.mux.CapturePaneByIndex(sess.Name, win.Index, agentPaneIdx, 20)
				if err != nil {
					continue
				}
				agentState, reason := detect.DetectState(content)
				if agentState == detect.StateUnknown {
					agentState = detect.StateRunning
				}

				existing := win.Agent
				if existing != nil && existing.Model != "" {
					m.store.SetAgentState(sess.Name, win.Name, state.AgentInfo{
						Name:       agentName,
						State:      agentState,
						WaitReason: reason,
						Model:      existing.Model,
						ContextPct: existing.ContextPct,
					})
				} else {
					m.store.SetAgentState(sess.Name, win.Name, state.AgentInfo{
						Name:       agentName,
						State:      agentState,
						WaitReason: reason,
					})
				}
			}
		}
		return nil
	}
}
