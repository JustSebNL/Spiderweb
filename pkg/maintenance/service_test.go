package maintenance

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JustSebNL/Spiderweb/pkg/config"
	"github.com/JustSebNL/Spiderweb/pkg/systemdb"
)

type fakeRuntimeStats struct {
	snapshot CheapCognitionSnapshot
}

func (f fakeRuntimeStats) CheapCognitionSnapshot() CheapCognitionSnapshot {
	return f.snapshot
}

type fakeRuntimeController struct {
	startCalls int
	startErr   error
	mu         sync.Mutex
	blockCh    chan struct{}
}

func (f *fakeRuntimeController) Start() error {
	f.mu.Lock()
	f.startCalls++
	blockCh := f.blockCh
	startErr := f.startErr
	f.mu.Unlock()
	if blockCh != nil {
		<-blockCh
	}
	return startErr
}

func (f *fakeRuntimeController) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.startCalls
}

func TestMaintenance_ShouldDeferMaintenanceWhenRecentlyActive(t *testing.T) {
	t.Parallel()

	svc := NewService(
		t.TempDir(),
		config.MaintenanceConfig{
			Enabled:           true,
			BusyWindowMinutes: 10,
		},
		config.CheapCognitionConfig{},
		config.TriggerConfig{},
		nil,
		nil,
		nil,
	)

	recent := CheapCognitionSnapshot{
		LastDecisionAt: time.Now().Add(-2 * time.Minute),
	}
	old := CheapCognitionSnapshot{
		LastDecisionAt: time.Now().Add(-30 * time.Minute),
	}

	t.Logf("recent decision at: %s", recent.LastDecisionAt.Format(time.RFC3339))
	t.Logf("old decision at: %s", old.LastDecisionAt.Format(time.RFC3339))

	deferred, reason := svc.shouldDeferMaintenance(recent, nil, false)
	if !deferred {
		t.Fatalf("expected maintenance to defer for recent activity")
	}
	if !strings.Contains(reason, "recently active") {
		t.Fatalf("expected recent activity reason, got %q", reason)
	}
	deferred, _ = svc.shouldDeferMaintenance(old, nil, false)
	if deferred {
		t.Fatalf("expected maintenance to continue for older activity")
	}
	deferred, _ = svc.shouldDeferMaintenance(recent, nil, true)
	if deferred {
		t.Fatalf("baseline probe should not be deferred")
	}
}

func TestMaintenance_RestartBackoffActive(t *testing.T) {
	t.Parallel()

	svc := NewService(
		t.TempDir(),
		config.MaintenanceConfig{
			Enabled:               true,
			RestartBackoffMinutes: 30,
		},
		config.CheapCognitionConfig{},
		config.TriggerConfig{},
		nil,
		nil,
		nil,
	)

	recent := &HealthSnapshot{
		LastRemediationAt:     time.Now().Add(-5 * time.Minute),
		LastRemediationAction: "trigger:restart_requested",
	}
	old := &HealthSnapshot{
		LastRemediationAt:     time.Now().Add(-45 * time.Minute),
		LastRemediationAction: "trigger:restart_requested",
	}

	backedOff, reason := svc.restartBackoffActive(recent)
	t.Logf("recent remediation backed off=%t reason=%q", backedOff, reason)
	if !backedOff {
		t.Fatalf("expected recent restart remediation to be backed off")
	}

	backedOff, reason = svc.restartBackoffActive(old)
	t.Logf("old remediation backed off=%t reason=%q", backedOff, reason)
	if backedOff {
		t.Fatalf("expected old restart remediation to be allowed")
	}
}

func TestMaintenance_RemoveStalePID_EmptyFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pidFile := filepath.Join(dir, "empty.pid")
	if err := os.WriteFile(pidFile, []byte(""), 0o644); err != nil {
		t.Fatalf("write pid file: %v", err)
	}

	budget := 1
	action, ok := removeStalePID(pidFile, &budget)
	t.Logf("removeStalePID action=%q ok=%t remaining_budget=%d", action, ok, budget)

	if !ok || action != "removed_empty_pid_file" {
		t.Fatalf("expected empty pid file to be removed, got action=%q ok=%t", action, ok)
	}
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Fatalf("expected pid file to be gone, stat err=%v", err)
	}
}

