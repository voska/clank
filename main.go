package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/voska/clank/internal/logger"
	"github.com/voska/clank/internal/socket"
	"github.com/voska/clank/internal/state"
	"github.com/voska/clank/internal/tmux"
	"github.com/voska/clank/internal/ui"
)

var version = "dev"

func main() {
	versionFlag := flag.Bool("version", false, "Print version")
	width := flag.Int("width", 30, "Sidebar width in columns")
	pollInterval := flag.Duration("poll", 1*time.Second, "tmux poll interval")
	scrapeInterval := flag.Duration("scrape", 2*time.Second, "Content scrape interval")
	socketPath := flag.String("socket", "/tmp/clank.sock", "Unix socket path")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("clank %s\n", version)
		os.Exit(0)
	}

	if err := logger.Init("/tmp/clank.log"); err != nil {
		log.Printf("clank: warning: could not open log: %v", err)
	}
	defer logger.Close()
	logger.Log("starting clank %s (width=%d poll=%s scrape=%s socket=%s)", version, *width, *pollInterval, *scrapeInterval, *socketPath)

	const statePath = "/tmp/clank-state.json"

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

	config := ui.Config{
		Width:          *width,
		PollInterval:   *pollInterval,
		ScrapeInterval: *scrapeInterval,
	}
	model := ui.NewModel(store, mux, events, config)

	// Save state periodically in background
	go func() {
		for {
			time.Sleep(5 * time.Second)
			if err := store.Save(statePath); err != nil {
				logger.Log("save error: %v", err)
			}
		}
	}()

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		log.Fatalf("clank: %v", err)
	}
	store.Save(statePath)
}
