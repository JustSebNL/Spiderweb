package agentdb

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type AgentPresence struct {
	AgentID         string    `json:"agent_id"`
	AgentName       string    `json:"agent_name"`
	ManagerID       string    `json:"manager_id,omitempty"`
	PipelineID      string    `json:"pipeline_id,omitempty"`
	State           string    `json:"state"`
	Status          string    `json:"status,omitempty"`
	Channel         string    `json:"channel,omitempty"`
	ChatID          string    `json:"chat_id,omitempty"`
	LastTaskSummary string    `json:"last_task_summary,omitempty"`
	LastError       string    `json:"last_error,omitempty"`
	LastSeenAt      time.Time `json:"last_seen_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func Open(path string) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("agent db path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	store := &Store{db: db}
	if err := store.init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) init() error {
	stmts := []string{
		`PRAGMA journal_mode = WAL;`,
		`PRAGMA busy_timeout = 5000;`,
		`CREATE TABLE IF NOT EXISTS agent_status (
			agent_id TEXT PRIMARY KEY,
			agent_name TEXT NOT NULL,
			manager_id TEXT,
			pipeline_id TEXT,
			state TEXT NOT NULL,
			status TEXT,
			channel TEXT,
			chat_id TEXT,
			last_task_summary TEXT,
			last_error TEXT,
			last_seen_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_agent_status_updated_at ON agent_status(updated_at);`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) RecordPresence(p AgentPresence) error {
	if p.AgentID == "" {
		return fmt.Errorf("agent id is required")
	}
	if p.AgentName == "" {
		p.AgentName = p.AgentID
	}
	if p.State == "" {
		p.State = "busy"
	}
	if p.LastSeenAt.IsZero() {
		p.LastSeenAt = time.Now().UTC()
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = p.LastSeenAt
	}
	_, err := s.db.Exec(`
		INSERT INTO agent_status (
			agent_id, agent_name, manager_id, pipeline_id, state, status, channel, chat_id,
			last_task_summary, last_error, last_seen_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(agent_id) DO UPDATE SET
			agent_name = excluded.agent_name,
			manager_id = excluded.manager_id,
			pipeline_id = excluded.pipeline_id,
			state = excluded.state,
			status = excluded.status,
			channel = excluded.channel,
			chat_id = excluded.chat_id,
			last_task_summary = excluded.last_task_summary,
			last_error = excluded.last_error,
			last_seen_at = excluded.last_seen_at,
			updated_at = excluded.updated_at`,
		p.AgentID,
		p.AgentName,
		p.ManagerID,
		p.PipelineID,
		p.State,
		p.Status,
		p.Channel,
		p.ChatID,
		p.LastTaskSummary,
		p.LastError,
		p.LastSeenAt.UTC().Format(time.RFC3339),
		p.UpdatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *Store) CurrentAgents() ([]AgentPresence, error) {
	rows, err := s.db.Query(`
		SELECT agent_id, agent_name, manager_id, pipeline_id, state, status, channel, chat_id,
		       last_task_summary, last_error, last_seen_at, updated_at
		FROM agent_status
		ORDER BY agent_name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	now := time.Now().UTC()
	var out []AgentPresence
	for rows.Next() {
		var item AgentPresence
		var lastSeenAt string
		var updatedAt string
		if err := rows.Scan(
			&item.AgentID,
			&item.AgentName,
			&item.ManagerID,
			&item.PipelineID,
			&item.State,
			&item.Status,
			&item.Channel,
			&item.ChatID,
			&item.LastTaskSummary,
			&item.LastError,
			&lastSeenAt,
			&updatedAt,
		); err != nil {
			return nil, err
		}
		item.LastSeenAt, _ = time.Parse(time.RFC3339, lastSeenAt)
		item.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		item.State = deriveAgentState(item.State, item.LastSeenAt, now)
		out = append(out, item)
	}
	return out, rows.Err()
}

func deriveAgentState(state string, lastSeenAt, now time.Time) string {
	if state == "degraded" || state == "restarting" {
		return state
	}
	if !lastSeenAt.IsZero() && now.Sub(lastSeenAt) > 5*time.Minute {
		return "offline"
	}
	if state == "idle" || state == "busy" || state == "online" {
		return state
	}
	return "online"
}
