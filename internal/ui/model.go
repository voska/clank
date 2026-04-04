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
	"github.com/voska/clank/internal/transcript"
)

type Config struct {
	Width          int
	PollInterval   time.Duration
	ScrapeInterval time.Duration
}

type Model struct {
	config            Config
	store             *state.Store
	mux               tmux.Mux
	events            <-chan socket.Event
	transcriptWatcher *transcript.Watcher
	cursor            int
	spinnerFrame      int
	width             int
	height            int
	items             []item
	screenRows        []int
	pruned            bool
	filterActive      bool
}

type item struct {
	isSession bool
	session   string
	window    string
	sessIndex int
	winIndex  int
}

// agentPane holds info about a detected agent pane from tmux.
type agentPane struct {
	name      string
	session   string
	window    string
	winIndex  int
	paneIndex int
	panePID   int
	title     string
}

type tickMsg struct{}
type scrapeMsg struct{}
type spinnerMsg struct{}
type socketEventMsg socket.Event
type transcriptUpdateMsg transcript.StateUpdate

type pollResultMsg struct {
	snapshots []state.SessionSnapshot
}

type scrapeResultMsg struct {
	updates []scrapedAgent
	exits   []agentExit
}

type scrapedAgent struct {
	session  string
	winIndex int
	info     state.AgentInfo
}

type agentExit struct {
	session  string
	winIndex int
	name     string
}

func NewModel(store *state.Store, mux tmux.Mux, events <-chan socket.Event, config Config, tw *transcript.Watcher) Model {
	return Model{
		config:            config,
		store:             store,
		mux:               mux,
		events:            events,
		transcriptWatcher: tw,
		width:             config.Width,
	}
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.pollTmux(),
		m.tickCmd(),
		m.scrapeCmd(),
		m.spinnerCmd(),
		m.listenSocket(),
	}
	if m.transcriptWatcher != nil {
		cmds = append(cmds, m.listenTranscripts())
	}
	return tea.Batch(cmds...)
}

