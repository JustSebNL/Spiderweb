package cognition

import "time"

// Event is the normalized Spiderweb event shape used for cheap cognition.
type Event struct {
	EventID        string                 `json:"event_id,omitempty"`
	Source         string                 `json:"source,omitempty"`
	SourceObject   string                 `json:"source_object,omitempty"`
	EventType      string                 `json:"event_type,omitempty"`
	OccurredAt     time.Time              `json:"occurred_at,omitempty"`
	Actor          string                 `json:"actor,omitempty"`
	ImportanceHint string                 `json:"importance_hint,omitempty"`
	Project        string                 `json:"project,omitempty"`
	Payload        map[string]any         `json:"payload,omitempty"`
	Metadata       map[string]any         `json:"metadata,omitempty"`
	DedupeKey      string                 `json:"dedupe_key,omitempty"`
	Version        int                    `json:"version,omitempty"`
}

// ClassificationResult is the compact triage result returned by the cheap cognition layer.
type ClassificationResult struct {
	Priority         string `json:"priority"`
	Category         string `json:"category"`
	EscalationNeeded bool   `json:"escalation_needed"`
	OneLineSummary   string `json:"one_line_summary"`
	RawContent       string `json:"-"`
}
