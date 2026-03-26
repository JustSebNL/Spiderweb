package maintenance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/JustSebNL/Spiderweb/pkg/config"
	"github.com/JustSebNL/Spiderweb/pkg/logger"
	"github.com/JustSebNL/Spiderweb/pkg/systemdb"
)

type RuntimeController interface {
	Start() error
}

type RuntimeStats interface {
	CheapCognitionSnapshot() CheapCognitionSnapshot
}

type CheapCognitionSnapshot struct {
	ClassificationCalls int64
	ClassificationFails int64
	Forwarded           int64
	Skipped             int64
	LastLatencyMS       int64
	LastError           string
	LastDecisionAt      time.Time
}

type ProcessState struct {
	Name    string `json:"name"`
	PIDFile string `json:"pid_file,omitempty"`
	Running bool   `json:"running"`
	Owned   bool   `json:"owned"`
	Action  string `json:"action,omitempty"`
	Message string `json:"message,omitempty"`
}

type HealthSnapshot struct {
	Timestamp             time.Time               `json:"timestamp"`
	Score                 int                     `json:"score"`
	Summary               string                  `json:"summary"`
	Baseline              bool                    `json:"baseline,omitempty"`
	Deferred              bool                    `json:"deferred,omitempty"`
	DeferReason           string                  `json:"defer_reason,omitempty"`
	ComparedTo            string                  `json:"compared_to,omitempty"`
	Recommendations       []string                `json:"recommendations,omitempty"`
	Processes             map[string]ProcessState `json:"processes,omitempty"`
	CheapCognition        CheapCognitionSnapshot  `json:"cheap_cognition"`
	LastRemediationAt     time.Time               `json:"last_remediation_at,omitempty"`
	LastRemediationAction string                  `json:"last_remediation_action,omitempty"`
	BaselineScore         int                     `json:"baseline_score,omitempty"`
	BaselineSummary       string                  `json:"baseline_summary,omitempty"`
	BaselineAt            time.Time               `json:"baseline_at,omitempty"`
	PreCheckScore         int                     `json:"pre_check_score,omitempty"`
	PreCheckSummary       string                  `json:"pre_check_summary,omitempty"`
	PreCheckAt            time.Time               `json:"pre_check_at,omitempty"`
	PostCareScore         int                     `json:"post_care_score,omitempty"`
	PostCareSummary       string                  `json:"post_care_summary,omitempty"`
	PostCareAt            time.Time               `json:"post_care_at,omitempty"`
	CycleDurationMs       int64                   `json:"cycle_duration_ms,omitempty"`
	ScoreDelta            int                     `json:"score_delta,omitempty"`
	ActionsTaken          []string                `json:"actions_taken,omitempty"`
	RegressionFlags       []string                `json:"regression_flags,omitempty"`
}

type Service struct {
	cfg          config.MaintenanceConfig
	cheapCfg     config.CheapCognitionConfig
	triggerCfg   config.TriggerConfig
	workspace    string
	healthFile   string
	baselineFile string
	systemDBFile string
	runtimeStats RuntimeStats
	triggerCtl   RuntimeController
	brainCtl     RuntimeController
	mu           sync.RWMutex
	stopChan     chan struct{}
	runningCycle atomic.Bool
	startupDelay time.Duration
}

func NewService(
	workspace string,
	cfg config.MaintenanceConfig,
	cheapCfg config.CheapCognitionConfig,
	triggerCfg config.TriggerConfig,
	runtimeStats RuntimeStats,
	triggerCtl RuntimeController,
	brainCtl RuntimeController,
) *Service {
	if cfg.IntervalHours <= 0 {
		cfg.IntervalHours = 12
	}
	if cfg.BudgetPercent <= 0 {
		cfg.BudgetPercent = 5
	}
	if cfg.BusyWindowMinutes <= 0 {
		cfg.BusyWindowMinutes = 10
	}
	if cfg.RestartBackoffMinutes <= 0 {
		cfg.RestartBackoffMinutes = 30
	}
	if cfg.MaxLogMB <= 0 {
		cfg.MaxLogMB = 16
	}
	if cfg.HighLatencyMs <= 0 {
		cfg.HighLatencyMs = 2500
	}
	if cfg.MaxCheapFailures <= 0 {
		cfg.MaxCheapFailures = 3
	}
	if cfg.MaxForwardSkips <= 0 {
		cfg.MaxForwardSkips = 500
	}
	healthFile := expandHome(cfg.HealthFile)
	if healthFile == "" {
		healthFile = filepath.Join(workspace, "state", "runtime-health.json")
	}
	return &Service{
		cfg:          cfg,
		cheapCfg:     cheapCfg,
		triggerCfg:   triggerCfg,
		workspace:    workspace,
		healthFile:   healthFile,
		baselineFile: healthFile + ".baseline",
		systemDBFile: filepath.Join(filepath.Dir(healthFile), "system.db"),
		runtimeStats: runtimeStats,
		triggerCtl:   triggerCtl,
		brainCtl:     brainCtl,
		startupDelay: 5 * time.Second,
	}
}

