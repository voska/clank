package socket

import (
	"bufio"
	"encoding/json"
	"net"
	"os"

	"github.com/voska/clank/internal/logger"
)

type Event struct {
	Session    string  `json:"session"`
	Window     string  `json:"window"`
	State      string  `json:"state"`
	Reason     string  `json:"reason,omitempty"`
	Agent      string  `json:"agent,omitempty"`
	Model      string  `json:"model,omitempty"`
	ContextPct float64 `json:"context_pct,omitempty"`
}

type Listener struct {
	ln       net.Listener
	sockPath string
}

func Listen(sockPath string, events chan<- Event) (*Listener, error) {
	os.Remove(sockPath)

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		return nil, err
	}

	l := &Listener{ln: ln, sockPath: sockPath}
	go l.accept(events)
	return l, nil
}

func (l *Listener) accept(events chan<- Event) {
	for {
		conn, err := l.ln.Accept()
		if err != nil {
			return
		}
		go l.handle(conn, events)
	}
}

func (l *Listener) handle(conn net.Conn, events chan<- Event) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		var ev Event
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			logger.Log("socket: invalid JSON: %v", err)
			continue
		}
		events <- ev
	}
}

func (l *Listener) Close() error {
	err := l.ln.Close()
	os.Remove(l.sockPath)
	return err
}
