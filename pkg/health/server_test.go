package health

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JustSebNL/Spiderweb/pkg/agentdb"
	"github.com/JustSebNL/Spiderweb/pkg/bus"
	"github.com/JustSebNL/Spiderweb/pkg/config"
	"github.com/JustSebNL/Spiderweb/pkg/constants"
	"github.com/JustSebNL/Spiderweb/pkg/maintenance"
)

func localhostRequest(method, target string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, target, body)
	req.RemoteAddr = "127.0.0.1:12345"
	return req
}

func prepareObserverRuntime(t *testing.T) (*Server, string) {
	t.Helper()
	s := NewServer("127.0.0.1", 0)
	s.SetReady(true)

	workspace := t.TempDir()
	healthFile := filepath.Join(workspace, "runtime-health.json")
	s.SetWorkspace(workspace)
	s.SetObserverHealthFile(healthFile)

	svc := maintenance.NewService(
		workspace,
		config.MaintenanceConfig{
			Enabled:               true,
			HealthFile:            healthFile,
			AutoRemediate:         false,
			BudgetPercent:         5,
			RestartBackoffMinutes: 30,
		},
		config.CheapCognitionConfig{Enabled: false},
		config.TriggerConfig{
			Enabled: true,
			PIDFile: filepath.Join(workspace, "missing-trigger.pid"),
		},
		nil,
		nil,
		nil,
	)
	svc.RunOnce(context.Background())
	return s, workspace
}

func TestValveState_NoBus(t *testing.T) {
	s := NewServer("127.0.0.1", 0)
	s.SetReady(true)

	req := httptest.NewRequest(http.MethodGet, "/valve/state", nil)
	w := httptest.NewRecorder()
	s.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp ValveResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.State != int(constants.ValveStateSystemError) {
		t.Fatalf("state = %d, want %d", resp.State, int(constants.ValveStateSystemError))
	}
}

func TestValveOffer_AcceptsWhenQueueHasSpace(t *testing.T) {
	s := NewServer("127.0.0.1", 0)
	mb := bus.NewMessageBus()
	s.SetMessageBus(mb)
	s.SetReady(true)

	body, _ := json.Marshal(ValveOfferRequest{
		Channel:  "cli",
		SenderID: "u1",
		ChatID:   "c1",
		Content:  "hello",
	})

	req := httptest.NewRequest(http.MethodPost, "/valve/offer", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	msg, ok := mb.ConsumeInbound(context.Background())
	if !ok {
		t.Fatal("expected inbound message")
	}
	if msg.Content != "hello" {
		t.Fatalf("content = %q, want %q", msg.Content, "hello")
	}
}

func TestObserverOverviewHandler(t *testing.T) {
	s := NewServer("127.0.0.1", 0)
	s.SetReady(true)

	workspace := t.TempDir()
	s.SetWorkspace(workspace)

	stateDir := filepath.Join(workspace, "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state: %v", err)
	}

	current := maintenance.HealthSnapshot{
		Timestamp:       time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC),
		Score:           82,
		Summary:         "watch",
		ComparedTo:      "startup",
		Recommendations: []string{"Trigger worker is offline"},
		Processes: map[string]maintenance.ProcessState{
			"trigger": {
				Name:    "trigger",
				Running: false,
				Owned:   true,
				PIDFile: "/tmp/trigger.pid",
			},
			"cheap_cognition_vllm": {
				Name:    "cheap_cognition_vllm",
				Running: true,
				Owned:   true,
				PIDFile: "/tmp/youtu.pid",
			},
		},
	}
	baseline := maintenance.HealthSnapshot{
		Timestamp: time.Date(2026, 3, 23, 9, 0, 0, 0, time.UTC),
		Score:     97,
		Summary:   "healthy",
	}

	writeSnapshot := func(path string, snapshot maintenance.HealthSnapshot) {
		t.Helper()
		data, err := json.Marshal(snapshot)
		if err != nil {
			t.Fatalf("marshal snapshot: %v", err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatalf("write snapshot: %v", err)
		}
	}

	writeSnapshot(filepath.Join(stateDir, "runtime-health.json"), current)
	writeSnapshot(filepath.Join(stateDir, "runtime-health.json.baseline"), baseline)

	req := httptest.NewRequest(http.MethodGet, "/observer/overview", nil)
	w := httptest.NewRecorder()
	s.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Status   string `json:"status"`
		Score    int    `json:"score"`
		Summary  string `json:"summary"`
		Services []struct {
			ID    string `json:"id"`
			State string `json:"state"`
		} `json:"services"`
		Baseline *struct {
			Score int `json:"score"`
		} `json:"baseline"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal overview: %v", err)
	}
	if resp.Status != "watch" || resp.Summary != "watch" || resp.Score != 82 {
		t.Fatalf("unexpected overview payload: %+v", resp)
	}
	if resp.Baseline == nil || resp.Baseline.Score != 97 {
		t.Fatalf("baseline missing or wrong: %+v", resp.Baseline)
	}
	if len(resp.Services) != 2 {
		t.Fatalf("services len = %d, want 2", len(resp.Services))
	}
}

func TestObserverOverviewHandler_NotFoundWithoutSnapshot(t *testing.T) {
	s := NewServer("127.0.0.1", 0)
	s.SetReady(true)
	s.SetWorkspace(t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/observer/overview", nil)
	w := httptest.NewRecorder()
	s.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestObserverStats24hHandler(t *testing.T) {
	s, _ := prepareObserverRuntime(t)

	req := httptest.NewRequest(http.MethodGet, "/observer/stats/24h", nil)
	w := httptest.NewRecorder()
	s.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		TotalEvents    int `json:"total_events"`
		ErrorEvents    int `json:"error_events"`
		CriticalEvents int `json:"critical_events"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal stats24h: %v", err)
	}
	if resp.TotalEvents == 0 {
		t.Fatalf("expected observer stats endpoint to report events")
	}
}

