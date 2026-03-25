package systemdb

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type Event struct {
	CreatedAt   time.Time `json:"created_at"`
	ServiceID   string    `json:"service_id,omitempty"`
	Severity    string    `json:"severity"`
	EventType   string    `json:"event_type"`
	Message     string    `json:"message"`
	PayloadJSON string    `json:"payload_json,omitempty"`
}

type ServiceStatus struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	State     string    `json:"state"`
	Running   bool      `json:"running"`
	Owned     bool      `json:"owned"`
	Action    string    `json:"action,omitempty"`
	Message   string    `json:"message,omitempty"`
	PIDFile   string    `json:"pid_file,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Stats24h struct {
	Since          time.Time `json:"since"`
	Until          time.Time `json:"until"`
	TotalEvents    int       `json:"total_events"`
	ErrorEvents    int       `json:"error_events"`
	CriticalEvents int       `json:"critical_events"`
	WarningEvents  int       `json:"warning_events"`
}

type ProcessRecord struct {
	Name    string `json:"name"`
	PIDFile string `json:"pid_file,omitempty"`
	Running bool   `json:"running"`
	Owned   bool   `json:"owned"`
	Action  string `json:"action,omitempty"`
	Message string `json:"message,omitempty"`
}

type MaintenanceRecord struct {
	Timestamp             time.Time                `json:"timestamp"`
	Score                 int                      `json:"score"`
	Summary               string                   `json:"summary"`
	Baseline              bool                     `json:"baseline,omitempty"`
	Deferred              bool                     `json:"deferred,omitempty"`
	ComparedTo            string                   `json:"compared_to,omitempty"`
	Recommendations       []string                 `json:"recommendations,omitempty"`
	Processes             map[string]ProcessRecord `json:"processes,omitempty"`
	LastRemediationAt     time.Time                `json:"last_remediation_at,omitempty"`
	LastRemediationAction string                   `json:"last_remediation_action,omitempty"`
	PayloadJSON           string                   `json:"payload_json,omitempty"`
}

type SnapshotRecord struct {
	ID                    int       `json:"id"`
	Kind                  string    `json:"kind"`
	CreatedAt             time.Time `json:"created_at"`
	Score                 int       `json:"score"`
	Summary               string    `json:"summary"`
	Deferred              bool      `json:"deferred"`
	ComparedTo            string    `json:"compared_to,omitempty"`
	LastRemediationAt     time.Time `json:"last_remediation_at,omitempty"`
	LastRemediationAction string    `json:"last_remediation_action,omitempty"`
	PayloadJSON           string    `json:"payload_json,omitempty"`
}

type JournalEntry struct {
	Date      string    `json:"date"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Style     string    `json:"style"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func Open(path string) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("system db path is empty")
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
		`CREATE TABLE IF NOT EXISTS benchmark_snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			kind TEXT NOT NULL,
			created_at TEXT NOT NULL,
			score INTEGER NOT NULL,
			summary TEXT NOT NULL,
			deferred INTEGER NOT NULL DEFAULT 0,
			compared_to TEXT,
			last_remediation_at TEXT,
			last_remediation_action TEXT,
			payload_json TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS service_status (
			service_id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			state TEXT NOT NULL,
			running INTEGER NOT NULL,
			owned INTEGER NOT NULL,
			action TEXT,
			message TEXT,
			pid_file TEXT,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS observer_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at TEXT NOT NULL,
			service_id TEXT,
			severity TEXT NOT NULL,
			event_type TEXT NOT NULL,
			message TEXT NOT NULL,
			payload_json TEXT
		);`,
		`CREATE INDEX IF NOT EXISTS idx_observer_events_created_at ON observer_events(created_at);`,
		`CREATE TABLE IF NOT EXISTS observer_journal (
			journal_date TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			body TEXT NOT NULL,
			style TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) RecordMaintenanceRun(snapshot MaintenanceRecord, baseline *MaintenanceRecord) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err = insertSnapshot(tx, snapshot); err != nil {
		return err
	}
	if err = upsertServices(tx, snapshot); err != nil {
		return err
	}
	if err = insertEvents(tx, snapshot, baseline); err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

func (s *Store) Stats24h() (Stats24h, error) {
	until := time.Now().UTC()
	since := until.Add(-24 * time.Hour)
	row := s.db.QueryRow(`
		SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN severity = 'error' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN severity = 'critical' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN severity = 'warning' THEN 1 ELSE 0 END), 0)
		FROM observer_events
		WHERE created_at >= ?`, since.Format(time.RFC3339))
	var out Stats24h
	out.Since = since
	out.Until = until
	if err := row.Scan(&out.TotalEvents, &out.ErrorEvents, &out.CriticalEvents, &out.WarningEvents); err != nil {
		return Stats24h{}, err
	}
	return out, nil
}

