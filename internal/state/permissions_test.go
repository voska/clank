package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveFilePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := New().Save(path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode&0o077 != 0 {
		t.Fatalf("state permissions = %o, want no group or other access", mode)
	}
}
