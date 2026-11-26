package ui

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

var spinnerFrames = []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")

var SuppressSpinner bool

type Spinner struct {
	stop chan struct{}
	done chan struct{}
	once sync.Once
}

func StartSpinner(label string) *Spinner {
	if SuppressSpinner || !enabled || TermWidth() == 0 {
		return &Spinner{}
	}
	return startSpinnerTo(os.Stdout, label, 80*time.Millisecond)
}

func startSpinnerTo(w io.Writer, label string, interval time.Duration) *Spinner {
	s := &Spinner{
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		i := 0
		fmt.Fprintf(w, "\r%s %s\033[K", Cyan(string(spinnerFrames[i])), Dim(label))
		i = (i + 1) % len(spinnerFrames)

		for {
			select {
			case <-s.stop:
				fmt.Fprint(w, "\r\033[K")
				return
			case <-ticker.C:
				fmt.Fprintf(w, "\r%s %s\033[K", Cyan(string(spinnerFrames[i])), Dim(label))
				i = (i + 1) % len(spinnerFrames)
			}
		}
	}()
	return s
}

func (s *Spinner) Stop() {
	if s == nil || s.stop == nil {
		return
	}
	s.once.Do(func() { close(s.stop) })
	<-s.done
}

func FormatModelSwitch(role, model string) string {
	return "\n" + wrap(RoleColor(role), "● "+role) + Dim(" · "+model)
}

func ModelSwitch(role, model string) {
	fmt.Println(FormatModelSwitch(role, model))
}
