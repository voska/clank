package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/voska/clank/internal/logger"
	"github.com/voska/clank/internal/socket"
	"github.com/voska/clank/internal/state"
	"github.com/voska/clank/internal/tmux"
	"github.com/voska/clank/internal/transcript"
	"github.com/voska/clank/internal/ui"
)

var version = "dev"

func stateDir() string {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, "clank")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "clank")
}

func runtimeDir() string {
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return dir
	}
	return os.TempDir()
}

func main() {
	versionFlag := flag.Bool("version", false, "Print version")
	width := flag.Int("width", 30, "Sidebar width in columns")
	pollInterval := flag.Duration("poll", 1*time.Second, "tmux poll interval")
	scrapeInterval := flag.Duration("scrape", 2*time.Second, "Content scrape interval")
	defaultSocket := filepath.Join(runtimeDir(), fmt.Sprintf("clank-%d.sock", os.Getuid()))
	socketPath := flag.String("socket", defaultSocket, "Unix socket path")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("clank %s\n", version)
		os.Exit(0)
	}

	sd := stateDir()
	os.MkdirAll(sd, 0700)

	logPath := filepath.Join(sd, "clank.log")
	if err := logger.Init(logPath); err != nil {
		log.Printf("clank: warning: could not open log: %v", err)
	}
	defer logger.Close()

	statePath := filepath.Join(sd, "state.json")
	logger.Log("starting clank %s (width=%d poll=%s scrape=%s socket=%s state=%s)", version, *width, *pollInterval, *scrapeInterval, *socketPath, statePath)

	store := state.New()
	if err := store.Load(statePath); err == nil {
		logger.Log("restored state from %s", statePath)
	}
	mux := &tmux.Live{}

	events := make(chan socket.Event, 100)
	listener, err := socket.Listen(*socketPath, events)
	if err != nil {
		log.Fatalf("clank: failed to start socket listener: %v", err)
	}
	defer listener.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var tw *transcript.Watcher
	tw, err = transcript.NewWatcher()
	if err != nil {
		logger.Log("warning: transcript watcher disabled: %v", err)
	} else {
		go tw.Run(ctx)
		defer tw.Close()
		logger.Log("transcript watcher enabled")
	}

	config := ui.Config{
		Width:          *width,
		PollInterval:   *pollInterval,
		ScrapeInterval: *scrapeInterval,
	}
	model := ui.NewModel(store, mux, events, config, tw)

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := store.Save(statePath); err != nil {
					logger.Log("save error: %v", err)
				}
			}
		}
	}()

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		log.Fatalf("clank: %v", err)
	}
	cancel()
	store.Save(statePath)
}