func TestObserverEventsHandler(t *testing.T) {
	s, _ := prepareObserverRuntime(t)

	req := httptest.NewRequest(http.MethodGet, "/observer/events?limit=5", nil)
	w := httptest.NewRecorder()
	s.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Events []struct {
			EventType string `json:"event_type"`
		} `json:"events"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal events: %v", err)
	}
	if len(resp.Events) == 0 {
		t.Fatalf("expected at least one observer event")
	}
}

func TestObserverSelfCareCyclesHandler(t *testing.T) {
	s, _ := prepareObserverRuntime(t)

	req := httptest.NewRequest(http.MethodGet, "/observer/self-care/cycles?limit=5", nil)
	w := httptest.NewRecorder()
	s.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Cycles []struct {
			Kind               string   `json:"kind"`
			Score              int      `json:"score"`
			BaselineScore      int      `json:"baseline_score"`
			BaselineSnapshotID int      `json:"baseline_snapshot_id"`
			PostCareScore      int      `json:"post_care_score"`
			PostCareSnapshotID int      `json:"post_care_snapshot_id"`
			CycleDurationMs    int64    `json:"cycle_duration_ms"`
			ActionsTaken       []string `json:"actions_taken"`
			RegressionFlags    []string `json:"regression_flags"`
		} `json:"cycles"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal cycles: %v", err)
	}
	if len(resp.Cycles) == 0 {
		t.Fatalf("expected at least one self-care cycle")
	}
	if resp.Cycles[0].BaselineScore == 0 {
		t.Fatalf("expected baseline score in self-care cycle payload")
	}
	if resp.Cycles[0].PostCareScore == 0 {
		t.Fatalf("expected post-care score in self-care cycle payload")
	}
	if resp.Cycles[0].BaselineSnapshotID == 0 {
		t.Fatalf("expected baseline snapshot id in self-care cycle payload")
	}
	if resp.Cycles[0].PostCareSnapshotID == 0 {
		t.Fatalf("expected post-care snapshot id in self-care cycle payload")
	}
	if resp.Cycles[0].CycleDurationMs < 0 {
		t.Fatalf("expected non-negative cycle duration")
	}
}

