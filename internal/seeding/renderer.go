package seeding

import (
	"github.com/pterm/pterm"
)

// Renderer renders seeding events to console with docker-compose-like UI.
type Renderer struct {
	sectionShown bool
}

// NewRenderer creates a renderer instance.
func NewRenderer() *Renderer { return &Renderer{} }

// ShowSection prints the Seeding section header once and marks it as shown.
func (r *Renderer) ShowSection() {
	if !r.sectionShown {
		// Header rendering moved to cmd/seed.go with a spinner-based line.
		// Keep state so repeated calls do not attempt to re-render.
		r.sectionShown = true
	}
}

// Render processes a single event.
func (r *Renderer) Render(ev Event) {
	if !r.sectionShown {
		// Header rendering moved to cmd/seed.go.
		r.sectionShown = true
	}
	switch ev.Type {
	case EventMCPLog:
		// Suppressed to keep UI clean
	case EventSeedState:
		// Suppressed to keep UI clean
	case EventBatchPlan:
		// Suppressed to keep UI clean
	case EventBatchProgress:
		// Suppressed to keep UI clean
	case EventTableStatus:
		// Suppressed to keep UI clean
	}
}

func stringListToBulletItems(items []string) (out []pterm.BulletListItem) {
	for _, s := range items {
		out = append(out, pterm.BulletListItem{Level: 0, Text: s})
	}
	return out
}