func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopChan != nil || !s.cfg.Enabled {
		return nil
	}
	s.stopChan = make(chan struct{})
	go s.runLoop(s.stopChan)
	logger.InfoCF("maintenance", "Maintenance service started", map[string]any{"interval_hours": s.cfg.IntervalHours})
	return nil
}

func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopChan == nil {
		return
	}
	close(s.stopChan)
	s.stopChan = nil
}

func (s *Service) runLoop(stopChan chan struct{}) {
	interval := time.Duration(s.cfg.IntervalHours) * time.Hour
	ticker := time.NewTicker(interval)
	startup := time.NewTimer(s.startupDelay)
	defer ticker.Stop()
	defer startup.Stop()
	for {
		select {
		case <-stopChan:
			return
		case <-startup.C:
			s.RunOnce(context.Background())
		case <-ticker.C:
			s.RunOnce(context.Background())
		}
	}
}

func (s *Service) RunOnce(ctx context.Context) {
	if !s.cfg.Enabled {
		return
	}
	if !s.runningCycle.CompareAndSwap(false, true) {
		logger.InfoC("maintenance", "Skipping overlapping maintenance run")
		return
	}
	defer s.runningCycle.Store(false)

	baseline, _ := s.readBaseline()
	previous, _ := s.readSnapshot()
	activeProbe := baseline == nil
	if activeProbe {
		logger.InfoCF("maintenance", "Creating startup runtime baseline", map[string]any{"budget_percent": s.cfg.BudgetPercent})
	}
	snapshot := s.collectSnapshot(ctx, baseline, previous, activeProbe)
	if activeProbe {
		snapshot.Baseline = true
		snapshot.ComparedTo = "startup"
		if err := s.writeBaseline(snapshot); err != nil {
			logger.WarnCF("maintenance", "Failed to write runtime baseline", map[string]any{"error": err.Error()})
		}
	}
	if err := s.writeSnapshot(snapshot); err != nil {
		logger.WarnCF("maintenance", "Failed to write health snapshot", map[string]any{"error": err.Error()})
		return
	}
	if err := s.writeSystemRecord(snapshot, baseline); err != nil {
		logger.WarnCF("maintenance", "Failed to write observer system record", map[string]any{"error": err.Error()})
	}
	logger.InfoCF("maintenance", "Runtime self-check completed", map[string]any{"score": snapshot.Score, "summary": snapshot.Summary, "baseline": snapshot.Baseline})
}

