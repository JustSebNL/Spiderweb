package observer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/JustSebNL/Spiderweb/pkg/agentdb"
	"github.com/JustSebNL/Spiderweb/pkg/maintenance"
	"github.com/JustSebNL/Spiderweb/pkg/systemdb"
)

var (
	ErrHealthFileNotSet = errors.New("observer health file not set")
	ErrWorkspaceNotSet  = errors.New("workspace not set")
)

type Store struct {
	healthFile   string
	baselineFile string
	systemDBFile string
	agentDBFile  string
}

type Overview struct {
	GeneratedAt           time.Time          `json:"generated_at"`
	Status                string             `json:"status"`
	Score                 int                `json:"score"`
	Summary               string             `json:"summary"`
	Deferred              bool               `json:"deferred"`
	ComparedTo            string             `json:"compared_to,omitempty"`
	Recommendations       []string           `json:"recommendations,omitempty"`
	LastRemediationAt     time.Time          `json:"last_remediation_at,omitempty"`
	LastRemediationAction string             `json:"last_remediation_action,omitempty"`
	Services              []ServiceStatus    `json:"services"`
	CheapCognition        any                `json:"cheap_cognition"`
	Baseline              *BenchmarkSummary  `json:"baseline,omitempty"`
	Stats24h              *systemdb.Stats24h `json:"stats_24h,omitempty"`
}

type BenchmarkSummary struct {
	Timestamp time.Time `json:"timestamp"`
	Score     int       `json:"score"`
	Summary   string    `json:"summary"`
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
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type AgentStatus struct {
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
	LastSeenAt      time.Time `json:"last_seen_at,omitempty"`
	UpdatedAt       time.Time `json:"updated_at,omitempty"`
}

type AgentSummary struct {
	GeneratedAt time.Time      `json:"generated_at"`
	Total       int            `json:"total"`
	ByState     map[string]int `json:"by_state"`
	ByPipeline  map[string]int `json:"by_pipeline"`
}

type BenchmarksResponse struct {
	GeneratedAt time.Time                   `json:"generated_at"`
	Current     *maintenance.HealthSnapshot `json:"current,omitempty"`
	Baseline    *maintenance.HealthSnapshot `json:"baseline,omitempty"`
}

type EventsResponse struct {
	GeneratedAt time.Time        `json:"generated_at"`
	Events      []systemdb.Event `json:"events"`
}

type SelfCareCycle struct {
	ID                    int       `json:"id"`
	Kind                  string    `json:"kind"`
	Timestamp             time.Time `json:"timestamp"`
	Score                 int       `json:"score"`
	Summary               string    `json:"summary"`
	Deferred              bool      `json:"deferred"`
	ComparedTo            string    `json:"compared_to,omitempty"`
	LastRemediationAt     time.Time `json:"last_remediation_at,omitempty"`
	LastRemediationAction string    `json:"last_remediation_action,omitempty"`
	BaselineSnapshotID    int       `json:"baseline_snapshot_id,omitempty"`
	PreCheckSnapshotID    int       `json:"pre_check_snapshot_id,omitempty"`
	PostCareSnapshotID    int       `json:"post_care_snapshot_id,omitempty"`

	// Enhanced cycle tracking
	BaselineScore   int       `json:"baseline_score,omitempty"`
	BaselineSummary string    `json:"baseline_summary,omitempty"`
	BaselineAt      time.Time `json:"baseline_at,omitempty"`
	PreCheckScore   int       `json:"pre_check_score,omitempty"`
	PreCheckSummary string    `json:"pre_check_summary,omitempty"`
	PreCheckAt      time.Time `json:"pre_check_at,omitempty"`
	PostCareScore   int       `json:"post_care_score,omitempty"`
	PostCareSummary string    `json:"post_care_summary,omitempty"`
	PostCareAt      time.Time `json:"post_care_at,omitempty"`
	CycleDurationMs int64     `json:"cycle_duration_ms,omitempty"`
	ScoreDelta      int       `json:"score_delta,omitempty"`
	ActionsTaken    []string  `json:"actions_taken,omitempty"`
	RegressionFlags []string  `json:"regression_flags,omitempty"`
}

type SelfCareCyclesResponse struct {
	GeneratedAt time.Time       `json:"generated_at"`
	Cycles      []SelfCareCycle `json:"cycles"`
}

type DashboardResponse struct {
	GeneratedAt time.Time               `json:"generated_at"`
	Overview    *Overview               `json:"overview,omitempty"`
	Benchmarks  *BenchmarksResponse     `json:"benchmarks,omitempty"`
	Services    []ServiceStatus         `json:"services,omitempty"`
	Agents      []AgentStatus           `json:"agents,omitempty"`
	Stats24h    *systemdb.Stats24h      `json:"stats_24h,omitempty"`
	Events      []systemdb.Event        `json:"events,omitempty"`
	SelfCare    *SelfCareCyclesResponse `json:"self_care,omitempty"`
}

type JournalResponse struct {
	GeneratedAt time.Time              `json:"generated_at"`
	Entry       *systemdb.JournalEntry `json:"entry,omitempty"`
}

type JournalScheduleConfig struct {
	Enabled   bool      `json:"enabled"`
	CronExpr  string    `json:"cron_expr"` // Default: "50 23 * * *" (23:50 daily)
	Timezone  string    `json:"timezone"`  // Default: "UTC"
	LastRunAt time.Time `json:"last_run_at,omitempty"`
	NextRunAt time.Time `json:"next_run_at,omitempty"`
}

type JournalConfig struct {
	Enabled        bool   `json:"enabled"`         // Enable/disable journal generation
	RolloverHour   int    `json:"rollover_hour"`   // Hour of day to generate journal (0-23, default: 23)
	RolloverMinute int    `json:"rollover_minute"` // Minute of hour (0-59, default: 50)
	StyleMode      string `json:"style_mode"`      // Journal style: "dark_humor", "formal", "minimal"
	MaxLengthCap   int    `json:"max_length_cap"`  // Max journal body length in chars (0 = no cap)
}

type ObserverConfig struct {
	Journal JournalConfig `json:"journal"`
}

func DefaultObserverConfig() ObserverConfig {
	return ObserverConfig{
		Journal: JournalConfig{
			Enabled:        true,
			RolloverHour:   23,
			RolloverMinute: 50,
			StyleMode:      "dark_humor",
			MaxLengthCap:   2000,
		},
	}
}

// configFilePath returns the path to the observer config file
func (s *Store) configFilePath() string {
	base := ""
	if s.healthFile != "" {
		base = filepath.Dir(s.healthFile)
	}
	if base == "" {
		base = "."
	}
	return filepath.Join(base, "observer-config.json")
}

// LoadConfig loads the observer configuration from disk
func (s *Store) LoadConfig() (ObserverConfig, error) {
	configPath := s.configFilePath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultObserverConfig(), nil
		}
		return ObserverConfig{}, err
	}

	var config ObserverConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return DefaultObserverConfig(), nil
	}

	// Validate and apply defaults
	if config.Journal.StyleMode == "" {
		config.Journal.StyleMode = "dark_humor"
	}
	if config.Journal.RolloverHour < 0 || config.Journal.RolloverHour > 23 {
		config.Journal.RolloverHour = 23
	}
	if config.Journal.RolloverMinute < 0 || config.Journal.RolloverMinute > 59 {
		config.Journal.RolloverMinute = 50
	}
	if config.Journal.MaxLengthCap < 0 {
		config.Journal.MaxLengthCap = 2000
	}

	return config, nil
}

