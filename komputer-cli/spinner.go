package main

import (
	"fmt"
	"sync"
	"time"
)

// ─── Spinner ────────────────────────────────────────────────────────────────

type Spinner struct {
	mu      sync.Mutex
	msg     string
	stop    chan struct{}
	stopped chan struct{}
}

func NewSpinner(msg string) *Spinner {
	s := &Spinner{
		msg:     msg,
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
	go s.run()
	return s
}

func (s *Spinner) run() {
	defer close(s.stopped)
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.stop:
			// Clear the spinner line.
			fmt.Printf("\r\033[K")
			return
		case <-ticker.C:
			s.mu.Lock()
			msg := s.msg
			s.mu.Unlock()
			frame := busyStyle.Render(frames[i%len(frames)])
			fmt.Printf("\r\033[K%s %s", frame, dimStyle.Render(msg))
			i++
		}
	}
}

func (s *Spinner) SetMessage(msg string) {
	s.mu.Lock()
	s.msg = msg
	s.mu.Unlock()
}

func (s *Spinner) Stop() {
	select {
	case <-s.stop:
		return // already stopped
	default:
		close(s.stop)
		<-s.stopped
	}
}