func (s *Service) collectSnapshot(ctx context.Context, baseline *HealthSnapshot, previous *HealthSnapshot, activeProbe bool) HealthSnapshot {
	startedAt := time.Now().UTC()
	now := time.Now().UTC()
	snapshot := HealthSnapshot{
		Timestamp:       now,
		Score:           100,
		Processes:       map[string]ProcessState{},
		Recommendations: []string{},
	}
	if s.runtimeStats != nil {
		snapshot.CheapCognition = s.runtimeStats.CheapCognitionSnapshot()
	}
	if previous != nil {
		snapshot.LastRemediationAt = previous.LastRemediationAt
		snapshot.LastRemediationAction = previous.LastRemediationAction
	}

	cleanupBudget := s.maintenanceBudgetSlots()
	deferMaintenance, deferReason := s.shouldDeferMaintenance(snapshot.CheapCognition, previous, activeProbe)
	if deferMaintenance {
		snapshot.Deferred = true
		snapshot.DeferReason = deferReason
		snapshot.Recommendations = append(snapshot.Recommendations, deferReason)
	}
	if s.cheapCfg.Enabled {
		state, penalty, notes := s.inspectCheapRuntime(ctx, activeProbe, deferMaintenance, previous, &cleanupBudget)
		snapshot.Processes[state.Name] = state
		snapshot.Score -= penalty
		snapshot.Recommendations = append(snapshot.Recommendations, notes...)
		snapshot = mergeRemediation(snapshot, state)
		snapshot.Score -= s.scoreCheapCognition(snapshot.CheapCognition, &snapshot.Recommendations)
	}

	if s.triggerCfg.Enabled {
		state, penalty, notes := s.inspectTriggerProcess(deferMaintenance, previous, &cleanupBudget)
		snapshot.Processes[state.Name] = state
		snapshot.Score -= penalty
		snapshot.Recommendations = append(snapshot.Recommendations, notes...)
		snapshot = mergeRemediation(snapshot, state)
	}

	if baseline != nil {
		snapshot.ComparedTo = baseline.Timestamp.Format(time.RFC3339)
		snapshot.Score -= compareToBaseline(*baseline, snapshot, &snapshot.Recommendations)
	}
	if snapshot.Score < 0 {
		snapshot.Score = 0
	}
	snapshot.Summary = summarizeScore(snapshot.Score)
	s.enrichCycleMetadata(&snapshot, baseline, previous, startedAt)
	return snapshot
}

func (s *Service) inspectCheapRuntime(ctx context.Context, activeProbe bool, deferMaintenance bool, previous *HealthSnapshot, cleanupBudget *int) (ProcessState, int, []string) {
	runtime := strings.ToLower(strings.TrimSpace(s.cheapCfg.Runtime))
	if runtime == "" || runtime == "auto" {
		runtime = "vllm"
	}
	state := ProcessState{Name: "cheap_cognition_" + runtime, Owned: true}
	notes := []string{}
	penalty := 0

	switch runtime {
	case "vllm":
		pidFile := envOrDefault("BRAIN_VLLM_PID_FILE", envOrDefault("YOUTU_VLLM_PID_FILE", filepath.Join(envOrDefault("BRAIN_DIR", filepath.Join(filepath.Dir(s.workspace), "brain")), "brain-vllm.pid")))
		state.PIDFile = pidFile
		state.Running = pidAlive(pidFile)
		if !state.Running {
			penalty += 25
			notes = append(notes, "Cheap cognition vLLM process is not running")
			if s.cfg.AutoRemediate && !deferMaintenance {
				if trim, ok := removeStalePID(pidFile, cleanupBudget); ok {
					state.Action = trim
				}
			}
			if s.cfg.AutoRemediate && s.cfg.RestartOnProcessDeath && s.brainCtl != nil && !deferMaintenance && hasBudget(cleanupBudget) {
				if backedOff, reason := s.restartBackoffActive(previous); backedOff {
					notes = append(notes, reason)
				} else if !consumeBudget(cleanupBudget) {
					notes = append(notes, "Maintenance budget exhausted before requesting vLLM restart")
				} else if err := s.brainCtl.Start(); err != nil {
					state.Action = "restart_failed"
					state.Message = err.Error()
					notes = append(notes, "Automatic vLLM restart failed")
				} else {
					state.Action = "restart_requested"
					state.Message = "Requested native vLLM restart"
				}
			}
		}
		if activeProbe {
			if latency, ok := s.measureCheapLatency(ctx); ok && latency > 0 {
				state.Message = fmt.Sprintf("startup_latency_ms=%d", latency)
				if latency > int64(s.cfg.HighLatencyMs) {
					penalty += 15
					notes = append(notes, fmt.Sprintf("Cheap cognition startup latency is high (%dms)", latency))
				}
			}
		}
		if !deferMaintenance {
			if trimmed, ok := trimLogIfNeeded(envOrDefault("BRAIN_VLLM_LOG_FILE", envOrDefault("YOUTU_VLLM_LOG_FILE", filepath.Join(envOrDefault("BRAIN_DIR", filepath.Join(filepath.Dir(s.workspace), "brain")), "brain-vllm.log"))), s.cfg.MaxLogMB, cleanupBudget); ok {
				notes = append(notes, trimmed)
			}
		}
	case "llama_cpp":
		pidFile := filepath.Join(envOrDefault("BRAIN_DIR", filepath.Join(filepath.Dir(s.workspace), "brain")), "llama-server.pid")
		state.PIDFile = pidFile
		state.Running = pidAlive(pidFile)
		if !state.Running {
			penalty += 20
			notes = append(notes, "Cheap cognition llama.cpp process is not running")
			if s.cfg.AutoRemediate && !deferMaintenance {
				if action, ok := removeStalePID(pidFile, cleanupBudget); ok {
					state.Action = action
				}
			}
		}
		if !deferMaintenance {
			if trimmed, ok := trimLogIfNeeded(filepath.Join(envOrDefault("BRAIN_DIR", filepath.Join(filepath.Dir(s.workspace), "brain")), "llama-server.log"), s.cfg.MaxLogMB, cleanupBudget); ok {
				notes = append(notes, trimmed)
			}
		}
	default:
		state.Running = false
		penalty += 20
		notes = append(notes, "Cheap cognition runtime is unknown")
	}

	return state, penalty, notes
}