func TestMaintenance_TrimLogIfNeeded(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	logFile := filepath.Join(dir, "runtime.log")
	content := strings.Repeat("0123456789abcdef\n", 70000)
	if err := os.WriteFile(logFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write log file: %v", err)
	}

	budget := 1
	action, ok := trimLogIfNeeded(logFile, 1, &budget)
	t.Logf("trimLogIfNeeded action=%q ok=%t remaining_budget=%d", action, ok, budget)

	if !ok {
		t.Fatalf("expected oversized log to be trimmed")
	}

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read trimmed log: %v", err)
	}
	t.Logf("trimmed log size=%d bytes", len(data))
	if !strings.HasPrefix(string(data), "[trimmed by Spiderweb maintenance]\n") {
		t.Fatalf("expected trimmed prefix in log file")
	}
}

func TestMaintenance_RemoveStalePID_InvalidPID(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pidFile := filepath.Join(dir, "invalid.pid")
	if err := os.WriteFile(pidFile, []byte("not-a-pid"), 0o644); err != nil {
		t.Fatalf("write pid file: %v", err)
	}

	budget := 1
	action, ok := removeStalePID(pidFile, &budget)
	if !ok || action != "stale_pid_removed" {
		t.Fatalf("expected invalid pid file to be removed as stale, got action=%q ok=%t", action, ok)
	}
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Fatalf("expected invalid pid file to be gone, stat err=%v", err)
	}
}

func TestMaintenance_CollectSnapshot_DefersAndSkipsRestartWhenBusy(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	triggerCtl := &fakeRuntimeController{}
	brainCtl := &fakeRuntimeController{}

	stats := fakeRuntimeStats{
		snapshot: CheapCognitionSnapshot{
			LastDecisionAt: time.Now().Add(-1 * time.Minute),
			LastLatencyMS:  120,
		},
	}

	svc := NewService(
		workspace,
		config.MaintenanceConfig{
			Enabled:               true,
			AutoRemediate:         true,
			BusyWindowMinutes:     10,
			RestartBackoffMinutes: 30,
			BudgetPercent:         5,
			MaxLogMB:              1,
			HighLatencyMs:         2500,
			MaxCheapFailures:      3,
			MaxForwardSkips:       500,
			RestartOnProcessDeath: true,
		},
		config.CheapCognitionConfig{
			Enabled: false,
		},
		config.TriggerConfig{
			Enabled:   true,
			AutoStart: true,
			Workdir:   filepath.Join(workspace, "trigger"),
			PIDFile:   filepath.Join(workspace, "trigger.pid"),
			LogFile:   filepath.Join(workspace, "trigger.log"),
		},
		stats,
		triggerCtl,
		brainCtl,
	)

	snapshot := svc.collectSnapshot(context.Background(), nil, nil, false)
	t.Logf("snapshot summary=%s score=%d deferred=%t recommendations=%v", snapshot.Summary, snapshot.Score, snapshot.Deferred, snapshot.Recommendations)

	if !snapshot.Deferred {
		t.Fatalf("expected snapshot to defer maintenance during recent activity")
	}
	if !strings.Contains(snapshot.DeferReason, "recently active") {
		t.Fatalf("expected recent activity defer reason, got %q", snapshot.DeferReason)
	}
	if triggerCtl.startCalls != 0 {
		t.Fatalf("expected trigger restart to be skipped during defer window, got %d calls", triggerCtl.startCalls)
	}
	if state, ok := snapshot.Processes["trigger"]; ok {
		t.Logf("trigger process state: running=%t action=%q message=%q", state.Running, state.Action, state.Message)
	}
}

func TestMaintenance_CollectSnapshot_RestartConsumesBudget(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	triggerCtl := &fakeRuntimeController{}

	svc := NewService(
		workspace,
		config.MaintenanceConfig{
			Enabled:               true,
			AutoRemediate:         true,
			BudgetPercent:         5,
			MaxLogMB:              1,
			RestartBackoffMinutes: 30,
			RestartOnProcessDeath: true,
		},
		config.CheapCognitionConfig{},
		config.TriggerConfig{
			Enabled:   true,
			AutoStart: true,
			Workdir:   filepath.Join(workspace, "trigger"),
			PIDFile:   filepath.Join(workspace, "trigger.pid"),
			LogFile:   filepath.Join(workspace, "trigger.log"),
		},
		nil,
		triggerCtl,
		nil,
	)

	logFile := filepath.Join(workspace, "trigger.log")
	content := strings.Repeat("0123456789abcdef\n", 70000)
	if err := os.WriteFile(logFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write log file: %v", err)
	}

	snapshot := svc.collectSnapshot(context.Background(), nil, nil, false)
	if triggerCtl.startCalls != 1 {
		t.Fatalf("expected one restart request within budget, got %d", triggerCtl.startCalls)
	}

	if !strings.Contains(snapshot.LastRemediationAction, "restart_requested") {
		t.Fatalf("expected restart remediation to win the single maintenance budget slot, got %q", snapshot.LastRemediationAction)
	}
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read trigger log: %v", err)
	}
	if strings.HasPrefix(string(data), "[trimmed by Spiderweb maintenance]\n") {
		t.Fatalf("expected log trim to be skipped after restart consumed the budget")
	}
}

