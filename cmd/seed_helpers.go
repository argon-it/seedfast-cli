// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package cmd provides CLI commands for the Seedfast database seeding tool.
// This file contains helper functions for UI management during the seeding process.
package cmd

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"atomicgo.dev/cursor"
	"github.com/pterm/pterm"
)

// startHeaderSpinner starts a header spinner animation in the terminal.
// It hides the cursor, creates an area for the spinner, and starts a goroutine
// that updates the spinner animation at regular intervals using the provided frames.
//
// The spinner displays a "Seeding" message with rotating animation frames and
// runs until the stop channel is closed.
func startHeaderSpinner(headerAreaPtr **pterm.AreaPrinter, frames []string, headerIdxPtr *int, stop chan struct{}, wgPtr *sync.WaitGroup) {
	cursor.Hide()
	area, aerr := pterm.DefaultArea.WithRemoveWhenDone(true).Start()
	if aerr != nil {
		cursor.Show()
		return
	}
	*headerAreaPtr = area
	wgPtr.Add(1)
	go func() {
		defer wgPtr.Done()
		t := time.NewTicker(120 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				*headerIdxPtr++
				area.Update(fmt.Sprintf("%s Seeding", frames[*headerIdxPtr%len(frames)]))
			case <-stop:
				return
			}
		}
	}()
}

// stopHeaderSpinner stops the header spinner animation and cleans up resources.
// It closes the stop channel, waits for the spinner goroutine to finish, stops
// the area display, and shows the cursor again.
func stopHeaderSpinner(headerAreaPtr **pterm.AreaPrinter, stopPtr *chan struct{}, wgPtr *sync.WaitGroup) {
	close(*stopPtr)
	wgPtr.Wait()
	if *headerAreaPtr != nil {
		(*headerAreaPtr).Stop()
		*headerAreaPtr = nil
	}
	*stopPtr = make(chan struct{})
	cursor.Show()
}

// startAreaSpinner starts an area spinner for displaying progress information.
// It creates an area display, hides the cursor, and starts a goroutine that
// periodically updates the display using the provided update function.
//
// The spinner runs until the stop channel is closed and calls the update
// function on each animation frame to refresh the displayed content.
func startAreaSpinner(areaPtr **pterm.AreaPrinter, wgPtr *sync.WaitGroup, stop chan struct{}, frameIdxPtr *int, update func()) {
	cursor.Hide()
	area, _ := pterm.DefaultArea.WithRemoveWhenDone(true).Start()
	*areaPtr = area
	wgPtr.Add(1)
	go func() {
		defer wgPtr.Done()
		t := time.NewTicker(120 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				*frameIdxPtr++
				update()
			case <-stop:
				return
			}
		}
	}()
}

// stopAreaSpinner stops the area spinner animation and cleans up resources.
// It closes the stop channel, waits for the spinner goroutine to finish,
// stops the area display, and shows the cursor again.
func stopAreaSpinner(areaPtr **pterm.AreaPrinter, wgPtr *sync.WaitGroup, stopPtr *chan struct{}) {
	close(*stopPtr)
	wgPtr.Wait()
	if *areaPtr != nil {
		(*areaPtr).Stop()
		*areaPtr = nil
	}
	*stopPtr = make(chan struct{})
	cursor.Show()
}

// extractTablesAndPreviewFlexible parses tables and preview text from a flexible JSON payload.
// It supports several common payload shapes to be resilient to backend API changes.
// The function attempts to extract table names and preview text from various JSON structures,
// handling different field names and nested structures gracefully.
//
// It returns a slice of unique table names and a preview string, or empty values if parsing fails.
func extractTablesAndPreviewFlexible(jsonStr string) ([]string, string) {
	var top map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &top); err != nil {
		return nil, ""
	}
	collectTables := func(node any) (out []string) {
		m, ok := node.(map[string]any)
		if !ok {
			return nil
		}
		if v, ok := m["tables"]; ok {
			if arr, ok := v.([]any); ok {
				for _, e := range arr {
					if s, ok := e.(string); ok && s != "" {
						out = append(out, s)
					}
				}
			}
		}
		return out
	}
	var tables []string
	tables = append(tables, collectTables(top)...)
	for _, key := range []string{"plan", "scope", "data"} {
		if v, ok := top[key]; ok {
			tables = append(tables, collectTables(v)...)
		}
	}
	if len(tables) > 1 {
		seen := make(map[string]struct{})
		var uniq []string
		for _, t := range tables {
			if _, ok := seen[t]; ok {
				continue
			}
			seen[t] = struct{}{}
			uniq = append(uniq, t)
		}
		tables = uniq
	}
	preview := ""
	if v, ok := top["preview"]; ok {
		if s, ok2 := v.(string); ok2 {
			preview = s
		}
	}
	if preview == "" {
		if v, ok := top["text"]; ok {
			if s, ok2 := v.(string); ok2 {
				preview = s
			}
		}
	}
	return tables, preview
}
