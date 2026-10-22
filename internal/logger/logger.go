// Package logger writes best-effort diagnostic logs for clank.
package logger

import (
	"fmt"
	"os"
	"sync"
	"time"
)

var (
	mu   sync.Mutex
	file *os.File
)

// Init opens the diagnostic log at path.
func Init(path string) error {
	mu.Lock()
	defer mu.Unlock()
	var err error
	// #nosec G304 -- path is the configured diagnostic log destination.
	file, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	return err
}

// Close releases the log file.
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if file != nil {
		_ = file.Close()
	}
}

const maxLogSize = 100 * 1024 // 100KB

// Log appends a formatted diagnostic message.
func Log(format string, args ...any) {
	mu.Lock()
	defer mu.Unlock()
	if file == nil {
		return
	}

	// Rotate if too large
	if info, err := file.Stat(); err == nil && info.Size() > maxLogSize {
		_ = file.Truncate(0)
		_, _ = file.Seek(0, 0)
		_, _ = fmt.Fprintf(file, "%s  log rotated (exceeded %dKB)\n", time.Now().Format("15:04:05.000"), maxLogSize/1024)
	}

	ts := time.Now().Format("15:04:05.000")
	_, _ = fmt.Fprintf(file, "%s  %s\n", ts, fmt.Sprintf(format, args...))
}