func TestMaintenance_RunOnce_WritesSystemDB(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	healthFile := filepath.Join(workspace, "runtime-health.json")

	svc := NewService(
		workspace,
		config.MaintenanceConfig{
			Enabled:               true,
			HealthFile:            healthFile,
			AutoRemediate:         false,
			BudgetPercent:         5,
			RestartBackoffMinutes: 30,
		},
		config.CheapCognitionConfig{
			Enabled: false,
		},
		config.TriggerConfig{
			Enabled: false,
		},
		fakeRuntimeStats{},
		nil,
		nil,
	)

	svc.RunOnce(context.Background())

	store, err := systemdb.Open(filepath.Join(workspace, "system.db"))
	if err != nil {
		t.Fatalf("open system db: %v", err)
	}
	defer store.Close()

	stats, err := store.Stats24h()
	if err != nil {
		t.Fatalf("stats24h: %v", err)
	}
	if stats.TotalEvents == 0 {
		t.Fatalf("expected maintenance run to emit at least one observer event")
	}
}

func TestMaintenance_StopPreventsDelayedStartupRun(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	healthFile := filepath.Join(workspace, "runtime-health.json")

	svc := NewService(
		workspace,
		config.MaintenanceConfig{
			Enabled:    true,
			HealthFile: healthFile,
		},
		config.CheapCognitionConfig{},
		config.TriggerConfig{},
		fakeRuntimeStats{},
		nil,
		nil,
	)
	svc.startupDelay = 20 * time.Millisecond

	if err := svc.Start(); err != nil {
		t.Fatalf("start service: %v", err)
	}
	svc.Stop()

	time.Sleep(60 * time.Millisecond)
	if _, err := os.Stat(healthFile); !os.IsNotExist(err) {
		t.Fatalf("expected no startup health file after stop, stat err=%v", err)
	}
}

func TestMaintenance_RunOnceSkipsOverlap(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	triggerCtl := &fakeRuntimeController{blockCh: make(chan struct{})}

	svc := NewService(
		workspace,
		config.MaintenanceConfig{
			Enabled:               true,
			AutoRemediate:         true,
			RestartOnProcessDeath: true,
			BudgetPercent:         5,
			MaxLogMB:              1,
		},
		config.CheapCognitionConfig{},
		config.TriggerConfig{
			Enabled:   true,
			AutoStart: true,
			Workdir:   filepath.Join(workspace, "trigger"),
			PIDFile:   filepath.Join(workspace, "trigger.pid"),
			LogFile:   filepath.Join(workspace, "trigger.log"),
		},
		fakeRuntimeStats{},
		triggerCtl,
		nil,
	)

	done := make(chan struct{})
	go func() {
		defer close(done)
		svc.RunOnce(context.Background())
	}()

	time.Sleep(20 * time.Millisecond)
	svc.RunOnce(context.Background())

	if calls := triggerCtl.Calls(); calls != 1 {
		t.Fatalf("expected only one overlapping restart attempt, got %d", calls)
	}

	close(triggerCtl.blockCh)
	<-done
}

func TestMaintenance_ShouldDeferMaintenanceWhenSustainedLoadDetected(t *testing.T) {
	t.Parallel()

	svc := NewService(
		t.TempDir(),
		config.MaintenanceConfig{
			Enabled:           true,
			BusyWindowMinutes: 10,
			MaxCheapFailures:  3,
		},
		config.CheapCognitionConfig{},
		config.TriggerConfig{},
		nil,
		nil,
		nil,
	)

	previous := &HealthSnapshot{
		Timestamp: time.Now().Add(-5 * time.Minute),
		CheapCognition: CheapCognitionSnapshot{
			ClassificationCalls: 10,
			Forwarded:           5,
			Skipped:             2,
			ClassificationFails: 0,
		},
	}
	current := CheapCognitionSnapshot{
		ClassificationCalls: 28,
		Forwarded:           14,
		Skipped:             6,
		ClassificationFails: 0,
		LastDecisionAt:      time.Now().Add(-20 * time.Minute),
	}

	deferred, reason := svc.shouldDeferMaintenance(current, previous, false)
	if !deferred {
		t.Fatalf("expected maintenance to defer for sustained recent load")
	}
	if !strings.Contains(reason, "sustained recent load") {
		t.Fatalf("expected sustained load reason, got %q", reason)
	}
}