// SaveConfig saves the observer configuration to disk
func (s *Store) SaveConfig(config ObserverConfig) error {
	configPath := s.configFilePath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0o644)
}

// IsJournalEnabled returns whether journal generation is enabled
func (s *Store) IsJournalEnabled() bool {
	config, err := s.LoadConfig()
	if err != nil {
		return true // Default to enabled
	}
	return config.Journal.Enabled
}

// GetJournalConfig returns the current journal configuration
func (s *Store) GetJournalConfig() JournalConfig {
	config, err := s.LoadConfig()
	if err != nil {
		return DefaultObserverConfig().Journal
	}
	return config.Journal
}

// GenerateDailyJournalWithConfig generates a journal entry respecting the configuration
func (s *Store) GenerateDailyJournalWithConfig(now time.Time) (*systemdb.JournalEntry, error) {
	config, err := s.LoadConfig()
	if err != nil {
		config = DefaultObserverConfig()
	}

	if !config.Journal.Enabled {
		return nil, fmt.Errorf("journal generation is disabled")
	}

	entry, err := s.GenerateDailyJournal(now)
	if err != nil {
		return nil, err
	}

	// Apply max length cap if configured
	if config.Journal.MaxLengthCap > 0 && len(entry.Body) > config.Journal.MaxLengthCap {
		entry.Body = entry.Body[:config.Journal.MaxLengthCap-3] + "..."
	}

	return entry, nil
}

func NewStore(workspace, healthFile string) *Store {
	healthFile = strings.TrimSpace(healthFile)
	workspace = strings.TrimSpace(workspace)
	if healthFile == "" && workspace != "" {
		healthFile = filepath.Join(workspace, "state", "runtime-health.json")
	}
	baselineFile := ""
	systemDBFile := ""
	agentDBFile := ""
	if healthFile != "" {
		baselineFile = healthFile + ".baseline"
		systemDBFile = filepath.Join(filepath.Dir(healthFile), "system.db")
		agentDBFile = filepath.Join(filepath.Dir(healthFile), "agent.db")
	}
	return &Store{
		healthFile:   healthFile,
		baselineFile: baselineFile,
		systemDBFile: systemDBFile,
		agentDBFile:  agentDBFile,
	}
}

