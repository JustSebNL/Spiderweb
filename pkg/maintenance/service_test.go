package maintenance

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JustSebNL/Spiderweb/pkg/config"
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
}

func (f *fakeRuntimeController) Start() error {
	f.startCalls++
	return f.startErr
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

	if !svc.shouldDeferMaintenance(recent, false) {
		t.Fatalf("expected maintenance to defer for recent activity")
	}
	if svc.shouldDeferMaintenance(old, false) {
		t.Fatalf("expected maintenance to continue for older activity")
	}
	if svc.shouldDeferMaintenance(recent, true) {
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
	youtuCtl := &fakeRuntimeController{}

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
		youtuCtl,
	)

	snapshot := svc.collectSnapshot(context.Background(), nil, nil, false)
	t.Logf("snapshot summary=%s score=%d deferred=%t recommendations=%v", snapshot.Summary, snapshot.Score, snapshot.Deferred, snapshot.Recommendations)

	if !snapshot.Deferred {
		t.Fatalf("expected snapshot to defer maintenance during recent activity")
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