func TestObserverDashboardHandler(t *testing.T) {
	s, workspace := prepareObserverRuntime(t)

	agentStore, err := agentdb.Open(filepath.Join(workspace, "agent.db"))
	if err != nil {
		t.Fatalf("open agent db: %v", err)
	}
	defer agentStore.Close()
	now := time.Now().UTC()
	if err := agentStore.RecordPresence(agentdb.AgentPresence{
		AgentID:    "pipeline-alerts",
		AgentName:  "Pipeline Alerts",
		PipelineID: "alerts",
		State:      "busy",
		Status:     "busy",
		LastSeenAt: now,
		UpdatedAt:  now,
	}); err != nil {
		t.Fatalf("record presence: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/observer/dashboard?limit=5", nil)
	w := httptest.NewRecorder()
	s.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Overview *struct {
			Score int `json:"score"`
		} `json:"overview"`
		Services []any `json:"services"`
		Agents   []any `json:"agents"`
		Events   []any `json:"events"`
		SelfCare *struct {
			Cycles []any `json:"cycles"`
		} `json:"self_care"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal dashboard: %v", err)
	}
	if resp.Overview == nil || resp.Overview.Score == 0 {
		t.Fatalf("expected dashboard overview score")
	}
	if len(resp.Services) == 0 || len(resp.Events) == 0 || resp.SelfCare == nil || len(resp.SelfCare.Cycles) == 0 {
		t.Fatalf("expected dashboard response to include services, events, and self-care cycles")
	}
	if len(resp.Agents) == 0 {
		t.Fatalf("expected dashboard response to include agents")
	}
}

func TestObserverAgentsHandler(t *testing.T) {
	s := NewServer("127.0.0.1", 0)
	s.SetReady(true)

	workspace := t.TempDir()
	healthFile := filepath.Join(workspace, "runtime-health.json")
	s.SetWorkspace(workspace)
	s.SetObserverHealthFile(healthFile)

	store, err := agentdb.Open(filepath.Join(workspace, "agent.db"))
	if err != nil {
		t.Fatalf("open agent db: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC()
	if err := store.RecordPresence(agentdb.AgentPresence{
		AgentID:         "pipeline-alerts",
		AgentName:       "Pipeline Alerts",
		PipelineID:      "alerts",
		State:           "busy",
		Status:          "busy",
		Channel:         "slack",
		ChatID:          "room-1",
		LastTaskSummary: "processed alert batch",
		LastSeenAt:      now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("record presence: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/observer/agents", nil)
	w := httptest.NewRecorder()
	s.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Agents []struct {
			AgentID    string `json:"agent_id"`
			PipelineID string `json:"pipeline_id"`
			State      string `json:"state"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal agents: %v", err)
	}
	if len(resp.Agents) != 1 {
		t.Fatalf("agents len = %d, want 1", len(resp.Agents))
	}
	if resp.Agents[0].AgentID != "pipeline-alerts" || resp.Agents[0].PipelineID != "alerts" || resp.Agents[0].State != "busy" {
		t.Fatalf("unexpected agents payload: %+v", resp.Agents[0])
	}
}