func (s *Service) inspectTriggerProcess(deferMaintenance bool, previous *HealthSnapshot, cleanupBudget *int) (ProcessState, int, []string) {
	workdir := expandHome(s.triggerCfg.Workdir)
	if workdir == "" {
		workdir = filepath.Join(filepath.Dir(s.workspace), "trigger")
	}
	pidFile := expandHome(s.triggerCfg.PIDFile)
	if pidFile == "" {
		pidFile = filepath.Join(workdir, ".trigger.pid")
	}
	state := ProcessState{Name: "trigger", PIDFile: pidFile, Owned: true}
	notes := []string{}
	penalty := 0
	state.Running = pidAlive(pidFile)
	if !state.Running {
		penalty += 10
		notes = append(notes, "Optional Trigger worker is not running")
		if s.cfg.AutoRemediate && !deferMaintenance {
			if action, ok := removeStalePID(pidFile, cleanupBudget); ok {
				state.Action = action
			}
		}
		if s.cfg.AutoRemediate && s.cfg.RestartOnProcessDeath && s.triggerCtl != nil && s.triggerCfg.AutoStart && !deferMaintenance && hasBudget(cleanupBudget) {
			if backedOff, reason := s.restartBackoffActive(previous); backedOff {
				notes = append(notes, reason)
			} else if !consumeBudget(cleanupBudget) {
				notes = append(notes, "Maintenance budget exhausted before requesting Trigger restart")
			} else if err := s.triggerCtl.Start(); err != nil {
				state.Action = "restart_failed"
				state.Message = err.Error()
			} else {
				state.Action = "restart_requested"
				state.Message = "Requested Trigger restart"
			}
		}
	}
	if !deferMaintenance {
		if trimmed, ok := trimLogIfNeeded(expandHome(s.triggerCfg.LogFile), s.cfg.MaxLogMB, cleanupBudget); ok {
			notes = append(notes, trimmed)
		}
	}
	return state, penalty, notes
}

func (s *Service) measureCheapLatency(ctx context.Context) (int64, bool) {
	if strings.TrimSpace(s.cheapCfg.BaseURL) == "" {
		return 0, false
	}
	measureCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	start := time.Now()
	req, err := http.NewRequestWithContext(measureCtx, http.MethodGet, strings.TrimRight(s.cheapCfg.BaseURL, "/")+"/models", nil)
	if err != nil {
		return 0, false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, false
	}
	_ = resp.Body.Close()
	return time.Since(start).Milliseconds(), true
}

func (s *Service) scoreCheapCognition(stats CheapCognitionSnapshot, recommendations *[]string) int {
	penalty := 0
	if stats.ClassificationFails >= int64(s.cfg.MaxCheapFailures) {
		penalty += 20
		*recommendations = append(*recommendations, "Cheap cognition is failing repeatedly")
	}
	if stats.Skipped >= int64(s.cfg.MaxForwardSkips) {
		penalty += 10
		*recommendations = append(*recommendations, "Cheap cognition is skipping a high number of forwarded messages")
	}
	if stats.LastLatencyMS > int64(s.cfg.HighLatencyMs) {
		penalty += 10
		*recommendations = append(*recommendations, "Cheap cognition last observed latency is above target")
	}
	return penalty
}

