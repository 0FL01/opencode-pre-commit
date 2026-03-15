package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestStartSpinnerWithTicker_WritesFrameAndMessage(t *testing.T) {
	var buf bytes.Buffer
	ticks := make(chan time.Time, 2)
	stopped := make(chan struct{})

	stop := startSpinnerWithTicker(&buf, "Loading...", ticks, func() { close(stopped) })

	ticks <- time.Now()
	ticks <- time.Now()

	// Give goroutine time to process ticks.
	time.Sleep(20 * time.Millisecond)

	stop()

	// Wait for ticker stop to be called.
	<-stopped

	// Give goroutine time to write clear sequence.
	time.Sleep(20 * time.Millisecond)

	output := buf.String()
	if !strings.Contains(output, "Loading...") {
		t.Errorf("output should contain message, got %q", output)
	}
	if !strings.Contains(output, spinnerFrames[0]) {
		t.Errorf("output should contain first spinner frame, got %q", output)
	}
}

func TestStartSpinnerWithTicker_ClearsOnStop(t *testing.T) {
	var buf bytes.Buffer
	ticks := make(chan time.Time, 1)
	stopped := false

	stop := startSpinnerWithTicker(&buf, "msg", ticks, func() { stopped = true })

	ticks <- time.Now()
	time.Sleep(20 * time.Millisecond)

	stop()
	time.Sleep(20 * time.Millisecond)

	if !stopped {
		t.Error("stopTicker should have been called")
	}

	output := buf.String()
	if !strings.Contains(output, "\r\033[K") {
		t.Errorf("output should contain clear sequence, got %q", output)
	}
}

func TestStartSpinnerWithTicker_MultipleFrames(t *testing.T) {
	var buf bytes.Buffer
	ticks := make(chan time.Time, 10)

	stop := startSpinnerWithTicker(&buf, "test", ticks, func() {})

	for i := 0; i < len(spinnerFrames)+1; i++ {
		ticks <- time.Now()
	}
	time.Sleep(20 * time.Millisecond)

	stop()
	time.Sleep(20 * time.Millisecond)

	output := buf.String()
	// After len(spinnerFrames)+1 ticks, the first frame should appear again (wrapping).
	count := strings.Count(output, spinnerFrames[0])
	if count < 2 {
		t.Errorf("expected first frame to appear at least twice (wrap around), appeared %d times", count)
	}
}

func TestStartSpinner_ReturnsStopFunc(t *testing.T) {
	var buf bytes.Buffer
	stop := startSpinner(&buf, "test")
	// Just verify stop doesn't panic.
	time.Sleep(10 * time.Millisecond)
	stop()
}
