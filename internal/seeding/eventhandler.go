package seeding

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"seedfast/cli/internal/bridge/model"

	"github.com/pterm/pterm"
)

// BackendEventType represents the type of event from the backend.
type BackendEventType string

const (
	BackendEventPlanProposed      BackendEventType = "plan_proposed"
	BackendEventAskHuman          BackendEventType = "ask_human"
	BackendEventSessionReady      BackendEventType = "session_ready"
	BackendEventTableStarted      BackendEventType = "table_started"
	BackendEventTableDone         BackendEventType = "table_done"
	BackendEventTableFailed       BackendEventType = "table_failed"
	BackendEventWorkflowCompleted BackendEventType = "workflow_completed"
	BackendEventStreamClosed      BackendEventType = "stream_closed"
	BackendEventStreamError       BackendEventType = "stream_error"
)

// ResponseSender is a function that sends responses back to the backend.
type ResponseSender func(ctx context.Context, resp model.SQLResponse) error

// PlanProposedPayload represents the payload for plan_proposed events.
type PlanProposedPayload struct {
	Preview string   `json:"preview"`
	Tables  []string `json:"tables"`
}

// AskHumanPayload represents the payload for ask_human events.
type AskHumanPayload struct {
	QuestionID string `json:"question_id"`
	Question   string `json:"question"`
	Context    struct {
		Tables []string `json:"tables"`
	} `json:"context"`
}

// TableStartedPayload represents the payload for table_started events.
type TableStartedPayload struct {
	Name      string `json:"name"`
	Remaining int    `json:"remaining"`
}

// TableDonePayload represents the payload for table_done events.
type TableDonePayload struct {
	Name string `json:"name"`
}

// TableFailedPayload represents the payload for table_failed events.
type TableFailedPayload struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// ExtractTablesAndPreview attempts to parse tables and preview text from a
// flexible JSON payload. It supports several common shapes to be resilient
// to backend changes.
func ExtractTablesAndPreview(jsonStr string) ([]string, string) {
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

	// Deduplicate tables
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

	// Extract preview text
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

// PromptUser displays a question to the user and returns their answer.
// If the user provides an empty answer (just Enter), it returns (true, "").
// If the user provides text, it returns (false, answer).
func PromptUser(question string) (accepted bool, answer string) {
	prompt := question
	if strings.TrimSpace(prompt) == "" {
		prompt = "Press Enter to accept the proposed scope and continue. Or type any message to send to the planner."
	}

	pterm.Println(prompt)
	pterm.Print("Your answer: ")

	reader := bufio.NewReader(os.Stdin)
	ans, _ := reader.ReadString('\n')
	ans = strings.TrimSpace(ans)

	if ans == "" {
		pterm.Info.Println("Empty input interpreted as acceptance. Continuing with the proposed scope.")
		return true, ""
	}

	return false, ans
}

// DisplayPlanScope displays the proposed seeding scope to the user.
// It shows either a preview text or a bulleted list of table names.
func DisplayPlanScope(preview string, tables []string) {
	pterm.Println(pterm.NewStyle(pterm.FgLightCyan, pterm.Bold).Sprint("Proposed seeding scope"))

	if preview != "" {
		pterm.Println(preview)
	} else if len(tables) > 0 {
		items := make([]pterm.BulletListItem, len(tables))
		for i, t := range tables {
			items[i] = pterm.BulletListItem{Level: 0, Text: t}
		}
		_ = pterm.DefaultBulletList.WithItems(items).Render()
	}
}

// CreateHumanResponse creates a response payload for ask_human events.
func CreateHumanResponse(questionID string, accepted bool, answer string) map[string]any {
	if accepted {
		return map[string]any{
			"human_answer": true,
			"question_id":  questionID,
			"answer":       map[string]any{"raw": ""},
		}
	}
	return map[string]any{
		"human_answer": false,
		"question_id":  questionID,
		"answer":       map[string]any{"raw": answer},
	}
}

// SendHumanResponse sends a human response back to the backend.
func SendHumanResponse(ctx context.Context, sender ResponseSender, questionID string, respObj map[string]any) error {
	b, err := json.Marshal(respObj)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	return sender(ctx, model.SQLResponse{
		RequestID:  questionID,
		Success:    true,
		ResultJSON: string(b),
	})
}
