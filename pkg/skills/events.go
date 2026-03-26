package skills

import "time"

// SkillExecutionEvent represents a skill execution event
type SkillExecutionEvent struct {
	SkillName string                 `json:"skill_name"`
	Input     string                 `json:"input"`
	Context   map[string]interface{} `json:"context"`
	RequestID string                 `json:"request_id"`
	Timestamp time.Time              `json:"timestamp"`
}