func (s *Store) Overview() (*Overview, error) {
	current, err := s.ReadCurrent()
	if err != nil {
		return nil, err
	}
	baseline, _ := s.ReadBaseline()

	out := &Overview{
		GeneratedAt:           time.Now().UTC(),
		Status:                current.Summary,
		Score:                 current.Score,
		Summary:               current.Summary,
		Deferred:              current.Deferred,
		ComparedTo:            current.ComparedTo,
		Recommendations:       current.Recommendations,
		LastRemediationAt:     current.LastRemediationAt,
		LastRemediationAction: current.LastRemediationAction,
		Services:              summarizeServices(current),
		CheapCognition:        current.CheapCognition,
	}
	if baseline != nil {
		out.Baseline = &BenchmarkSummary{
			Timestamp: baseline.Timestamp,
			Score:     baseline.Score,
			Summary:   baseline.Summary,
		}
	}
	if db, err := s.openSystemDB(); err == nil {
		defer db.Close()
		if services, err := db.CurrentServices(); err == nil && len(services) > 0 {
			out.Services = mapSystemServices(services)
		}
		if stats, err := db.Stats24h(); err == nil {
			out.Stats24h = &stats
		}
	}
	return out, nil
}

func (s *Store) Benchmarks() (*BenchmarksResponse, error) {
	current, err := s.ReadCurrent()
	if err != nil {
		return nil, err
	}
	baseline, _ := s.ReadBaseline()
	return &BenchmarksResponse{
		GeneratedAt: time.Now().UTC(),
		Current:     current,
		Baseline:    baseline,
	}, nil
}

func (s *Store) Services() ([]ServiceStatus, error) {
	if db, err := s.openSystemDB(); err == nil {
		defer db.Close()
		if services, err := db.CurrentServices(); err == nil && len(services) > 0 {
			return mapSystemServices(services), nil
		}
	}

	current, err := s.ReadCurrent()
	if err != nil {
		return nil, err
	}
	return summarizeServices(current), nil
}

