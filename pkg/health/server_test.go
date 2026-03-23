package health

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JustSebNL/Spiderweb/pkg/bus"
	"github.com/JustSebNL/Spiderweb/pkg/constants"
	"github.com/JustSebNL/Spiderweb/pkg/maintenance"
)

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