func (s *Store) CurrentServices() ([]ServiceStatus, error) {
	rows, err := s.db.Query(`
		SELECT service_id, name, state, running, owned, action, message, pid_file, updated_at
		FROM service_status
		ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ServiceStatus
	for rows.Next() {
		var item ServiceStatus
		var running int
		var owned int
		var updatedAt string
		if err := rows.Scan(&item.ID, &item.Name, &item.State, &running, &owned, &item.Action, &item.Message, &item.PIDFile, &updatedAt); err != nil {
			return nil, err
		}
		item.Running = running == 1
		item.Owned = owned == 1
		item.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) RecentEvents(limit int) ([]Event, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(`
		SELECT created_at, service_id, severity, event_type, message, payload_json
		FROM observer_events
		ORDER BY created_at DESC, id DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Event
	for rows.Next() {
		var item Event
		var createdAt string
		if err := rows.Scan(&createdAt, &item.ServiceID, &item.Severity, &item.EventType, &item.Message, &item.PayloadJSON); err != nil {
			return nil, err
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) EventsBetween(start, end time.Time, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.Query(`
		SELECT created_at, service_id, severity, event_type, message, payload_json
		FROM observer_events
		WHERE created_at >= ? AND created_at < ?
		ORDER BY created_at DESC, id DESC
		LIMIT ?`, start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Event
	for rows.Next() {
		var item Event
		var createdAt string
		if err := rows.Scan(&createdAt, &item.ServiceID, &item.Severity, &item.EventType, &item.Message, &item.PayloadJSON); err != nil {
			return nil, err
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) RecentSnapshots(limit int) ([]SnapshotRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(`
		SELECT id, kind, created_at, score, summary, deferred, compared_to, last_remediation_at, last_remediation_action, payload_json
		FROM benchmark_snapshots
		ORDER BY created_at DESC, id DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SnapshotRecord
	for rows.Next() {
		var item SnapshotRecord
		var createdAt string
		var deferred int
		var lastRemediationAt string
		if err := rows.Scan(
			&item.ID,
			&item.Kind,
			&createdAt,
			&item.Score,
			&item.Summary,
			&deferred,
			&item.ComparedTo,
			&lastRemediationAt,
			&item.LastRemediationAction,
			&item.PayloadJSON,
		); err != nil {
			return nil, err
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		item.Deferred = deferred == 1
		item.LastRemediationAt, _ = time.Parse(time.RFC3339, lastRemediationAt)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) UpsertJournal(entry JournalEntry) error {
	now := time.Now().UTC()
	if entry.Date == "" {
		return fmt.Errorf("journal date is required")
	}
	if entry.Style == "" {
		entry.Style = "dark_humor"
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = now
	}
	entry.UpdatedAt = now
	_, err := s.db.Exec(`
		INSERT INTO observer_journal (journal_date, title, body, style, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(journal_date) DO UPDATE SET
			title = excluded.title,
			body = excluded.body,
			style = excluded.style,
			updated_at = excluded.updated_at`,
		entry.Date,
		entry.Title,
		entry.Body,
		entry.Style,
		entry.CreatedAt.UTC().Format(time.RFC3339),
		entry.UpdatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *Store) LatestJournal() (*JournalEntry, error) {
	row := s.db.QueryRow(`
		SELECT journal_date, title, body, style, created_at, updated_at
		FROM observer_journal
		ORDER BY journal_date DESC
		LIMIT 1`)
	return scanJournalRow(row)
}

func (s *Store) JournalByDate(date string) (*JournalEntry, error) {
	row := s.db.QueryRow(`
		SELECT journal_date, title, body, style, created_at, updated_at
		FROM observer_journal
		WHERE journal_date = ?`, date)
	return scanJournalRow(row)
}

func scanJournalRow(row *sql.Row) (*JournalEntry, error) {
	var entry JournalEntry
	var createdAt string
	var updatedAt string
	if err := row.Scan(&entry.Date, &entry.Title, &entry.Body, &entry.Style, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	entry.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	entry.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &entry, nil
}

// DeleteEventsBefore removes events older than the specified timestamp
// Returns the number of deleted events
func (s *Store) DeleteEventsBefore(cutoff time.Time) (int, error) {
	result, err := s.db.Exec(`
		DELETE FROM observer_events
		WHERE created_at < ?`,
		cutoff.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(affected), nil
}

func insertSnapshot(tx *sql.Tx, snapshot MaintenanceRecord) error {
	kind := "current"
	if snapshot.Baseline {
		kind = "baseline"
	}
	payload := snapshot.PayloadJSON
	if payload == "" {
		data, err := json.Marshal(snapshot)
		if err != nil {
			return err
		}
		payload = string(data)
	}
	_, err := tx.Exec(`
		INSERT INTO benchmark_snapshots (
			kind, created_at, score, summary, deferred, compared_to, last_remediation_at, last_remediation_action, payload_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		kind,
		snapshot.Timestamp.Format(time.RFC3339),
		snapshot.Score,
		snapshot.Summary,
		boolToInt(snapshot.Deferred),
		snapshot.ComparedTo,
		formatTime(snapshot.LastRemediationAt),
		snapshot.LastRemediationAction,
		payload,
	)
	return err
}

func upsertServices(tx *sql.Tx, snapshot MaintenanceRecord) error {
	keys := make([]string, 0, len(snapshot.Processes))
	for id := range snapshot.Processes {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	for _, id := range keys {
		proc := snapshot.Processes[id]
		state := serviceState(proc)
		_, err := tx.Exec(`
			INSERT INTO service_status (
				service_id, name, state, running, owned, action, message, pid_file, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(service_id) DO UPDATE SET
				name = excluded.name,
				state = excluded.state,
				running = excluded.running,
				owned = excluded.owned,
				action = excluded.action,
				message = excluded.message,
				pid_file = excluded.pid_file,
				updated_at = excluded.updated_at`,
			id,
			proc.Name,
			state,
			boolToInt(proc.Running),
			boolToInt(proc.Owned),
			proc.Action,
			proc.Message,
			proc.PIDFile,
			snapshot.Timestamp.Format(time.RFC3339),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertEvents(tx *sql.Tx, snapshot MaintenanceRecord, baseline *MaintenanceRecord) error {
	events := deriveEvents(snapshot, baseline)
	for _, event := range events {
		_, err := tx.Exec(`
			INSERT INTO observer_events (created_at, service_id, severity, event_type, message, payload_json)
			VALUES (?, ?, ?, ?, ?, ?)`,
			event.CreatedAt.Format(time.RFC3339),
			event.ServiceID,
			event.Severity,
			event.EventType,
			event.Message,
			event.PayloadJSON,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func deriveEvents(snapshot MaintenanceRecord, baseline *MaintenanceRecord) []Event {
	var events []Event
	add := func(serviceID, severity, eventType, message string, payload any) {
		payloadJSON := ""
		if payload != nil {
			if data, err := json.Marshal(payload); err == nil {
				payloadJSON = string(data)
			}
		}
		events = append(events, Event{
			CreatedAt:   snapshot.Timestamp,
			ServiceID:   serviceID,
			Severity:    severity,
			EventType:   eventType,
			Message:     message,
			PayloadJSON: payloadJSON,
		})
	}

	if snapshot.Baseline {
		add("", "info", "baseline_created", "Startup baseline benchmark recorded", map[string]any{
			"score":   snapshot.Score,
			"summary": snapshot.Summary,
		})
	}
	switch snapshot.Summary {
	case "degraded":
		add("", "critical", "runtime_degraded", "Runtime health is degraded", map[string]any{"score": snapshot.Score})
	case "watch":
		add("", "warning", "runtime_watch", "Runtime health requires attention", map[string]any{"score": snapshot.Score})
	}
	if baseline != nil && snapshot.Score < baseline.Score-20 {
		add("", "warning", "baseline_regression", "Runtime score has materially regressed against baseline", map[string]any{
			"baseline_score": baseline.Score,
			"current_score":  snapshot.Score,
		})
	}
	for id, proc := range snapshot.Processes {
		if !proc.Running {
			add(id, "error", "service_offline", fmt.Sprintf("%s is offline", proc.Name), proc)
		}
		switch proc.Action {
		case "restart_requested":
			add(id, "warning", "restart_requested", fmt.Sprintf("%s restart requested", proc.Name), proc)
		case "restart_failed":
			add(id, "error", "restart_failed", fmt.Sprintf("%s restart failed", proc.Name), proc)
		case "stale_pid_removed", "removed_empty_pid_file":
			add(id, "info", proc.Action, fmt.Sprintf("%s maintenance cleaned stale pid state", proc.Name), proc)
		}
	}
	return events
}

func serviceState(proc ProcessRecord) string {
	if proc.Action == "restart_requested" {
		return "restarting"
	}
	if proc.Action == "restart_failed" {
		return "degraded"
	}
	if proc.Running {
		return "online"
	}
	return "offline"
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func formatTime(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.Format(time.RFC3339)
}