func (s *Store) Stats24h() (*systemdb.Stats24h, error) {
	db, err := s.openSystemDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	stats, err := db.Stats24h()
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (s *Store) Events(limit int) (*EventsResponse, error) {
	db, err := s.openSystemDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	events, err := db.RecentEvents(limit)
	if err != nil {
		return nil, err
	}
	return &EventsResponse{
		GeneratedAt: time.Now().UTC(),
		Events:      events,
	}, nil
}

func (s *Store) Agents() ([]AgentStatus, error) {
	return s.AgentsFiltered("", "", "")
}

func (s *Store) AgentsFiltered(state, pipeline, manager string) ([]AgentStatus, error) {
	db, err := s.openAgentDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	agents, err := db.CurrentAgents()
	if err != nil {
		return nil, err
	}
	out := make([]AgentStatus, 0, len(agents))
	for _, item := range agents {
		if state != "" && !strings.EqualFold(item.State, state) {
			continue
		}
		if pipeline != "" && !strings.EqualFold(item.PipelineID, pipeline) {
			continue
		}
		if manager != "" && !strings.EqualFold(item.ManagerID, manager) {
			continue
		}
		out = append(out, AgentStatus{
			AgentID:         item.AgentID,
			AgentName:       item.AgentName,
			ManagerID:       item.ManagerID,
			PipelineID:      item.PipelineID,
			State:           item.State,
			Status:          item.Status,
			Channel:         item.Channel,
			ChatID:          item.ChatID,
			LastTaskSummary: item.LastTaskSummary,
			LastError:       item.LastError,
			LastSeenAt:      item.LastSeenAt,
			UpdatedAt:       item.UpdatedAt,
		})
	}
	return out, nil
}

func (s *Store) AgentSummary() (*AgentSummary, error) {
	agents, err := s.Agents()
	if err != nil {
		return nil, err
	}
	summary := &AgentSummary{
		GeneratedAt: time.Now().UTC(),
		Total:       len(agents),
		ByState:     map[string]int{},
		ByPipeline:  map[string]int{},
	}
	for _, agent := range agents {
		summary.ByState[agent.State]++
		pipeline := strings.TrimSpace(agent.PipelineID)
		if pipeline == "" {
			pipeline = "unassigned"
		}
		summary.ByPipeline[pipeline]++
	}
	return summary, nil
}

func (s *Store) SelfCareCycles(limit int) (*SelfCareCyclesResponse, error) {
	db, err := s.openSystemDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	snapshots, err := db.RecentSnapshots(limit)
	if err != nil {
		return nil, err
	}
	snapshotIDByTime := make(map[string]int, len(snapshots))
	for _, item := range snapshots {
		snapshotIDByTime[item.CreatedAt.Format(time.RFC3339)] = item.ID
	}
	cycles := make([]SelfCareCycle, 0, len(snapshots))
	for _, item := range snapshots {
		cycle := SelfCareCycle{
			ID:                    item.ID,
			Kind:                  item.Kind,
			Timestamp:             item.CreatedAt,
			Score:                 item.Score,
			Summary:               item.Summary,
			Deferred:              item.Deferred,
			ComparedTo:            item.ComparedTo,
			LastRemediationAt:     item.LastRemediationAt,
			LastRemediationAction: item.LastRemediationAction,
			PostCareSnapshotID:    item.ID,
		}

		// Parse enhanced cycle data from payload if available
		if item.PayloadJSON != "" {
			var payload map[string]any
			if err := json.Unmarshal([]byte(item.PayloadJSON), &payload); err == nil {
				// Extract baseline info
				if baselineScore, ok := payload["baseline_score"].(float64); ok {
					cycle.BaselineScore = int(baselineScore)
				}
				if baselineSummary, ok := payload["baseline_summary"].(string); ok {
					cycle.BaselineSummary = baselineSummary
				}
				if baselineAt, ok := payload["baseline_at"].(string); ok {
					if t, err := time.Parse(time.RFC3339, baselineAt); err == nil {
						cycle.BaselineAt = t
						cycle.BaselineSnapshotID = snapshotIDByTime[t.Format(time.RFC3339)]
					}
				}
				if baseline, ok := payload["baseline"].(bool); ok && baseline {
					if cycle.BaselineScore == 0 {
						cycle.BaselineScore = cycle.Score
					}
					if cycle.BaselineSummary == "" {
						cycle.BaselineSummary = cycle.Summary
					}
					if cycle.BaselineAt.IsZero() {
						cycle.BaselineAt = cycle.Timestamp
					}
					if cycle.BaselineSnapshotID == 0 {
						cycle.BaselineSnapshotID = item.ID
					}
				}

				// Extract pre-check info if available
				if preCheckScore, ok := payload["pre_check_score"].(float64); ok {
					cycle.PreCheckScore = int(preCheckScore)
				}
				if preCheckSummary, ok := payload["pre_check_summary"].(string); ok {
					cycle.PreCheckSummary = preCheckSummary
				}
				if preCheckAt, ok := payload["pre_check_at"].(string); ok {
					if t, err := time.Parse(time.RFC3339, preCheckAt); err == nil {
						cycle.PreCheckAt = t
						cycle.PreCheckSnapshotID = snapshotIDByTime[t.Format(time.RFC3339)]
					}
				}

				// Extract post-care info if available
				if postCareScore, ok := payload["post_care_score"].(float64); ok {
					cycle.PostCareScore = int(postCareScore)
				}
				if postCareSummary, ok := payload["post_care_summary"].(string); ok {
					cycle.PostCareSummary = postCareSummary
				}
				if postCareAt, ok := payload["post_care_at"].(string); ok {
					if t, err := time.Parse(time.RFC3339, postCareAt); err == nil {
						cycle.PostCareAt = t
						if cycle.PostCareSnapshotID == 0 {
							cycle.PostCareSnapshotID = snapshotIDByTime[t.Format(time.RFC3339)]
						}
					}
				}

				// Extract cycle duration
				if durationMs, ok := payload["cycle_duration_ms"].(float64); ok {
					cycle.CycleDurationMs = int64(durationMs)
				}

				// Extract score delta
				if scoreDelta, ok := payload["score_delta"].(float64); ok {
					cycle.ScoreDelta = int(scoreDelta)
				}

				// Extract actions taken
				if actions, ok := payload["actions_taken"].([]any); ok {
					for _, action := range actions {
						if actionStr, ok := action.(string); ok {
							cycle.ActionsTaken = append(cycle.ActionsTaken, actionStr)
						}
					}
				}
				if flags, ok := payload["regression_flags"].([]any); ok {
					for _, flag := range flags {
						if flagStr, ok := flag.(string); ok {
							cycle.RegressionFlags = append(cycle.RegressionFlags, flagStr)
						}
					}
				}
			}
		}
		if cycle.PostCareSnapshotID == 0 {
			cycle.PostCareSnapshotID = item.ID
		}

		cycles = append(cycles, cycle)
	}
	return &SelfCareCyclesResponse{
		GeneratedAt: time.Now().UTC(),
		Cycles:      cycles,
	}, nil
}

func (s *Store) Dashboard(limit int) (*DashboardResponse, error) {
	overview, err := s.Overview()
	if err != nil {
		return nil, err
	}
	benchmarks, err := s.Benchmarks()
	if err != nil {
		return nil, err
	}
	services, err := s.Services()
	if err != nil {
		return nil, err
	}
	agents, _ := s.Agents()
	stats, _ := s.Stats24h()
	events, _ := s.Events(limit)
	cycles, _ := s.SelfCareCycles(limit)

	resp := &DashboardResponse{
		GeneratedAt: time.Now().UTC(),
		Overview:    overview,
		Benchmarks:  benchmarks,
		Services:    services,
		Agents:      agents,
		Stats24h:    stats,
		SelfCare:    cycles,
	}
	if events != nil {
		resp.Events = events.Events
	}
	return resp, nil
}

func (s *Store) GenerateDailyJournal(now time.Time) (*systemdb.JournalEntry, error) {
	db, err := s.openSystemDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	now = now.UTC()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)
	dateKey := dayStart.Format("2006-01-02")

	events, err := db.EventsBetween(dayStart, dayEnd, 500)
	if err != nil {
		return nil, err
	}
	services, err := db.CurrentServices()
	if err != nil {
		return nil, err
	}
	stats, err := db.Stats24h()
	if err != nil {
		return nil, err
	}
	cycles, err := db.RecentSnapshots(24)
	if err != nil {
		return nil, err
	}

	entry := buildDailyJournal(dateKey, events, services, stats, cycles)
	if err := db.UpsertJournal(entry); err != nil {
		return nil, err
	}
	return db.JournalByDate(dateKey)
}