func compareToBaseline(baseline HealthSnapshot, current HealthSnapshot, recommendations *[]string) int {
	penalty := 0
	if baseline.CheapCognition.LastLatencyMS > 0 && current.CheapCognition.LastLatencyMS > baseline.CheapCognition.LastLatencyMS*2 {
		penalty += 10
		*recommendations = append(*recommendations, fmt.Sprintf("Cheap cognition latency has doubled since baseline (%dms -> %dms)", baseline.CheapCognition.LastLatencyMS, current.CheapCognition.LastLatencyMS))
	}
	if baseline.CheapCognition.ClassificationFails == 0 && current.CheapCognition.ClassificationFails > 0 {
		penalty += 10
		*recommendations = append(*recommendations, "Cheap cognition is failing after a clean startup baseline")
	}
	if current.Score < baseline.Score-20 {
		penalty += 10
		*recommendations = append(*recommendations, "Runtime health score is materially below startup baseline")
	}
	return penalty
}

func (s *Service) writeSnapshot(snapshot HealthSnapshot) error {
	if err := os.MkdirAll(filepath.Dir(s.healthFile), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.healthFile, data, 0o644)
}

func (s *Service) writeBaseline(snapshot HealthSnapshot) error {
	if err := os.MkdirAll(filepath.Dir(s.baselineFile), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.baselineFile, data, 0o644)
}

