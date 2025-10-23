// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"seedfast/cli/internal/auth"
	bbridge "seedfast/cli/internal/bridge"
	"seedfast/cli/internal/bridge/model"
	"seedfast/cli/internal/dsn"
	"seedfast/cli/internal/keychain"
	"seedfast/cli/internal/logging"
	"seedfast/cli/internal/manifest"
	"seedfast/cli/internal/seeding"
	"seedfast/cli/internal/sqlexec"

	"atomicgo.dev/cursor"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	verboseSeed bool
)

// seedCmd represents the seed command for executing database seeding operations.
// It connects to the backend via gRPC bridge and orchestrates the seeding process,
// handling task distribution, progress tracking, and user interactions for database seeding.
var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Execute database seeding via gRPC bridge",
	Long: `The seed command initiates the database seeding process by connecting to the backend
service via gRPC bridge. It handles task distribution to worker pools, tracks progress,
manages user interactions for decision-making, and provides real-time feedback during
the seeding operation.

The command supports interactive seeding with progress indicators and can handle
connection interruptions gracefully.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		// Enable verbose mode for all modules if --verbose is set
		if verboseSeed {
			os.Setenv("SEEDFAST_VERBOSE", "1")
		}

		st, err := auth.Load()
		if err != nil || !st.LoggedIn {
			if verboseSeed {
				fmt.Printf("[DEBUG] seed: auth.Load() error or not logged in - err: %v, LoggedIn: %v\n", err, st.LoggedIn)
			}
			fmt.Println("⚠️  You need to be logged in to start seeding.")
			fmt.Println("   Please run: seedfast login")
			return nil
		}

		startAt := time.Now()
		// File logging disabled - events are no longer written to seed_events.log
		logf := func(format string, args ...any) {
			// No-op: logging disabled
		}

		// Fetch manifest from server
		m, err := manifest.GetEndpoints(cmd.Context())
		if err != nil {
			return err
		}

		br := bbridge.New()

		// Pre-seed check: resolve DSN from env or keychain (not from config)
		rawDSN := ""
		if env := os.Getenv("SEEDFAST_DSN"); strings.TrimSpace(env) != "" {
			rawDSN = strings.TrimSpace(env)
		} else if env := os.Getenv("DATABASE_URL"); strings.TrimSpace(env) != "" {
			rawDSN = strings.TrimSpace(env)
		}
		if strings.TrimSpace(rawDSN) == "" {
			if km, err := keychain.GetManager(); err == nil {
				if v, err := km.LoadDBDSN(); err == nil && strings.TrimSpace(v) != "" {
					rawDSN = strings.TrimSpace(v)
				}
			}
		}
		if strings.TrimSpace(rawDSN) == "" {
			fmt.Println("⚠️  No database connection configured.")
			fmt.Println("   Please run 'seedfast connect' to configure your database,")
			return nil
		}

		// Parse and normalize the DSN to handle special characters
		normalizedDSN, err := dsn.Parse(rawDSN)
		if err != nil {
			fmt.Println("❌ Invalid database connection string.")
			if parseErr, ok := err.(*dsn.ParseError); ok {
				fmt.Println("   " + parseErr.Error())
			}
			fmt.Println("   Please run 'seedfast connect' to reconfigure your database.")
			return err
		}

		// Display database connection info (masked)
		maskedDSN := logging.Mask(normalizedDSN)
		dbName := deriveDBName(normalizedDSN)
		pterm.Println()
		pterm.Println(pterm.NewStyle(pterm.FgLightCyan).Sprint("→ Database:   ") + pterm.NewStyle(pterm.FgCyan, pterm.Bold).Sprint(dbName))
		pterm.Println(pterm.NewStyle(pterm.FgLightCyan).Sprint("→ Connection: ") + pterm.NewStyle(pterm.FgLightBlue).Sprint(maskedDSN))
		pterm.Println()

		// Use gRPC address from manifest (no fallback)
		addr := m.GRPCAddress()

		// Validate access token and resolve user before connecting
		token := ""
		if km, err := keychain.GetManager(); err == nil {
			if t, err := km.LoadAccessToken(); err == nil {
				token = t
			}
		}
		if token == "" {
			return errors.New("not logged in; run 'seedfast login' first")
		}
		svc := auth.NewService(m.HTTPBaseURL(), m.HTTP)
		if _, ok, _ := svc.WhoAmI(cmd.Context()); !ok {
			return errors.New("session invalid or expired; run 'seedfast login' again")
		}
		if err := br.Connect(cmd.Context(), addr, token); err != nil {
			pterm.Printf("❌ Failed to connect to Seedfast service\n")
			pterm.Println(logging.PresentError("", err))
			return err
		}
		// Ensure bridge is closed and token is cleared from memory when seeding finishes
		defer func() {
			_ = br.Close(cmd.Context())
		}()
		if err := br.Init(cmd.Context(), "", dbName); err != nil {
			pterm.Printf("❌ Failed to initialize seeding session\n")
			pterm.Println(logging.PresentError("", err))
			return err
		}

		// Start the Seeding header spinner (stick style). It will be removed on questions.
		render := seeding.NewRenderer()
		render.ShowSection()
		var headerArea *pterm.AreaPrinter
		headerFrames := []string{"|", "/", "-", "\\"}
		headerIdx := 0
		headerSpinStop := make(chan struct{})
		var headerSpinWG sync.WaitGroup
		headerStarted := false
		startHeader := func() {
			if headerStarted {
				return
			}
			var err error
			cursor.Hide()
			headerArea, err = pterm.DefaultArea.WithRemoveWhenDone(true).Start()
			if err != nil {
				cursor.Show()
				return
			}
			headerStarted = true
			headerSpinWG.Add(1)
			go func() {
				defer headerSpinWG.Done()
				t := time.NewTicker(120 * time.Millisecond)
				defer t.Stop()
				for {
					select {
					case <-t.C:
						headerIdx++
						headerArea.Update(fmt.Sprintf("%s Seeding", headerFrames[headerIdx%len(headerFrames)]))
					case <-headerSpinStop:
						return
					}
				}
			}()
		}
		stopHeader := func() {
			if !headerStarted {
				return
			}
			close(headerSpinStop)
			headerSpinWG.Wait()
			if headerArea != nil {
				headerArea.Stop()
				headerArea = nil
			}
			headerSpinStop = make(chan struct{})
			headerStarted = false
			cursor.Show()
		}
		startHeader()

		// Open DB pool silently; avoid noisy spinners
		pool, err := pgxpool.New(cmd.Context(), normalizedDSN)
		if err != nil {
			pterm.Printf("❌ Failed to connect to database\n")
			pterm.Println(logging.PresentError("", err))
			return err
		}
		defer pool.Close()
		exec := sqlexec.New(pool)

		doneEvents := make(chan struct{})
		scopeShown := false
		var doneTables []string
		var area *pterm.AreaPrinter
		// Use braille spinner frames similar to docker CLI
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		frameIdx := 0
		active := map[string]int{}
		completed := map[string]struct{}{}
		failed := map[string]string{}
		var order []string
		var widthMutex sync.Mutex
		var maxLineLen int
		var lastRendered string
		var planSpinner *pterm.SpinnerPrinter
		planningActive := false
		awaitingDecision := false
		var spinWG sync.WaitGroup
		spinStop := make(chan struct{})
		var streamErr error
		var streamClosed bool
		var earlyNotified bool
		var workflowCompleted bool
		var seedingFailed bool
		// Track expected tables (from plan) to distinguish full completion vs early close
		expectedTables := map[string]struct{}{}
		expectedCount := 0

		resetAreaState := func() {
			active = map[string]int{}
			completed = map[string]struct{}{}
			failed = map[string]string{}
			order = nil
			expectedTables = map[string]struct{}{}
			expectedCount = 0
			widthMutex.Lock()
			maxLineLen = 0
			lastRendered = ""
			widthMutex.Unlock()
		}

		// extractTablesAndPreview attempts to parse tables and preview text from a
		// flexible JSON payload. It supports several common shapes to be resilient
		// to backend changes.
		extractTablesAndPreview := func(jsonStr string) ([]string, string) {
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

		updateArea := func() {
			if area == nil {
				return
			}
			widthMutex.Lock()
			defer widthMutex.Unlock()
			lines := make([]string, 0, len(order))
			localMax := 0
			for _, name := range order {
				if rem, ok := active[name]; ok {
					spin := frames[frameIdx%len(frames)]
					line := spin + " seeding " + name
					if rem > 0 {
						line += " (remaining " + fmt.Sprint(rem) + ")"
					}
					if l := utf8.RuneCountInString(line); l > localMax {
						localMax = l
					}
					lines = append(lines, line)
					continue
				}
				if _, ok := completed[name]; ok {
					line := "✓ seeded " + name
					if l := utf8.RuneCountInString(line); l > localMax {
						localMax = l
					}
					lines = append(lines, line)
					continue
				}
				if _, ok := failed[name]; ok {
					line := "✗ failed " + name
					if l := utf8.RuneCountInString(line); l > localMax {
						localMax = l
					}
					lines = append(lines, line)
				}
			}
			if localMax > maxLineLen {
				maxLineLen = localMax
			}
			for i := range lines {
				if pad := maxLineLen - utf8.RuneCountInString(lines[i]); pad > 0 {
					lines[i] = lines[i] + strings.Repeat(" ", pad)
				}
			}
			text := strings.Join(lines, "\n")
			if text == lastRendered {
				return
			}
			lastRendered = text
			area.Update(text)
		}
		startArea := func() {
			if area != nil {
				return
			}
			cursor.Hide()
			area, _ = pterm.DefaultArea.WithRemoveWhenDone(true).Start()
			spinWG.Add(1)
			go func() {
				defer spinWG.Done()
				t := time.NewTicker(120 * time.Millisecond)
				defer t.Stop()
				for {
					select {
					case <-t.C:
						frameIdx++
						updateArea()
					case <-spinStop:
						return
					}
				}
			}()
		}
		stopArea := func() {
			if area == nil {
				return
			}
			close(spinStop)
			spinWG.Wait()
			area.Stop()
			area = nil
			spinStop = make(chan struct{})
			cursor.Show()
		}

		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()

		go func() {
			for ev := range br.Events() {
				logf("event type=%s payload_len=%d", ev.Type, len(ev.Message))
				// Handle transport lifecycle events raised by gRPC client
				if string(ev.Type) == "stream_error" {
					streamErr = errors.New(ev.Message)
					// Stop UI immediately and notify
					// no prep spinner in the new UI
					if planningActive && planSpinner != nil {
						planSpinner.Stop()
						planningActive = false
					}
					stopArea()
					// Display user-friendly error message
					logging.PresentStreamError(ev.Message)
					// Cancel workers to expedite shutdown
					cancel()
					earlyNotified = true
					break
				}
				if string(ev.Type) == "stream_closed" {
					// Normal close; decide whether to warn based on completion of all expected tables
					streamClosed = true
					logf("stream_closed received")
					// no prep spinner in the new UI
					if planningActive && planSpinner != nil {
						planSpinner.Stop()
						planningActive = false
					}
					stopArea()
					// Do not print here; final summary will decide success/warning
					cancel()
					break
				}
				// Explicit workflow completion signal from server
				if string(ev.Type) == "workflow_completed" {
					workflowCompleted = true
					logf("workflow_completed received")
					// Mark any remaining active tables as completed to avoid lingering spinners
					for name := range active {
						delete(active, name)
						completed[name] = struct{}{}
						if _, exists := expectedTables[name]; !exists {
							expectedTables[name] = struct{}{}
							expectedCount = len(expectedTables)
						}
					}
					updateArea()
					// Stop UI and close the bridge; proceed to final summary without waiting for stream close
					stopArea()
					_ = br.Close(cmd.Context())
					cancel()
					break
				}
				if string(ev.Type) == "plan_proposed" {
					var payload struct {
						Preview string   `json:"preview"`
						Tables  []string `json:"tables"`
					}
					if err := json.Unmarshal([]byte(ev.Message), &payload); err == nil {
						// Remove header spinner before showing plan/questions
						stopHeader()
						// Replan/reset UI
						if planningActive && planSpinner != nil {
							planSpinner.Stop()
							planningActive = false
						}
						stopArea()
						resetAreaState()
						// Capture expected tables from plan (prefer exact), otherwise try flexible parsing
						candidateTables := payload.Tables
						candidatePreview := payload.Preview
						if len(candidateTables) == 0 {
							if t, p := extractTablesAndPreview(ev.Message); len(t) > 0 {
								candidateTables, candidatePreview = t, p
							}
						}
						if len(candidateTables) > 0 {
							for _, t := range candidateTables {
								expectedTables[t] = struct{}{}
							}
							expectedCount = len(expectedTables)
						}
						awaitingDecision = true
						pterm.Println(pterm.NewStyle(pterm.FgLightCyan, pterm.Bold).Sprint("Proposed seeding scope"))
						if candidatePreview != "" {
							pterm.Println(candidatePreview)
							scopeShown = true
						} else if len(candidateTables) > 0 {
							items := func(items []string) []pterm.BulletListItem {
								var out []pterm.BulletListItem
								for _, s := range items {
									out = append(out, pterm.BulletListItem{Level: 0, Text: s})
								}
								return out
							}(candidateTables)
							_ = pterm.DefaultBulletList.WithItems(items).Render()
							scopeShown = true
						}
					}
					continue
				}

				if string(ev.Type) == "ask_human" {
					var payload struct {
						QuestionID string `json:"question_id"`
						Question   string `json:"question"`
						Context    struct {
							Tables []string `json:"tables"`
						} `json:"context"`
					}
					if err := json.Unmarshal([]byte(ev.Message), &payload); err == nil {
						// Remove the header spinner line entirely when a question arrives
						stopHeader()
						// If we were already planning, stop that spinner before showing a new question
						if planningActive && planSpinner != nil {
							planSpinner.Stop()
							planningActive = false
						}
						// Capture expected tables if provided in context (fallback)
						if !scopeShown && len(payload.Context.Tables) > 0 {
							for _, t := range payload.Context.Tables {
								expectedTables[t] = struct{}{}
							}
							expectedCount = len(expectedTables)
						}
						if !scopeShown && len(payload.Context.Tables) > 0 {
							pterm.Println(pterm.NewStyle(pterm.FgLightCyan, pterm.Bold).Sprint("Proposed seeding scope"))
							items := func(items []string) []pterm.BulletListItem {
								var out []pterm.BulletListItem
								for _, s := range items {
									out = append(out, pterm.BulletListItem{Level: 0, Text: s})
								}
								return out
							}(payload.Context.Tables)
							_ = pterm.DefaultBulletList.WithItems(items).Render()
							scopeShown = true
						}
						prompt := payload.Question
						pterm.Println()

						// Show custom question if provided by backend
						if strings.TrimSpace(prompt) != "" {
							pterm.Println(pterm.NewStyle(pterm.FgYellow, pterm.Bold).Sprint(prompt))
						} else {
							pterm.Println(pterm.NewStyle(pterm.FgYellow, pterm.Bold).Sprint("Do you agree with this seeding scope?"))
						}

						// Always show options
						pterm.Println("  • Press " + pterm.NewStyle(pterm.FgGreen).Sprint("Enter") + " or type " + pterm.NewStyle(pterm.FgGreen).Sprint("yes") + " to accept and continue")
						pterm.Println("  • Type " + pterm.NewStyle(pterm.FgRed).Sprint("no") + " to reject")
						pterm.Println("  • Or provide detailed feedback/instructions to refine the scope")
						pterm.Println()
						pterm.Print("Your answer: ")
						reader := bufio.NewReader(os.Stdin)
						ans, _ := reader.ReadString('\n')
						ans = strings.TrimSpace(ans)
						var respObj map[string]any
						if ans == "" {
							pterm.Info.Println("Empty input interpreted as acceptance. Continuing with the proposed scope.")
							respObj = map[string]any{
								"human_answer": true,
								"question_id":  payload.QuestionID,
								"answer":       map[string]any{"raw": ""},
							}
						} else {
							respObj = map[string]any{
								"human_answer": false,
								"question_id":  payload.QuestionID,
								"answer":       map[string]any{"raw": ans},
							}
						}
						if b, err := json.Marshal(respObj); err == nil {
							_ = br.SendSQLResponse(cmd.Context(), model.SQLResponse{RequestID: payload.QuestionID, Success: true, ResultJSON: string(b)})
							if ans != "" && !planningActive {
								// Print a newline instead of a spinner to keep UI clean and avoid flicker
								pterm.Println("")
								planningActive = true
							}
						}
					}
					continue
				}

				// session_ready informational event (ignored)
				if string(ev.Type) == "session_ready" {
					continue
				}

				if string(ev.Type) == "table_started" {
					var p struct {
						Name      string `json:"name"`
						Remaining int    `json:"remaining"`
					}
					if err := json.Unmarshal([]byte(ev.Message), &p); err == nil {
						// Ensure header spinner is not competing with the per-table area
						stopHeader()
						if awaitingDecision {
							awaitingDecision = false
							if planningActive && planSpinner != nil {
								planSpinner.Stop()
								planningActive = false
							}
						}
						if _, ok := active[p.Name]; !ok {
							order = append(order, p.Name)
						}
						active[p.Name] = p.Remaining
						// If expected plan was not provided, infer expected set from started tables
						if _, ok := expectedTables[p.Name]; !ok {
							expectedTables[p.Name] = struct{}{}
							expectedCount = len(expectedTables)
						}
						startArea()
						updateArea()
					}
					continue
				}
				if string(ev.Type) == "table_done" {
					var p struct {
						Name string `json:"name"`
					}
					if err := json.Unmarshal([]byte(ev.Message), &p); err == nil {
						delete(active, p.Name)
						completed[p.Name] = struct{}{}
						doneTables = append(doneTables, p.Name)
						logf("table_done name=%s completed_total=%d", p.Name, len(doneTables))
						updateArea()
					}
					continue
				}
				if string(ev.Type) == "table_failed" {
					var p struct {
						Name   string `json:"name"`
						Reason string `json:"reason"`
					}
					if err := json.Unmarshal([]byte(ev.Message), &p); err == nil {
						delete(active, p.Name)
						failed[p.Name] = p.Reason
						seedingFailed = true
						logf("table_failed name=%s reason=%s", p.Name, p.Reason)
						updateArea()
					}
					continue
				}

				// default: avoid mixing standard prints once area is active to prevent flicker
				// Intentionally suppress legacy plan/progress rendering to avoid noisy output
			}
			close(doneEvents)
		}()

		// Worker pool for tasks
		concurrency := 4

		// Pretty completion notifier
		notifyCompletion := func(elapsed time.Duration, tableCount int) {
			title := pterm.NewStyle(pterm.FgGreen, pterm.Bold).Sprint("Seeding Completed")
			details := fmt.Sprintf("Duration: %s\nTables seeded: %d", elapsed, tableCount)
			box := pterm.DefaultBox.WithTitle(title).WithPadding(1).Sprint(details)
			pterm.Println(box)
		}
		// Failure notifier
		notifyFailure := func(elapsed time.Duration) {
			title := pterm.NewStyle(pterm.FgRed, pterm.Bold).Sprint("Seeding Failed")
			details := fmt.Sprintf("Duration: %s\n\nThe seeding process has failed.\nYou will not be charged any credits for this session.", elapsed)
			box := pterm.DefaultBox.WithTitle(title).WithPadding(1).Sprint(details)
			pterm.Println(box)
		}
		// ctx/cancel already defined above
		var wg sync.WaitGroup
		wg.Add(concurrency)
		for i := 0; i < concurrency; i++ {
			go func() {
				defer wg.Done()
				for task := range br.Tasks() {
					// Log received task for debugging
					logf("DEBUG: Received SQL task - ID=%s, IsWrite=%v, Schema=%s, SQL=%s",
						task.RequestID, task.IsWrite, task.Schema, task.SQLStatement)

					// Use schema from task
					schema := task.Schema

					resultJSON, err := exec.ExecuteSQLInSchema(ctx, task.SQLStatement, task.IsWrite, schema)
					if err != nil {
						logf("ERROR: ExecuteSQLInSchema failed: %v", err)
						continue
					}

					// Check if the result contains an error field
					// The executor returns JSON like {"error": "..."} on failure
					var resultCheck struct {
						Error string `json:"error"`
					}
					success := true
					if err := json.Unmarshal([]byte(resultJSON), &resultCheck); err == nil {
						if resultCheck.Error != "" {
							success = false
							logf("DEBUG: SQL execution failed - ID=%s, Error=%s", task.RequestID, resultCheck.Error)
						}
					}

					// Log the result for debugging
					logf("DEBUG: SQL task completed - ID=%s, IsWrite=%v, Success=%v, ResultLength=%d",
						task.RequestID, task.IsWrite, success, len(resultJSON))

					// Send response with error handling
					resp := model.SQLResponse{
						RequestID:  task.RequestID,
						Success:    success,
						ResultJSON: resultJSON,
					}

					if sendErr := br.SendSQLResponse(ctx, resp); sendErr != nil {
						logf("ERROR: Failed to send SQL response: %v", sendErr)
						continue
					}

					logf("DEBUG: SQL response sent successfully - ID=%s", task.RequestID)
				}
			}()
		}
		doneTasks := make(chan struct{})
		go func() { wg.Wait(); close(doneTasks) }()

		<-doneEvents
		<-doneTasks

		stopArea()
		if planningActive && planSpinner != nil {
			planSpinner.Stop()
		}
		elapsed := time.Since(startAt).Round(time.Millisecond)
		if streamErr != nil {
			if !earlyNotified {
				pterm.Printf("Session duration: %s\n", elapsed)
				logging.PresentStreamError(streamErr.Error())
			}
			return streamErr
		}
		// Check if any tables failed
		if seedingFailed {
			notifyFailure(elapsed)
			return nil
		}
		// Prefer explicit workflow completion signal when provided by server
		if workflowCompleted {
			notifyCompletion(elapsed, len(doneTables))
			return nil
		}
		if len(doneTables) > 0 {
			// Only claim success if we completed all expected tables
			completedAll := expectedCount == 0 || len(completed) == expectedCount
			if completedAll {
				notifyCompletion(elapsed, len(doneTables))
			} else {
				pterm.Warning.Printf("Connection closed before completing all tables after %s :(\n", elapsed)
			}
		} else if streamClosed {
			notifyCompletion(elapsed, len(doneTables))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(seedCmd)
	// Verbose flag temporarily disabled
	// seedCmd.Flags().BoolVarP(&verboseSeed, "verbose", "v", false, "Enable verbose debug output")
}

// deriveDBName extracts the database name from a PostgreSQL DSN URL.
// It parses the DSN and returns the database name from the path component.
// Returns an empty string if the DSN cannot be parsed (fail-fast approach with no defaults).
//
// The function expects a DSN in the format: postgres://user:pass@host:5432/dbname?params
// and extracts "dbname" from the path component.
func deriveDBName(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return ""
	}
	p := strings.TrimPrefix(u.Path, "/")
	return p
}