func (s *Store) LatestJournal() (*JournalResponse, error) {
	db, err := s.openSystemDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	entry, err := db.LatestJournal()
	if err != nil {
		return nil, err
	}
	return &JournalResponse{
		GeneratedAt: time.Now().UTC(),
		Entry:       entry,
	}, nil
}

// JournalScheduleMessage returns the message to be used for cron job scheduling
func (s *Store) JournalScheduleMessage() string {
	return "Generate daily observer journal"
}

// JournalDayData holds raw observer data for one day, used by the journal agent
// to build its LLM prompt.
type JournalDayData struct {
	DateKey       string                    `json:"date_key"`
	Events        []systemdb.Event          `json:"events"`
	Services      []systemdb.ServiceStatus  `json:"services"`
	Stats         systemdb.Stats24h         `json:"stats"`
	Cycles        []systemdb.SnapshotRecord `json:"cycles"`
	OfflineCount  int                       `json:"offline_count"`
	DegradedCount int                       `json:"degraded_count"`
	RestartCount  int                       `json:"restart_count"`
	ErrorCount    int                       `json:"error_count"`
	CriticalCount int                       `json:"critical_count"`
	Score         int                       `json:"score"`
	ScoreSummary  string                    `json:"score_summary"`
}

// CollectJournalDayData gathers all raw observer data for the given UTC date.
func (s *Store) CollectJournalDayData(dateKey string) (*JournalDayData, error) {
	dayStart, err := time.Parse("2006-01-02", dateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid date key: %w", err)
	}
	dayEnd := dayStart.Add(24 * time.Hour)

	db, err := s.openSystemDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	events, _ := db.EventsBetween(dayStart, dayEnd, 500)
	services, _ := db.CurrentServices()
	stats, _ := db.Stats24h()
	cycles, _ := db.RecentSnapshots(24)

	data := &JournalDayData{
		DateKey:  dateKey,
		Events:   events,
		Services: services,
		Stats:    stats,
		Cycles:   cycles,
	}

	for _, svc := range services {
		switch svc.State {
		case "offline":
			data.OfflineCount++
		case "degraded", "restarting":
			data.DegradedCount++
		}
	}

	for _, e := range events {
		switch e.EventType {
		case "restart_requested", "restart_failed":
			data.RestartCount++
		}
		if e.Severity == "critical" {
			data.CriticalCount++
		}
		if e.Severity == "error" {
			data.ErrorCount++
		}
	}

	if len(cycles) > 0 {
		data.Score = cycles[0].Score
		data.ScoreSummary = cycles[0].Summary
	}

	return data, nil
}

// SaveJournal persists a journal entry to the system database.
func (s *Store) SaveJournal(entry systemdb.JournalEntry) error {
	db, err := s.openSystemDB()
	if err != nil {
		return err
	}
	defer db.Close()
	return db.UpsertJournal(entry)
}

// DefaultJournalSchedule returns the default schedule configuration for journal generation
// Default: 23:50 UTC daily (near day rollover)
func DefaultJournalSchedule() JournalScheduleConfig {
	return JournalScheduleConfig{
		Enabled:  true,
		CronExpr: "50 23 * * *", // 23:50 daily
		Timezone: "UTC",
	}
}

