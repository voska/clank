package state

import "fmt"

// winKey creates a unique key for a window within a session using the
// window index (not name), since multiple windows can share a name.
func winKey(session string, windowIndex int) string {
	return fmt.Sprintf("%s:%d", session, windowIndex)
}
