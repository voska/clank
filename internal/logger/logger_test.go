package logger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLogFilePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clank.log")
	if err := Init(path); err != nil {
		t.Fatal(err)
	}
	defer Close()
	Log("test message")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode&0o077 != 0 {
		t.Fatalf("log permissions = %o, want no group or other access", mode)
	}
}