// ClearEventsResult represents the result of clearing old events
type ClearEventsResult struct {
	DeletedCount  int       `json:"deleted_count"`
	RetentionDays int       `json:"retention_days"`
	ClearedAt     time.Time `json:"cleared_at"`
}

// ClearOldEvents removes events older than the specified retention period
// This is a bounded operation: max retention is 90 days
func (s *Store) ClearOldEvents(retentionDays int) (*ClearEventsResult, error) {
	// Bound the retention period
	if retentionDays <= 0 {
		retentionDays = 30 // Default: 30 days
	}
	if retentionDays > 90 {
		retentionDays = 90 // Max: 90 days
	}

	db, err := s.openSystemDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
	deleted, err := db.DeleteEventsBefore(cutoff)
	if err != nil {
		return nil, err
	}

	return &ClearEventsResult{
		DeletedCount:  deleted,
		RetentionDays: retentionDays,
		ClearedAt:     time.Now().UTC(),
	}, nil
}

// ResetBaselineResult represents the result of resetting the baseline
type ResetBaselineResult struct {
	PreviousBaseline *BenchmarkSummary `json:"previous_baseline,omitempty"`
	NewBaseline      *BenchmarkSummary `json:"new_baseline"`
	ResetAt          time.Time         `json:"reset_at"`
}

// ResetBaseline resets the health baseline to the current health state
func (s *Store) ResetBaseline() (*ResetBaselineResult, error) {
	// Read current baseline
	previousBaseline, _ := s.ReadBaseline()

	// Read current health state
	current, err := s.ReadCurrent()
	if err != nil {
		return nil, fmt.Errorf("cannot read current health state: %w", err)
	}

	// Write current state as new baseline
	if err := s.writeBaseline(*current); err != nil {
		return nil, fmt.Errorf("cannot write new baseline: %w", err)
	}

	result := &ResetBaselineResult{
		NewBaseline: &BenchmarkSummary{
			Timestamp: current.Timestamp,
			Score:     current.Score,
			Summary:   current.Summary,
		},
		ResetAt: time.Now().UTC(),
	}

	if previousBaseline != nil {
		result.PreviousBaseline = &BenchmarkSummary{
			Timestamp: previousBaseline.Timestamp,
			Score:     previousBaseline.Score,
			Summary:   previousBaseline.Summary,
		}
	}

	return result, nil
}

// writeBaseline writes a health snapshot as the baseline
func (s *Store) writeBaseline(snapshot maintenance.HealthSnapshot) error {
	if s.baselineFile == "" {
		return ErrHealthFileNotSet
	}
	if err := os.MkdirAll(filepath.Dir(s.baselineFile), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.baselineFile, data, 0o644)
}

type ReportInfo struct {
	GeneratedAt  time.Time `json:"generated_at"`
	Path         string    `json:"path"`
	Filename     string    `json:"filename"`
	DownloadPath string    `json:"download_path"`
}

func (s *Store) GenerateHTMLReport(limit int) (*ReportInfo, error) {
	dashboard, err := s.Dashboard(limit)
	if err != nil {
		return nil, err
	}
	reportDir := s.reportDir()
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return nil, err
	}
	filename := fmt.Sprintf("observer-report-%s.html", time.Now().UTC().Format("20060102-150405"))
	path := filepath.Join(reportDir, filename)

	tpl := template.Must(template.New("report").Parse(observerReportTemplate))
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, dashboard); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return nil, err
	}
	return &ReportInfo{
		GeneratedAt:  time.Now().UTC(),
		Path:         path,
		Filename:     filename,
		DownloadPath: "/observer/reports/latest?format=html",
	}, nil
}

func (s *Store) LatestReport() (*ReportInfo, error) {
	reportDir := s.reportDir()
	entries, err := os.ReadDir(reportDir)
	if err != nil {
		return nil, err
	}
	var latest os.DirEntry
	var latestInfo os.FileInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if latest == nil || info.ModTime().After(latestInfo.ModTime()) {
			latest = entry
			latestInfo = info
		}
	}
	if latest == nil {
		return nil, os.ErrNotExist
	}
	path := filepath.Join(reportDir, latest.Name())
	return &ReportInfo{
		GeneratedAt:  latestInfo.ModTime().UTC(),
		Path:         path,
		Filename:     latest.Name(),
		DownloadPath: "/observer/reports/latest?format=html",
	}, nil
}

func (s *Store) LatestReportHTML() ([]byte, *ReportInfo, error) {
	info, err := s.LatestReport()
	if err != nil {
		return nil, nil, err
	}
	data, err := os.ReadFile(info.Path)
	if err != nil {
		return nil, nil, err
	}
	return data, info, nil
}

