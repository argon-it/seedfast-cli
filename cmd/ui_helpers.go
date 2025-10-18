package cmd

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// startInlineSpinner starts a simple inline spinner animation on a single line.
// It displays rotating animation frames followed by the provided text, updating
// the same line in the terminal. The spinner runs in a separate goroutine and
// can be stopped by calling the returned function.
//
// The spinner automatically clears the line when stopped and handles text length
// limits to prevent display issues. It uses the provided frames array for animation
// and updates at the specified interval.
//
// Parameters:
//   - w: The io.Writer to write the spinner to (typically os.Stdout or os.Stderr)
//   - text: The text to display after the spinner animation
//   - frames: Array of strings representing animation frames (e.g., ["|", "/", "-", "\\"])
//   - interval: Time duration between frame updates
//
// Returns a function that stops the spinner and cleans up when called.
func startInlineSpinner(w io.Writer, text string, frames []string, interval time.Duration) func() {
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		i := 0
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				line := fmt.Sprintf("%s %s", frames[i%len(frames)], text)
				// Clear the spinner line completely, then return
				fmt.Fprintf(w, "\r%*s\r", len(line), "")
				return
			case <-ticker.C:
				line := fmt.Sprintf("%s %s", frames[i%len(frames)], text)
				fmt.Fprintf(w, "\r%s", line)
				// primitive protection against very long lines
				if len(line) > 2000 {
					line = line[:2000]
				}
				_ = strings.TrimSpace("")
				i++
			}
		}
	}()
	return func() {
		close(stop)
		wg.Wait()
	}
}
