package ui

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"
)

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func (s *syncBuffer) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Len()
}

func TestSpinnerWritesBrailleFramesAndStopsCleanly(t *testing.T) {
	previous := enabled
	enabled = true
	t.Cleanup(func() { enabled = previous })

	var out syncBuffer
	sp := startSpinnerTo(&out, "thinking...", 5*time.Millisecond)

	deadline := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(deadline) {
		text := out.String()
		if strings.Contains(text, "thinking...") && strings.ContainsRune(text, spinnerFrames[0]) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	sp.Stop()
	lenAfterStop := out.Len()

	sp.Stop()
	var nilSpinner *Spinner
	nilSpinner.Stop()

	time.Sleep(25 * time.Millisecond)
	if got := out.Len(); got != lenAfterStop {
		t.Fatalf("spinner wrote %d bytes after Stop() returned", got-lenAfterStop)
	}

	rendered := out.String()
	if !strings.Contains(rendered, "thinking...") {
		t.Errorf("expected spinner output to include label, got %q", rendered)
	}
	if !strings.ContainsRune(rendered, spinnerFrames[0]) {
		t.Errorf("expected spinner output to include braille frame, got %q", rendered)
	}
	if !strings.HasSuffix(rendered, "\r\033[K") {
		t.Errorf("expected spinner output to end with clear line escape, got %q", rendered)
	}
}

func TestSpinnerNoOpWhenSuppressedOrColorsDisabled(t *testing.T) {
	prevEnabled := enabled
	prevSuppress := SuppressSpinner
	t.Cleanup(func() {
		enabled = prevEnabled
		SuppressSpinner = prevSuppress
	})

	SuppressSpinner = true
	enabled = true
	sp := StartSpinner("ignored")
	if sp.stop != nil {
		t.Errorf("expected no-op spinner when SuppressSpinner is true")
	}
	sp.Stop()

	SuppressSpinner = false
	enabled = false
	sp2 := StartSpinner("ignored")
	if sp2.stop != nil {
		t.Errorf("expected no-op spinner when colors/TTY are disabled")
	}
	sp2.Stop()
}