func (s *Store) ReadCurrent() (*maintenance.HealthSnapshot, error) {
	return s.readSnapshot(s.healthFile)
}

func (s *Store) ReadBaseline() (*maintenance.HealthSnapshot, error) {
	return s.readSnapshot(s.baselineFile)
}

func (s *Store) readSnapshot(path string) (*maintenance.HealthSnapshot, error) {
	if path == "" {
		return nil, ErrHealthFileNotSet
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var snapshot maintenance.HealthSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (s *Store) openSystemDB() (*systemdb.Store, error) {
	if s.systemDBFile == "" {
		return nil, ErrHealthFileNotSet
	}
	return systemdb.Open(s.systemDBFile)
}

func (s *Store) openAgentDB() (*agentdb.Store, error) {
	if s.agentDBFile == "" {
		return nil, ErrHealthFileNotSet
	}
	return agentdb.Open(s.agentDBFile)
}

func (s *Store) reportDir() string {
	base := ""
	if s.healthFile != "" {
		base = filepath.Dir(s.healthFile)
	}
	if base == "" {
		base = "."
	}
	return filepath.Join(base, "reports")
}

func buildDailyJournal(
	dateKey string,
	events []systemdb.Event,
	services []systemdb.ServiceStatus,
	stats systemdb.Stats24h,
	cycles []systemdb.SnapshotRecord,
) systemdb.JournalEntry {
	offline := 0
	degraded := 0
	restarts := 0
	riots := 0
	for _, svc := range services {
		switch svc.State {
		case "offline":
			offline++
		case "degraded", "restarting":
			degraded++
		}
	}
	for _, event := range events {
		switch event.EventType {
		case "restart_requested", "restart_failed":
			restarts++
		}
		if event.Severity == "critical" || event.Severity == "error" {
			riots++
		}
	}

	lastScore := 0
	lastSummary := "quiet"
	if len(cycles) > 0 {
		lastScore = cycles[0].Score
		lastSummary = cycles[0].Summary
	}

	title := "Quiet shifts and civilized machinery"
	switch {
	case offline+degraded >= 3:
		title = "Riot control and mutiny suppression"
	case restarts >= 2:
		title = "Slackers, restarts, and restored order"
	case stats.CriticalEvents > 0 || stats.ErrorEvents > 0:
		title = "Minor degeneracy with acceptable recovery"
	}

	opening := "Today it was almost suspiciously calm."
	switch {
	case offline+degraded >= 3:
		opening = "Today the agents tried to stage a proper mutiny."
	case offline+degraded == 2:
		opening = "Today a pair of slackers tried to turn routine operations into a riot."
	case offline+degraded == 1:
		opening = "Today it all was smooth sailing until one nuisance decided peace was overrated."
	case stats.ErrorEvents > 0 || restarts > 0:
		opening = "Today had the usual background grumbling from a few degenerates, but nothing civilization could not correct."
	}

	middle := "The observer kept notes, kicked the worst offenders back into line, and preserved enough dignity to avoid calling it a catastrophe."
	switch {
	case restarts >= 3:
		middle = "After a few forced attitude adjustments and more restart requests than any polite society should need, the machinery stopped acting like a tavern brawl."
	case restarts > 0:
		middle = "A few strategic kicks in the form of restart requests reminded the troublemakers that uptime is not a democratic choice."
	case stats.WarningEvents > 0:
		middle = "There was grumbling around the edges, but it stayed at nuisance level rather than escalating into a full-blown labor dispute."
	}

	ending := fmt.Sprintf("By the end of the shift, the system settled at score %d with a %s mood. If no one starts a new world war before midnight, we may even call that respectable.", lastScore, lastSummary)
	if lastScore == 0 {
		ending = "The day ended without enough evidence for a proper verdict, which is still preferable to open rebellion."
	}

	body := strings.Join([]string{opening, middle, ending}, " ")
	return systemdb.JournalEntry{
		Date:      dateKey,
		Title:     title,
		Body:      body,
		Style:     "dark_humor",
		CreatedAt: time.Now().UTC(),
	}
}

const observerReportTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Spiderweb Observer Report</title>
  <style>
    body { font-family: "Segoe UI", Arial, sans-serif; margin: 0; background: #0d0f12; color: #f4f1ed; }
    .wrap { max-width: 1100px; margin: 0 auto; padding: 28px; }
    .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 16px; margin-bottom: 20px; }
    .card { background: rgba(255,255,255,0.05); border: 1px solid rgba(255,255,255,0.12); border-radius: 18px; padding: 16px; }
    h1,h2 { margin: 0 0 12px; }
    h1 { font-size: 2rem; }
    h2 { font-size: 1.1rem; color: #ffbc84; }
    .muted { color: #cfc6bf; }
    table { width: 100%; border-collapse: collapse; }
    th, td { text-align: left; padding: 10px 8px; border-bottom: 1px solid rgba(255,255,255,0.08); vertical-align: top; }
    th { color: #ffbc84; font-weight: 600; }
    .pill { display: inline-block; padding: 4px 10px; border-radius: 999px; background: rgba(255,255,255,0.08); }
  </style>
</head>
<body>
  <div class="wrap">
    <h1>Spiderweb Observer Report</h1>
    <p class="muted">Generated at {{ .GeneratedAt.Format "2006-01-02 15:04:05 MST" }}</p>
    {{ if .Overview }}
    <div class="grid">
      <div class="card"><h2>Status</h2><div class="pill">{{ .Overview.Status }}</div><p>Score: {{ .Overview.Score }}</p><p>{{ .Overview.Summary }}</p></div>
      <div class="card"><h2>24h Stats</h2>{{ if .Stats24h }}<p>Total events: {{ .Stats24h.TotalEvents }}</p><p>Errors: {{ .Stats24h.ErrorEvents }}</p><p>Critical: {{ .Stats24h.CriticalEvents }}</p>{{ else }}<p>No stats recorded.</p>{{ end }}</div>
      <div class="card"><h2>Services</h2><p>{{ len .Services }} tracked</p><p>Agents: {{ len .Agents }}</p></div>
    </div>
    {{ end }}
    <div class="card">
      <h2>Services</h2>
      <table><thead><tr><th>Name</th><th>State</th><th>Message</th></tr></thead><tbody>
      {{ range .Services }}<tr><td>{{ .Name }}</td><td>{{ .State }}</td><td>{{ .Message }}</td></tr>{{ end }}
      </tbody></table>
    </div>
    <div class="card">
      <h2>Agents</h2>
      <table><thead><tr><th>Agent</th><th>Pipeline</th><th>State</th><th>Last task</th></tr></thead><tbody>
      {{ range .Agents }}<tr><td>{{ .AgentName }}</td><td>{{ .PipelineID }}</td><td>{{ .State }}</td><td>{{ .LastTaskSummary }}</td></tr>{{ end }}
      </tbody></table>
    </div>
    <div class="card">
      <h2>Recent Events</h2>
      <table><thead><tr><th>Time</th><th>Severity</th><th>Type</th><th>Message</th></tr></thead><tbody>
      {{ range .Events }}<tr><td>{{ .CreatedAt.Format "15:04:05" }}</td><td>{{ .Severity }}</td><td>{{ .EventType }}</td><td>{{ .Message }}</td></tr>{{ end }}
      </tbody></table>
    </div>
    <div class="card">
      <h2>Self-Care History</h2>
      {{ if .SelfCare }}
      <table><thead><tr><th>Time</th><th>Kind</th><th>Score</th><th>Summary</th><th>Action</th></tr></thead><tbody>
      {{ range .SelfCare.Cycles }}<tr><td>{{ .Timestamp.Format "2006-01-02 15:04:05" }}</td><td>{{ .Kind }}</td><td>{{ .Score }}</td><td>{{ .Summary }}</td><td>{{ .LastRemediationAction }}</td></tr>{{ end }}
      </tbody></table>
      {{ else }}<p>No self-care history.</p>{{ end }}
    </div>
  </div>
</body>
</html>`

func summarizeServices(snapshot *maintenance.HealthSnapshot) []ServiceStatus {
	if snapshot == nil || len(snapshot.Processes) == 0 {
		return nil
	}

	out := make([]ServiceStatus, 0, len(snapshot.Processes))
	for id, proc := range snapshot.Processes {
		state := "offline"
		if proc.Running {
			state = "online"
		}
		if proc.Action == "restart_requested" {
			state = "restarting"
		} else if proc.Action == "restart_failed" {
			state = "degraded"
		}
		out = append(out, ServiceStatus{
			ID:      id,
			Name:    proc.Name,
			State:   state,
			Running: proc.Running,
			Owned:   proc.Owned,
			Action:  proc.Action,
			Message: proc.Message,
			PIDFile: proc.PIDFile,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func mapSystemServices(in []systemdb.ServiceStatus) []ServiceStatus {
	out := make([]ServiceStatus, 0, len(in))
	for _, item := range in {
		out = append(out, ServiceStatus{
			ID:        item.ID,
			Name:      item.Name,
			State:     item.State,
			Running:   item.Running,
			Owned:     item.Owned,
			Action:    item.Action,
			Message:   item.Message,
			PIDFile:   item.PIDFile,
			UpdatedAt: item.UpdatedAt,
		})
	}
	return out
}
