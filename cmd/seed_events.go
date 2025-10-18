package cmd

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"sync"

	"seedfast/cli/internal/bridge/model"
	"seedfast/cli/internal/terminal"

	"github.com/pterm/pterm"
)

// handlePlanProposed processes a plan_proposed event from the backend.
// It stops any active spinners, resets the UI state, displays the proposed seeding scope,
// and updates the expected tables for progress tracking.
//
// The function handles both structured JSON payloads with "tables" and "preview" fields,
// and falls back to flexible parsing for different payload formats.
func handlePlanProposed(
	message string,
	stopHeader func(),
	planningActive *bool,
	planSpinner **pterm.SpinnerPrinter,
	stopArea func(),
	resetAreaState func(),
	expectedTables map[string]struct{},
	expectedCount *int,
	awaitingDecision *bool,
	scopeShown *bool,
	mu *sync.Mutex,
) {
	stopHeader()
	if *planningActive && *planSpinner != nil {
		(*planSpinner).Stop()
		*planningActive = false
	}
	stopArea()
	resetAreaState()

	var payload struct {
		Preview string   `json:"preview"`
		Tables  []string `json:"tables"`
	}
	candidateTables := []string{}
	candidatePreview := ""
	if err := json.Unmarshal([]byte(message), &payload); err == nil {
		candidateTables = payload.Tables
		candidatePreview = payload.Preview
	}
	if len(candidateTables) == 0 {
		if t, p := extractTablesAndPreviewFlexible(message); len(t) > 0 {
			candidateTables, candidatePreview = t, p
		}
	}
	mu.Lock()
	if len(candidateTables) > 0 {
		for _, t := range candidateTables {
			expectedTables[t] = struct{}{}
		}
		*expectedCount = len(expectedTables)
	}
	mu.Unlock()
	*awaitingDecision = true
	pterm.Println(pterm.NewStyle(pterm.FgLightCyan, pterm.Bold).Sprint("Proposed seeding scope"))
	if candidatePreview != "" {
		pterm.Println(candidatePreview)
		*scopeShown = true
		return
	}
	if len(candidateTables) > 0 {
		var items []pterm.BulletListItem
		for _, s := range candidateTables {
			items = append(items, pterm.BulletListItem{Level: 0, Text: s})
		}
		_ = pterm.DefaultBulletList.WithItems(items).Render()
		*scopeShown = true
	}
}

// handleAskHuman processes an ask_human event from the backend.
// It presents a question to the user, collects their response, and sends it back
// to the backend for further processing. The function handles user input collection,
// response formatting, and state management for the interactive seeding process.
func handleAskHuman(
	message string,
	stopHeader func(),
	planningActive *bool,
	planSpinner **pterm.SpinnerPrinter,
	scopeShown *bool,
	expectedTables map[string]struct{},
	expectedCount *int,
	sendResponse func(resp model.SQLResponse),
	mu *sync.Mutex,
) {
	stopHeader()
	if *planningActive && *planSpinner != nil {
		(*planSpinner).Stop()
		*planningActive = false
	}
	var payload struct {
		QuestionID string `json:"question_id"`
		Question   string `json:"question"`
		Context    struct {
			Tables []string `json:"tables"`
		} `json:"context"`
	}
	if err := json.Unmarshal([]byte(message), &payload); err != nil {
		return
	}
	mu.Lock()
	if !*scopeShown && len(payload.Context.Tables) > 0 {
		for _, t := range payload.Context.Tables {
			expectedTables[t] = struct{}{}
		}
		*expectedCount = len(expectedTables)
	}
	mu.Unlock()

	if !*scopeShown && len(payload.Context.Tables) > 0 {
		pterm.Println(pterm.NewStyle(pterm.FgLightCyan, pterm.Bold).Sprint("Proposed seeding scope"))
		var items []pterm.BulletListItem
		for _, s := range payload.Context.Tables {
			items = append(items, pterm.BulletListItem{Level: 0, Text: s})
		}
		_ = pterm.DefaultBulletList.WithItems(items).Render()
		*scopeShown = true
	}
	prompt := strings.TrimSpace(payload.Question)
	if prompt == "" {
		prompt = "Press Enter to accept the proposed scope and continue. Or type any message to send to the planner."
	}
	pterm.Println(prompt)
	promptText := "Your answer: "
	pterm.Print(promptText)
	reader := bufio.NewReader(os.Stdin)
	ans, _ := reader.ReadString('\n')
	ans = strings.TrimSpace(ans)

	// Clear the "Your answer:" prompt and user input from terminal
	terminal.ClearPreviousLines(len(promptText) + len(ans))

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
		sendResponse(model.SQLResponse{RequestID: payload.QuestionID, Success: true, ResultJSON: string(b)})
		if ans != "" && !*planningActive {
			pterm.Println("")
			*planningActive = true
		}
	}
}

// handleTableStarted processes a table_started event from the backend.
// It updates the internal state to track the newly started table, manages the
// progress tracking, and triggers the UI area spinner to show progress for the
// active seeding operation.
func handleTableStarted(
	message string,
	stopHeader func(),
	awaitingDecision *bool,
	planningActive *bool,
	planSpinner **pterm.SpinnerPrinter,
	active map[string]int,
	order *[]string,
	expectedTables map[string]struct{},
	expectedCount *int,
	startArea func(),
	updateArea func(),
	mu *sync.Mutex,
) {
	var p struct {
		Name      string `json:"name"`
		Remaining int    `json:"remaining"`
	}
	if err := json.Unmarshal([]byte(message), &p); err != nil {
		return
	}
	stopHeader()
	if *awaitingDecision {
		*awaitingDecision = false
		if *planningActive && *planSpinner != nil {
			(*planSpinner).Stop()
			*planningActive = false
		}
	}
	mu.Lock()
	if _, ok := active[p.Name]; !ok {
		*order = append(*order, p.Name)
	}
	active[p.Name] = p.Remaining
	if _, ok := expectedTables[p.Name]; !ok {
		expectedTables[p.Name] = struct{}{}
		*expectedCount = len(expectedTables)
	}
	mu.Unlock()
	startArea()
	updateArea()
}

// handleTableDone processes a table_done event from the backend.
// It updates the internal state to mark the completed table, moves it from
// active to completed status, and triggers UI updates to reflect the progress.
func handleTableDone(
	message string,
	active map[string]int,
	completed map[string]struct{},
	doneTables *[]string,
	updateArea func(),
	mu *sync.Mutex,
) {
	var p struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(message), &p); err != nil {
		return
	}
	mu.Lock()
	delete(active, p.Name)
	completed[p.Name] = struct{}{}
	*doneTables = append(*doneTables, p.Name)
	mu.Unlock()
	updateArea()
}
