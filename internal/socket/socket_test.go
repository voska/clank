package socket

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// tempSockPath returns a short socket path safe for macOS's 103-char limit.
// t.TempDir() uses the test name, which can exceed the limit on macOS.
func tempSockPath(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "ck*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Errorf("RemoveAll: %v", err)
		}
	})
	return filepath.Join(dir, "t.sock")
}

func TestListenerReceivesEvent(t *testing.T) {
	sockPath := tempSockPath(t)
	events := make(chan Event, 10)

	l, err := Listen(sockPath, events)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			t.Errorf("Close listener: %v", err)
		}
	}()

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	if _, err := fmt.Fprintln(conn, `{"session":"proj-a","window":"dev","state":"running","agent":"claude"}`); err != nil {
		t.Fatalf("Write event: %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("Close connection: %v", err)
	}

	select {
	case ev := <-events:
		if ev.Session != "proj-a" {
			t.Errorf("session: got %q, want %q", ev.Session, "proj-a")
		}
		if ev.Window != "dev" {
			t.Errorf("window: got %q, want %q", ev.Window, "dev")
		}
		if ev.State != "running" {
			t.Errorf("state: got %q, want %q", ev.State, "running")
		}
		if ev.Agent != "claude" {
			t.Errorf("agent: got %q, want %q", ev.Agent, "claude")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestListenerHandlesMultipleConnections(t *testing.T) {
	sockPath := tempSockPath(t)
	events := make(chan Event, 10)

	l, err := Listen(sockPath, events)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			t.Errorf("Close listener: %v", err)
		}
	}()

	for i := 0; i < 3; i++ {
		conn, err := net.Dial("unix", sockPath)
		if err != nil {
			t.Fatalf("Dial %d: %v", i, err)
		}
		if _, err := fmt.Fprintf(conn, `{"session":"s%d","window":"w","state":"running","agent":"claude"}`+"\n", i); err != nil {
			t.Fatalf("Write event: %v", err)
		}
		if err := conn.Close(); err != nil {
			t.Fatalf("Close connection: %v", err)
		}
	}

	for i := 0; i < 3; i++ {
		select {
		case <-events:
			// got event
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for event %d", i)
		}
	}
}

func TestListenerCleansUpSocket(t *testing.T) {
	sockPath := tempSockPath(t)
	events := make(chan Event, 10)

	l, err := Listen(sockPath, events)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Close listener: %v", err)
	}

	if _, err := os.Stat(sockPath); !os.IsNotExist(err) {
		t.Error("socket file should be removed after Close")
	}
}
