// Package terminal provides utilities for terminal operations such as clearing text.
package terminal

import (
	"fmt"
	"math"
	"os"

	"golang.org/x/term"
)

// ClearPreviousLines clears text from the terminal that was previously printed.
// It calculates how many lines were used by the provided text based on the current
// terminal width, then moves up and clears each line.
//
// This is useful for cleaning up user input prompts after they've been entered.
//
// Parameters:
//   - textLength: The total number of characters in the text to clear (prompt + user input)
//
// The function:
//  1. Gets the current terminal width (defaults to 80 if unavailable)
//  2. Calculates how many lines the text occupied
//  3. Moves up and clears each line using ANSI escape sequences
//  4. Adds +1 to account for the extra line created when user presses Enter
func ClearPreviousLines(textLength int) {
	// Get terminal width to calculate line wrapping
	termWidth := 80 // default fallback
	if width, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && width > 0 {
		termWidth = width
	}

	// Calculate total lines used by the text
	totalLines := int(math.Ceil(float64(textLength) / float64(termWidth)))
	if totalLines < 1 {
		totalLines = 1 // At minimum, we have 1 line
	}

	// After Enter, cursor is on a NEW line below the input.
	// Add +1 to clear the current empty line the cursor is on
	linesToClear := totalLines + 1

	// Move up and clear each line
	for i := 0; i < linesToClear; i++ {
		fmt.Print("\r\x1b[2K") // Move to start and clear entire line
		if i < linesToClear-1 {
			fmt.Print("\x1b[1A") // Move up one line (don't move up on last iteration)
		}
	}
}