func (s *Service) readBaseline() (*HealthSnapshot, error) {
	data, err := os.ReadFile(s.baselineFile)
	if err != nil {
		return nil, err
	}
	var snapshot HealthSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (s *Service) readSnapshot() (*HealthSnapshot, error) {
	data, err := os.ReadFile(s.healthFile)
	if err != nil {
		return nil, err
	}
	var snapshot HealthSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (s *Service) writeSystemRecord(snapshot HealthSnapshot, baseline *HealthSnapshot) error {
	if s.systemDBFile == "" {
		return nil
	}
	store, err := systemdb.Open(s.systemDBFile)
	if err != nil {
		return err
	}
	defer store.Close()
	record, err := systemRecordFromSnapshot(snapshot)
	if err != nil {
		return err
	}
	var baselineRecord *systemdb.MaintenanceRecord
	if baseline != nil {
		converted, err := systemRecordFromSnapshot(*baseline)
		if err != nil {
			return err
		}
		baselineRecord = &converted
	}
	return store.RecordMaintenanceRun(record, baselineRecord)
}

func systemRecordFromSnapshot(snapshot HealthSnapshot) (systemdb.MaintenanceRecord, error) {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return systemdb.MaintenanceRecord{}, err
	}
	processes := make(map[string]systemdb.ProcessRecord, len(snapshot.Processes))
	for id, proc := range snapshot.Processes {
		processes[id] = systemdb.ProcessRecord{
			Name:    proc.Name,
			PIDFile: proc.PIDFile,
			Running: proc.Running,
			Owned:   proc.Owned,
			Action:  proc.Action,
			Message: proc.Message,
		}
	}
	return systemdb.MaintenanceRecord{
		Timestamp:             snapshot.Timestamp,
		Score:                 snapshot.Score,
		Summary:               snapshot.Summary,
		Baseline:              snapshot.Baseline,
		Deferred:              snapshot.Deferred,
		ComparedTo:            snapshot.ComparedTo,
		Recommendations:       append([]string(nil), snapshot.Recommendations...),
		Processes:             processes,
		LastRemediationAt:     snapshot.LastRemediationAt,
		LastRemediationAction: snapshot.LastRemediationAction,
		PayloadJSON:           string(payload),
	}, nil
}

func summarizeScore(score int) string {
	switch {
	case score >= 90:
		return "healthy"
	case score >= 70:
		return "watch"
	default:
		return "degraded"
	}
}

func (s *Service) shouldDeferMaintenance(stats CheapCognitionSnapshot, previous *HealthSnapshot, activeProbe bool) (bool, string) {
	if activeProbe || s.cfg.BusyWindowMinutes <= 0 {
		return false, ""
	}

	window := time.Duration(s.cfg.BusyWindowMinutes) * time.Minute
	if !stats.LastDecisionAt.IsZero() && time.Since(stats.LastDecisionAt) < window {
		return true, "Maintenance actions deferred because the runtime was recently active"
	}
	if previous == nil || previous.Timestamp.IsZero() {
		return false, ""
	}

	age := time.Since(previous.Timestamp)
	if age > window*2 {
		return false, ""
	}

	prevStats := previous.CheapCognition
	deltaCalls := deltaCounter(stats.ClassificationCalls, prevStats.ClassificationCalls)
	deltaHandled := deltaCounter(stats.Forwarded, prevStats.Forwarded) + deltaCounter(stats.Skipped, prevStats.Skipped)
	deltaFails := deltaCounter(stats.ClassificationFails, prevStats.ClassificationFails)

	if deltaCalls >= 12 || deltaHandled >= 10 || deltaFails >= int64(maxInt(2, s.cfg.MaxCheapFailures)) {
		return true, "Maintenance actions deferred because the runtime is under sustained recent load"
	}

	return false, ""
}

func (s *Service) restartBackoffActive(previous *HealthSnapshot) (bool, string) {
	if previous == nil || s.cfg.RestartBackoffMinutes <= 0 {
		return false, ""
	}
	if previous.LastRemediationAt.IsZero() || previous.LastRemediationAction == "" {
		return false, ""
	}
	if !strings.Contains(previous.LastRemediationAction, "restart") {
		return false, ""
	}
	window := time.Duration(s.cfg.RestartBackoffMinutes) * time.Minute
	if time.Since(previous.LastRemediationAt) < window {
		return true, fmt.Sprintf("Restart deferred because the last remediation was too recent (%s)", previous.LastRemediationAt.Format(time.RFC3339))
	}
	return false, ""
}

func mergeRemediation(snapshot HealthSnapshot, state ProcessState) HealthSnapshot {
	if state.Action == "" {
		return snapshot
	}
	snapshot.LastRemediationAt = snapshot.Timestamp
	snapshot.LastRemediationAction = state.Name + ":" + state.Action
	return snapshot
}

func (s *Service) enrichCycleMetadata(snapshot *HealthSnapshot, baseline *HealthSnapshot, previous *HealthSnapshot, startedAt time.Time) {
	if snapshot == nil {
		return
	}

	snapshot.PostCareScore = snapshot.Score
	snapshot.PostCareSummary = snapshot.Summary
	snapshot.PostCareAt = snapshot.Timestamp
	if !startedAt.IsZero() {
		snapshot.CycleDurationMs = snapshot.Timestamp.Sub(startedAt).Milliseconds()
		if snapshot.CycleDurationMs < 0 {
			snapshot.CycleDurationMs = 0
		}
	}

	if baseline != nil {
		snapshot.BaselineScore = baseline.Score
		snapshot.BaselineSummary = baseline.Summary
		snapshot.BaselineAt = baseline.Timestamp
	}
	if previous != nil {
		snapshot.PreCheckScore = previous.Score
		snapshot.PreCheckSummary = previous.Summary
		snapshot.PreCheckAt = previous.Timestamp
		snapshot.ScoreDelta = snapshot.Score - previous.Score
	}
	if snapshot.Baseline && snapshot.BaselineScore == 0 {
		snapshot.BaselineScore = snapshot.Score
		snapshot.BaselineSummary = snapshot.Summary
		snapshot.BaselineAt = snapshot.Timestamp
	}

	snapshot.ActionsTaken = actionsTaken(snapshot.Processes)
	snapshot.RegressionFlags = regressionFlags(snapshot, baseline, previous)
}

func (s *Service) maintenanceBudgetSlots() int {
	switch {
	case s.cfg.BudgetPercent <= 0:
		return 1
	case s.cfg.BudgetPercent <= 5:
		return 1
	case s.cfg.BudgetPercent <= 10:
		return 2
	default:
		slots := s.cfg.BudgetPercent / 5
		if slots > 5 {
			return 5
		}
		if slots < 1 {
			return 1
		}
		return slots
	}
}

func hasBudget(budget *int) bool {
	return budget == nil || *budget > 0
}

func consumeBudget(budget *int) bool {
	if budget == nil {
		return true
	}
	if *budget <= 0 {
		return false
	}
	*budget = *budget - 1
	return true
}

func removeStalePID(pidFile string, budget *int) (string, bool) {
	if pidFile == "" || !hasBudget(budget) {
		return "", false
	}

	data, err := os.ReadFile(pidFile)
	if err != nil {
		return "", false
	}
	pid := strings.TrimSpace(string(data))
	if pid == "" {
		if !consumeBudget(budget) {
			return "", false
		}
		if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
			return "", false
		}
		return "removed_empty_pid_file", true
	}

	if pidAlive(pidFile) {
		return "", false
	}
	if !consumeBudget(budget) {
		return "", false
	}
	if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
		return "", false
	}
	return "stale_pid_removed", true
}