func TestObserverAgentsSummaryHandler(t *testing.T) {
	s := NewServer("127.0.0.1", 0)
	s.SetReady(true)

	workspace := t.TempDir()
	healthFile := filepath.Join(workspace, "runtime-health.json")
	s.SetWorkspace(workspace)
	s.SetObserverHealthFile(healthFile)

	store, err := agentdb.Open(filepath.Join(workspace, "agent.db"))
	if err != nil {
		t.Fatalf("open agent db: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC()
	for _, item := range []agentdb.AgentPresence{
		{AgentID: "a1", AgentName: "A1", PipelineID: "alerts", State: "busy", LastSeenAt: now, UpdatedAt: now},
		{AgentID: "a2", AgentName: "A2", PipelineID: "alerts", State: "idle", LastSeenAt: now, UpdatedAt: now},
	} {
		if err := store.RecordPresence(item); err != nil {
			t.Fatalf("record presence: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/observer/agents/summary", nil)
	w := httptest.NewRecorder()
	s.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Total      int            `json:"total"`
		ByState    map[string]int `json:"by_state"`
		ByPipeline map[string]int `json:"by_pipeline"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal summary: %v", err)
	}
	if resp.Total != 2 || resp.ByPipeline["alerts"] != 2 {
		t.Fatalf("unexpected agent summary: %+v", resp)
	}
}

func TestObserverRestartHandler(t *testing.T) {
	s := NewServer("127.0.0.1", 0)
	s.SetReady(true)
	s.SetObserverRestartFunc(func(ctx context.Context, service string) (map[string]any, error) {
		return map[string]any{"ok": true, "action": "restart"}, nil
	})

	body := bytes.NewBufferString(`{"service":"trigger"}`)
	req := localhostRequest(http.MethodPost, "/observer/actions/restart", body)
	w := httptest.NewRecorder()
	s.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestObserverSelfCareRunHandler(t *testing.T) {
	s := NewServer("127.0.0.1", 0)
	s.SetReady(true)
	s.SetObserverSelfCareFunc(func(ctx context.Context) (map[string]any, error) {
		return map[string]any{"ok": true, "action": "self_care_run"}, nil
	})

	req := localhostRequest(http.MethodPost, "/observer/actions/self-care/run", nil)
	w := httptest.NewRecorder()
	s.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestObserverGenerateAndLatestReportHandlers(t *testing.T) {
	s, workspace := prepareObserverRuntime(t)

	agentStore, err := agentdb.Open(filepath.Join(workspace, "agent.db"))
	if err != nil {
		t.Fatalf("open agent db: %v", err)
	}
	defer agentStore.Close()
	now := time.Now().UTC()
	if err := agentStore.RecordPresence(agentdb.AgentPresence{
		AgentID:    "pipeline-alerts",
		AgentName:  "Pipeline Alerts",
		PipelineID: "alerts",
		State:      "busy",
		LastSeenAt: now,
		UpdatedAt:  now,
	}); err != nil {
		t.Fatalf("record presence: %v", err)
	}

	req := localhostRequest(http.MethodPost, "/observer/reports/generate?limit=5", nil)
	w := httptest.NewRecorder()
	s.server.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("generate status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/observer/reports/latest", nil)
	w = httptest.NewRecorder()
	s.server.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("latest status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/observer/reports/latest?format=html", nil)
	w = httptest.NewRecorder()
	s.server.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("latest html status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("unexpected content type: %q", ct)
	}
}

func TestObserverUIHandler(t *testing.T) {
	s := NewServer("127.0.0.1", 0)
	s.SetReady(true)

	ui := []byte("<!doctype html><html><body>ok</body></html>")
	path := filepath.Join(t.TempDir(), "observer.html")
	if err := os.WriteFile(path, ui, 0o644); err != nil {
		t.Fatalf("write ui: %v", err)
	}
	t.Setenv("SPIDERWEB_OBSERVER_UI_HTML", path)

	req := httptest.NewRequest(http.MethodGet, "/observer/ui", nil)
	w := httptest.NewRecorder()
	s.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("expected Cache-Control no-store, got %q", w.Header().Get("Cache-Control"))
	}
	if ct := w.Header().Get("Content-Type"); ct == "" {
		t.Fatalf("expected Content-Type to be set")
	}
}

func TestObserverJournalGenerateAndLatestHandlers(t *testing.T) {
	s, _ := prepareObserverRuntime(t)

	req := localhostRequest(http.MethodPost, "/observer/journal/generate", nil)
	w := httptest.NewRecorder()
	s.server.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("journal generate status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var generated struct {
		Entry *struct {
			Title string `json:"title"`
			Body  string `json:"body"`
		} `json:"entry"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &generated); err != nil {
		t.Fatalf("unmarshal generated journal: %v", err)
	}
	if generated.Entry == nil || generated.Entry.Title == "" || generated.Entry.Body == "" {
		t.Fatalf("expected generated journal entry")
	}

	req = httptest.NewRequest(http.MethodGet, "/observer/journal/latest", nil)
	w = httptest.NewRecorder()
	s.server.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("journal latest status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}
