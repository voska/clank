package state

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/voska/clank/internal/detect"
)

type AgentInfo struct {
	Name        string
	State       detect.State
	WaitReason  string
	Model       string
	ContextPct  float64
	LastChanged time.Time
	FromHook       bool // true if last update came from a hook
	FromTranscript bool // true if last update came from JSONL transcript watching
}

type WindowState struct {
	Name      string
	Index     int
	Active    bool
	HasUnseen bool
	Agent     *AgentInfo
}

type SessionState struct {
	Name      string
	Index     int
	Collapsed bool
	Windows   []WindowState
}

type SessionSnapshot struct {
	Name    string
	Index   int
	Windows []WindowSnapshot
}

type WindowSnapshot struct {
	Name   string
	Index  int
	Active bool
}

type Store struct {
	mu        sync.RWMutex
	sessions  []SessionState
	collapsed map[string]bool
	persisted map[string]persistedAgent // restored from disk on startup
}

func New() *Store {
	return &Store{
		collapsed: make(map[string]bool),
	}
}

func (s *Store) UpdateTree(snapshots []SessionSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing := make(map[string]*AgentInfo)
	unseen := make(map[string]bool)

	for _, sess := range s.sessions {
		for _, win := range sess.Windows {
			key := winKey(sess.Name, win.Index)
			if win.Agent != nil {
				agentCopy := *win.Agent
				existing[key] = &agentCopy
			}
			if win.HasUnseen {
				unseen[key] = true
			}
		}
	}

	newSessions := make([]SessionState, 0, len(snapshots))
	for _, snap := range snapshots {
		sess := SessionState{
			Name:      snap.Name,
			Index:     snap.Index,
			Collapsed: s.collapsed[snap.Name],
		}
		for _, wSnap := range snap.Windows {
			key := winKey(snap.Name, wSnap.Index)
			agent := existing[key]
			if agent == nil {
				agent = s.restoreAgent(key)
			}
			win := WindowState{
				Name:      wSnap.Name,
				Index:     wSnap.Index,
				Active:    wSnap.Active,
				HasUnseen: unseen[key],
				Agent:     agent,
			}
			sess.Windows = append(sess.Windows, win)
		}
		newSessions = append(newSessions, sess)
	}
	s.sessions = newSessions
}

func (s *Store) Sessions() []SessionState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]SessionState, len(s.sessions))
	for i, sess := range s.sessions {
		result[i] = SessionState{
			Name:      sess.Name,
			Index:     sess.Index,
			Collapsed: sess.Collapsed,
			Windows:   make([]WindowState, len(sess.Windows)),
		}
		copy(result[i].Windows, sess.Windows)
	}
	return result
}

func (s *Store) SetAgentState(session string, windowIndex int, info AgentInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, sess := range s.sessions {
		if sess.Name != session {
			continue
		}
		for j, win := range sess.Windows {
			if win.Index != windowIndex {
				continue
			}
			existing := s.sessions[i].Windows[j].Agent
			stateChanged := existing == nil || existing.State != info.State
			if stateChanged {
				info.LastChanged = time.Now()
				if !win.Active {
					s.sessions[i].Windows[j].HasUnseen = true
				}
			} else {
				info.LastChanged = existing.LastChanged
			}
			s.sessions[i].Windows[j].Agent = &info
			return
		}
	}
}

func (s *Store) ClearAgent(session string, windowIndex int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, sess := range s.sessions {
		if sess.Name != session {
			continue
		}
		for j, win := range sess.Windows {
			if win.Index != windowIndex {
				continue
			}
			s.sessions[i].Windows[j].Agent = nil
			return
		}
	}
}

func (s *Store) ClearUnseen(session string, windowIndex int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, sess := range s.sessions {
		if sess.Name != session {
			continue
		}
		for j, win := range sess.Windows {
			if win.Index != windowIndex {
				continue
			}
			s.sessions[i].Windows[j].HasUnseen = false
			return
		}
	}
}

func (s *Store) ToggleCollapse(session string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.collapsed[session] = !s.collapsed[session]
	for i, sess := range s.sessions {
		if sess.Name == session {
			s.sessions[i].Collapsed = s.collapsed[session]
			return
		}
	}
}

func (s *Store) HighestPriority(session string) detect.State {
	s.mu.RLock()
	defer s.mu.RUnlock()

	priority := map[detect.State]int{
		detect.StateWaiting: 4,
		detect.StateError:   3,
		detect.StateRunning: 2,
		detect.StateDone:    1,
		detect.StateExited:  0,
		detect.StateUnknown: 0,
	}

	highest := detect.State("")
	highestPri := -1

	for _, sess := range s.sessions {
		if sess.Name != session {
			continue
		}
		for _, win := range sess.Windows {
			if win.Agent == nil {
				continue
			}
			p := priority[win.Agent.State]
			if p > highestPri {
				highestPri = p
				highest = win.Agent.State
			}
		}
	}
	return highest
}

// persistedState is the JSON-serializable format for state persistence.
type persistedState struct {
	Agents map[string]persistedAgent `json:"agents"` // key: "session:window"
}

type persistedAgent struct {
	Name        string       `json:"name"`
	State       detect.State `json:"state"`
	LastChanged time.Time    `json:"last_changed"`
	FromHook    bool         `json:"from_hook"`
}

// Save persists agent state to a file so timers survive restarts.
func (s *Store) Save(path string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ps := persistedState{Agents: make(map[string]persistedAgent)}
	for _, sess := range s.sessions {
		for _, win := range sess.Windows {
			if win.Agent == nil {
				continue
			}
			key := winKey(sess.Name, win.Index)
			ps.Agents[key] = persistedAgent{
				Name:        win.Agent.Name,
				State:       win.Agent.State,
				LastChanged: win.Agent.LastChanged,
				FromHook:    win.Agent.FromHook,
			}
		}
	}

	data, err := json.Marshal(ps)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Load restores agent state from a file. Call before the first UpdateTree.
func (s *Store) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var ps persistedState
	if err := json.Unmarshal(data, &ps); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.persisted = ps.Agents
	return nil
}

// restoreAgent checks persisted state when a new agent is first detected.
func (s *Store) restoreAgent(key string) *AgentInfo {
	if s.persisted == nil {
		return nil
	}
	pa, ok := s.persisted[key]
	if !ok {
		return nil
	}
	return &AgentInfo{
		Name:        pa.Name,
		State:       pa.State,
		LastChanged: pa.LastChanged,
		FromHook:    false, // don't trust old hook data after restart
	}
}