func trimLogIfNeeded(path string, maxMB int, budget *int) (string, bool) {
	if path == "" || maxMB <= 0 || !hasBudget(budget) {
		return "", false
	}

	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return "", false
	}

	maxBytes := int64(maxMB) * 1024 * 1024
	if info.Size() <= maxBytes {
		return "", false
	}
	if !consumeBudget(budget) {
		return "", false
	}

	file, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer file.Close()

	keepBytes := maxBytes / 2
	if keepBytes < 64*1024 {
		keepBytes = minInt64(maxBytes, 64*1024)
	}
	if keepBytes > info.Size() {
		keepBytes = info.Size()
	}

	if _, err := file.Seek(-keepBytes, io.SeekEnd); err != nil {
		return "", false
	}
	buf := make([]byte, keepBytes)
	n, err := io.ReadFull(file, buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return "", false
	}
	buf = buf[:n]

	// Skip partial line if possible so the trimmed log remains readable.
	if idx := strings.IndexByte(string(buf), '\n'); idx >= 0 && idx+1 < len(buf) {
		buf = buf[idx+1:]
	}

	trimmed := append([]byte("[trimmed by Spiderweb maintenance]\n"), buf...)
	if err := os.WriteFile(path, trimmed, 0o644); err != nil {
		return "", false
	}
	return fmt.Sprintf("Trimmed oversized log %s", filepath.Base(path)), true
}

func pidAlive(pidFile string) bool {
	if pidFile == "" {
		return false
	}
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}
	pid := strings.TrimSpace(string(data))
	if pid == "" {
		return false
	}
	parsedPID, ok := parsePID(pid)
	if !ok {
		return false
	}
	proc, err := os.FindProcess(parsedPID)
	if err != nil {
		return false
	}
	if proc == nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func parsePID(value string) (int, bool) {
	pid, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || pid <= 0 {
		return 0, false
	}
	return pid, true
}

func expandHome(path string) string {
	if path == "" {
		return path
	}
	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		if len(path) > 1 && path[1] == '/' {
			return home + path[1:]
		}
		return home
	}
	return path
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func deltaCounter(current, previous int64) int64 {
	if current <= previous {
		return 0
	}
	return current - previous
}

func actionsTaken(processes map[string]ProcessState) []string {
	if len(processes) == 0 {
		return nil
	}
	keys := make([]string, 0, len(processes))
	for key := range processes {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	actions := make([]string, 0, len(keys))
	for _, key := range keys {
		proc := processes[key]
		if proc.Action == "" {
			continue
		}
		actions = append(actions, proc.Name+":"+proc.Action)
	}
	if len(actions) == 0 {
		return nil
	}
	return actions
}

func regressionFlags(snapshot *HealthSnapshot, baseline *HealthSnapshot, previous *HealthSnapshot) []string {
	if snapshot == nil {
		return nil
	}
	var flags []string
	if baseline != nil {
		if snapshot.Score < baseline.Score-20 {
			flags = append(flags, "baseline_score_regression")
		}
		if baseline.CheapCognition.LastLatencyMS > 0 && snapshot.CheapCognition.LastLatencyMS > baseline.CheapCognition.LastLatencyMS*2 {
			flags = append(flags, "cheap_cognition_latency_regression")
		}
		if baseline.CheapCognition.ClassificationFails == 0 && snapshot.CheapCognition.ClassificationFails > 0 {
			flags = append(flags, "cheap_cognition_failure_regression")
		}
	}
	if previous != nil {
		if snapshot.Score < previous.Score {
			flags = append(flags, "cycle_score_down")
		}
		if snapshot.LastRemediationAction != "" && snapshot.Score <= previous.Score {
			flags = append(flags, "remediation_no_improvement")
		}
	}
	if len(flags) == 0 {
		return nil
	}
	return flags
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