func (m Model) visibleSessions() []state.SessionState {
	sessions := m.store.Sessions()
	if !m.filterActive {
		return sessions
	}
	var filtered []state.SessionState
	for _, s := range sessions {
		if m.store.HasActiveAgents(s.Name) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// rebuildItems flattens the session/window tree into a navigable list
// and maps screen rows to item indices. Must be called after any state change.
func (m *Model) rebuildItems() {
	m.items = nil
	m.screenRows = nil

	sessions := m.visibleSessions()

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
		return m, m.pollTmux()
	case pollResultMsg:
		m.applyPollResult(msg)
		m.rebuildItems()
		return m, m.tickCmd()
	case scrapeMsg:
		return m, m.scrapeAgents()
	case scrapeResultMsg:
		m.applyScrapeResult(msg)
		m.rebuildItems()
		return m, m.scrapeCmd()
	case spinnerMsg:
		m.spinnerFrame++
		return m, m.spinnerCmd()
	case socketEventMsg:
		m.handleSocketEvent(socket.Event(msg))
		m.rebuildItems()
		return m, m.listenSocket()
	case transcriptUpdateMsg:
		m.handleTranscriptUpdate(transcript.StateUpdate(msg))
		m.rebuildItems()
		return m, m.listenTranscripts()
	}
	return m, nil
}

func (m Model) View() string {
	sessions := m.visibleSessions()

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

	if m.filterActive {
		if len(m.items) == 0 {
			b.WriteString(dimStyle.Render("  no active agents"))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  [f] filtered"))
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
	case "f":
		m.filterActive = !m.filterActive
		m.rebuildItems()
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
		existing := m.store.GetAgent(ev.Session, winIdx)
		if existing != nil {
			if ev.Model != "" {
				existing.Model = ev.Model
			}
			if ev.ContextPct > 0 {
				existing.ContextPct = ev.ContextPct
			}
			m.store.SetAgentState(ev.Session, winIdx, *existing)
		}
		return
	}

	m.store.SetAgentState(ev.Session, winIdx, state.AgentInfo{
		Name:       ev.Agent,
		State:      detect.State(ev.State),
		WaitReason: ev.Reason,
		Model:      ev.Model,
		ContextPct: ev.ContextPct,
		FromHook:   true,
	})
}

func (m Model) handleTranscriptUpdate(update transcript.StateUpdate) {
	logger.Log("transcript %s:%d agent=%s state=%s", update.Session, update.WinIndex, update.AgentName, update.State)
	existing := m.store.GetAgent(update.Session, update.WinIndex)
	var model string
	var ctxPct float64
	if existing != nil {
		model = existing.Model
		ctxPct = existing.ContextPct
	}
	m.store.SetAgentState(update.Session, update.WinIndex, state.AgentInfo{
		Name:           update.AgentName,
		State:          update.State,
		Model:          model,
		ContextPct:     ctxPct,
		FromTranscript: true,
	})
}

// --- Timer commands ---

func (m Model) listenTranscripts() tea.Cmd {
	if m.transcriptWatcher == nil {
		return nil
	}
	return func() tea.Msg {
		update, ok := <-m.transcriptWatcher.Updates
		if !ok {
			return nil
		}
		return transcriptUpdateMsg(update)
	}
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

// --- Poll ---

func (m Model) pollTmux() tea.Cmd {
	return func() tea.Msg {
		sessions, err := m.mux.ListSessions()
		if err != nil {
			logger.Log("poll  error listing sessions: %v", err)
			return pollResultMsg{}
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
		return pollResultMsg{snapshots: snapshots}
	}
}

func (m *Model) applyPollResult(msg pollResultMsg) {
	m.store.UpdateTree(msg.snapshots)

	if !m.pruned {
		m.store.PruneStale()
		m.pruned = true
	}

	for _, snap := range msg.snapshots {
		for _, win := range snap.Windows {
			if win.Active {
				m.store.ClearUnseen(snap.Name, win.Index)
			}
		}
	}
}

// --- Scrape ---

func identifyAgentPanes(allPanes []tmux.PaneInfo) ([]agentPane, []int) {
	var agents []agentPane
	var claudePIDs []int
	seen := make(map[string]bool)

	for _, p := range allPanes {
		key := fmt.Sprintf("%s:%d", p.Session, p.WinIndex)
		if seen[key] {
			continue
		}
		agentName := detect.IdentifyAgent(p.Command)
		if agentName == "" {
			continue
		}
		seen[key] = true
		agents = append(agents, agentPane{
			name:      agentName,
			session:   p.Session,
			window:    p.Window,
			winIndex:  p.WinIndex,
			paneIndex: p.PaneIndex,
			panePID:   p.PanePID,
			title:     p.Title,
		})
		if agentName == "claude" {
			claudePIDs = append(claudePIDs, p.PanePID)
		}
	}
	return agents, claudePIDs
}

func (m Model) trackTranscripts(agents []agentPane, claudePIDs []int) {
	transcriptMap := transcript.MapPanesToTranscripts(claudePIDs)
	for _, ap := range agents {
		if ap.name != "claude" {
			continue
		}
		if tpath, ok := transcriptMap[ap.panePID]; ok {
			m.transcriptWatcher.Track(ap.session, ap.winIndex, ap.name, tpath)
		}
	}
}

func (m Model) findExitedAgents(seen map[string]bool) []agentExit {
	var exits []agentExit
	for _, sess := range m.store.Sessions() {
		for _, win := range sess.Windows {
			key := fmt.Sprintf("%s:%d", sess.Name, win.Index)
			if !seen[key] && win.Agent != nil && win.Agent.State != detect.StateDone {
				logger.Log("scrape %s agent exited, transitioning to done", key)
				exits = append(exits, agentExit{
					session:  sess.Name,
					winIndex: win.Index,
					name:     win.Agent.Name,
				})
			}
		}
	}
	return exits
}

func (m Model) shouldSkipScrape(ap agentPane) bool {
	agent := m.store.GetAgent(ap.session, ap.winIndex)
	if agent == nil {
		return false
	}
	if agent.FromTranscript {
		return true
	}
	if agent.FromHook && time.Since(agent.LastChanged) < 10*time.Second {
		logger.Log("scrape %s:%s skipping — hook fresh (%s)", ap.session, ap.window, time.Since(agent.LastChanged).Round(time.Millisecond))
		return true
	}
	return false
}

func (m Model) detectAgentState(ap agentPane) (detect.State, string) {
	titleState := detect.DetectStateFromTitle(ap.title)

	if titleState == detect.StateRunning {
		return detect.StateRunning, ""
	}

	if titleState == detect.StateDone {
		content, err := m.mux.CapturePaneByIndex(ap.session, ap.winIndex, ap.paneIndex)
		if err != nil {
			return detect.StateDone, ""
		}
		if waiting, reason := detect.DetectWaiting(content); waiting {
			return detect.StateWaiting, reason
		}
		return detect.StateDone, ""
	}

	// No title match — full content scrape fallback
	content, err := m.mux.CapturePaneByIndex(ap.session, ap.winIndex, ap.paneIndex)
	if err != nil {
		return detect.StateUnknown, ""
	}
	agentState, reason := detect.DetectState(content)

	if agentState == detect.StateUnknown {
		existing := m.store.GetAgent(ap.session, ap.winIndex)
		if existing != nil && existing.State != "" {
			return existing.State, ""
		}
		return detect.StateRunning, ""
	}

	if agentState == detect.StateExited {
		return detect.StateDone, ""
	}

	return agentState, reason
}

func (m Model) scrapeAgents() tea.Cmd {
	return func() tea.Msg {
		allPanes, err := m.mux.AllPanes()
		if err != nil {
			logger.Log("scrape error: %v", err)
			return scrapeResultMsg{}
		}

		agents, claudePIDs := identifyAgentPanes(allPanes)

		if m.transcriptWatcher != nil && len(claudePIDs) > 0 {
			m.trackTranscripts(agents, claudePIDs)
		}

		seen := make(map[string]bool)
		for _, ap := range agents {
			seen[fmt.Sprintf("%s:%d", ap.session, ap.winIndex)] = true
		}

		exits := m.findExitedAgents(seen)

		var updates []scrapedAgent
		for _, ap := range agents {
			if m.shouldSkipScrape(ap) {
				continue
			}
			agentState, reason := m.detectAgentState(ap)
			logger.Log("scrape %s:%s agent=%s state=%s (title=%q)", ap.session, ap.window, ap.name, agentState, ap.title)
			updates = append(updates, scrapedAgent{
				session:  ap.session,
				winIndex: ap.winIndex,
				info: state.AgentInfo{
					Name:       ap.name,
					State:      agentState,
					WaitReason: reason,
				},
			})
		}

		return scrapeResultMsg{updates: updates, exits: exits}
	}
}

func (m *Model) applyScrapeResult(msg scrapeResultMsg) {
	for _, exit := range msg.exits {
		m.store.SetAgentState(exit.session, exit.winIndex, state.AgentInfo{
			Name:  exit.name,
			State: detect.StateDone,
		})
	}

	for _, sa := range msg.updates {
		existing := m.store.GetAgent(sa.session, sa.winIndex)
		// Skip if fresher data arrived while we were scraping
		if existing != nil {
			if existing.FromTranscript {
				continue
			}
			if existing.FromHook && time.Since(existing.LastChanged) < 10*time.Second {
				continue
			}
			sa.info.Model = existing.Model
			sa.info.ContextPct = existing.ContextPct
		}
		m.store.SetAgentState(sa.session, sa.winIndex, sa.info)
	}
}
