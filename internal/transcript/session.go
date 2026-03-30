package transcript

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// SessionInfo represents a Claude Code session file.
type SessionInfo struct {
	PID       int    `json:"pid"`
	SessionID string `json:"sessionId"`
	CWD       string `json:"cwd"`
}

// LoadAllSessions reads all session files from ~/.claude/sessions/.
func LoadAllSessions() (map[int]SessionInfo, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(home, ".claude", "sessions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	sessions := make(map[int]SessionInfo)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var info SessionInfo
		if err := json.Unmarshal(data, &info); err != nil {
			continue
		}
		sessions[info.PID] = info
	}
	return sessions, nil
}

// TranscriptPath returns the JSONL transcript file path for a session.
func TranscriptPath(info SessionInfo) string {
	home, _ := os.UserHomeDir()
	encoded := encodeCWD(info.CWD)
	return filepath.Join(home, ".claude", "projects", encoded, info.SessionID+".jsonl")
}

// encodeCWD encodes a CWD path the way Claude Code does:
// /Users/matt/Developer/homelab → -Users-matt-Developer-homelab
func encodeCWD(cwd string) string {
	return strings.ReplaceAll(cwd, "/", "-")
}

// MapPanesToTranscripts maps tmux pane PIDs to their Claude Code transcript paths.
// It reads all session files, finds which claude PIDs are children of which pane PIDs,
// and returns the mapping.
func MapPanesToTranscripts(panePIDs []int) map[int]string {
	sessions, err := LoadAllSessions()
	if err != nil {
		return nil
	}

	// Build a map of claude PID → parent PID
	claudeParents := make(map[int]int) // claude PID → parent PID
	for claudePID := range sessions {
		ppid := getParentPID(claudePID)
		if ppid > 0 {
			claudeParents[claudePID] = ppid
		}
	}

	// Build set of pane PIDs for fast lookup
	paneSet := make(map[int]bool)
	for _, pid := range panePIDs {
		paneSet[pid] = true
	}

	// Match: if a claude process's parent is a known pane PID, we have a match
	result := make(map[int]string) // pane PID → transcript path
	for claudePID, parentPID := range claudeParents {
		if paneSet[parentPID] {
			info := sessions[claudePID]
			path := TranscriptPath(info)
			if _, err := os.Stat(path); err == nil {
				result[parentPID] = path
			}
		}
	}

	return result
}

// getParentPID returns the parent PID of a process.
func getParentPID(pid int) int {
	out, err := exec.Command("ps", "-o", "ppid=", "-p", fmt.Sprintf("%d", pid)).Output()
	if err != nil {
		return 0
	}
	ppid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0
	}
	return ppid
}
