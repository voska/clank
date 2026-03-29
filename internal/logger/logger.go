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

func Init(path string) error {
	mu.Lock()
	defer mu.Unlock()
	var err error
	file, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	return err
}

func Close() {
	mu.Lock()
	defer mu.Unlock()
	if file != nil {
		file.Close()
	}
}

func Log(format string, args ...any) {
	mu.Lock()
	defer mu.Unlock()
	if file == nil {
		return
	}
	ts := time.Now().Format("15:04:05.000")
	fmt.Fprintf(file, "%s  %s\n", ts, fmt.Sprintf(format, args...))
}
