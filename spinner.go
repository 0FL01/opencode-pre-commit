package main

import (
	"fmt"
	"io"
	"time"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func startSpinner(w io.Writer, msg string) func() {
	ticker := time.NewTicker(80 * time.Millisecond)
	return startSpinnerWithTicker(w, msg, ticker.C, ticker.Stop)
}

func startSpinnerWithTicker(w io.Writer, msg string, ticks <-chan time.Time, stopTicker func()) func() {
	done := make(chan struct{})
	go func() {
		i := 0
		for {
			select {
			case <-done:
				fmt.Fprintf(w, "\r\033[K")
				return
			case <-ticks:
				fmt.Fprintf(w, "\r%s %s", spinnerFrames[i%len(spinnerFrames)], msg)
				i++
			}
		}
	}()
	return func() {
		stopTicker()
		close(done)
	}
}
