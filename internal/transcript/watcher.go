package transcript

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/voska/clank/internal/detect"
	"github.com/voska/clank/internal/logger"
)

// StateUpdate is sent when a transcript file change indicates a state transition.
type StateUpdate struct {
	Session   string
	WinIndex  int
	AgentName string
	State     detect.State
}

// tracked holds metadata for a watched transcript file.
type tracked struct {
	Session    string
	WinIndex   int
	AgentName  string
	LastOffset int64
	LastState  detect.State
}

// Watcher monitors Claude Code JSONL transcript files for state changes.
type Watcher struct {
	fsw     *fsnotify.Watcher
	mu      sync.Mutex
	files   map[string]*tracked // path → tracked
	Updates chan StateUpdate
}

// NewWatcher creates a transcript watcher.
func NewWatcher() (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		fsw:     fsw,
		files:   make(map[string]*tracked),
		Updates: make(chan StateUpdate, 100),
	}, nil
}

// Track starts watching a transcript file.
func (w *Watcher) Track(session string, winIndex int, agentName string, transcriptPath string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, ok := w.files[transcriptPath]; ok {
		// Already tracking — update session/window mapping (might have changed)
		t := w.files[transcriptPath]
		t.Session = session
		t.WinIndex = winIndex
		return nil
	}

	// Read initial state
	state, err := ReadLastState(transcriptPath)
	if err != nil {
		return err
	}

	stat, _ := os.Stat(transcriptPath)
	offset := int64(0)
	if stat != nil {
		offset = stat.Size()
	}

	w.files[transcriptPath] = &tracked{
		Session:    session,
		WinIndex:   winIndex,
		AgentName:  agentName,
		LastOffset: offset,
		LastState:  state,
	}

	if err := w.fsw.Add(transcriptPath); err != nil {
		delete(w.files, transcriptPath)
		return err
	}

	logger.Log("transcript track %s:%d → %s (initial=%s)", session, winIndex, transcriptPath, state)

	// Send initial state
	w.Updates <- StateUpdate{
		Session:   session,
		WinIndex:  winIndex,
		AgentName: agentName,
		State:     state,
	}

	return nil
}

// Untrack stops watching a transcript file.
func (w *Watcher) Untrack(transcriptPath string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, ok := w.files[transcriptPath]; !ok {
		return
	}
	_ = w.fsw.Remove(transcriptPath)
	delete(w.files, transcriptPath)
	logger.Log("transcript untrack %s", transcriptPath)
}

// Run processes fsnotify events until the context is cancelled.
func (w *Watcher) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) {
				w.handleWrite(event.Name)
			}
		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			logger.Log("transcript fsnotify error: %v", err)
		}
	}
}

func (w *Watcher) handleWrite(path string) {
	w.mu.Lock()
	t, ok := w.files[path]
	if !ok {
		w.mu.Unlock()
		return
	}
	session := t.Session
	winIndex := t.WinIndex
	agentName := t.AgentName
	lastOffset := t.LastOffset
	lastState := t.LastState
	w.mu.Unlock()

	// Read only new data since last offset
	// #nosec G304 -- path is an already tracked local transcript.
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	stat, err := f.Stat()
	if err != nil {
		return
	}
	currentSize := stat.Size()

	if currentSize <= lastOffset {
		// File was truncated or unchanged
		if currentSize < lastOffset {
			lastOffset = 0
		} else {
			return
		}
	}

	_, _ = f.Seek(lastOffset, io.SeekStart)
	data, err := io.ReadAll(f)
	if err != nil {
		return
	}

	// Update offset
	w.mu.Lock()
	if t2, ok := w.files[path]; ok {
		t2.LastOffset = currentSize
	}
	w.mu.Unlock()

	// Parse new lines backward to find the last state-relevant one
	lines := strings.Split(string(data), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		var rl rawLine
		if err := json.Unmarshal([]byte(line), &rl); err != nil {
			continue
		}
		if !rl.stateRelevant() {
			continue
		}
		state := DeriveState(rl)
		if state == detect.StateUnknown {
			continue
		}
		if state != lastState {
			logger.Log("transcript %s:%d state %s → %s", session, winIndex, lastState, state)
			w.mu.Lock()
			if t2, ok := w.files[path]; ok {
				t2.LastState = state
			}
			w.mu.Unlock()

			w.Updates <- StateUpdate{
				Session:   session,
				WinIndex:  winIndex,
				AgentName: agentName,
				State:     state,
			}
		}
		return
	}
}

// Close shuts down the watcher.
func (w *Watcher) Close() {
	_ = w.fsw.Close()
	close(w.Updates)
}
