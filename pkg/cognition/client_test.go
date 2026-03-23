package cognition

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JustSebNL/Spiderweb/pkg/config"
)

func TestClassifyEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %s, want /chat/completions", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q", got)
		}

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req["model"] != "tencent/Youtu-LLM-2B" {
			t.Fatalf("model = %v", req["model"])
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": "```json\n{\"priority\":\"high\",\"category\":\"deploy\",\"escalation_needed\":true,\"one_line_summary\":\"Deploy review requested.\"}\n```",
					},
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "tencent/Youtu-LLM-2B", 5*time.Second)
	result, err := client.ClassifyEvent(context.Background(), Event{
		EventID:   "evt_1",
		Source:    "slack",
		EventType: "message.created",
		Payload: map[string]any{
			"text": "Can someone review the deploy script?",
		},
	})
	if err != nil {
		t.Fatalf("ClassifyEvent() error = %v", err)
	}
	if result.Priority != "high" {
		t.Fatalf("Priority = %q", result.Priority)
	}
	if !result.EscalationNeeded {
		t.Fatal("EscalationNeeded = false, want true")
	}
}

func TestSummarizeText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": "Short operational summary.",
					},
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "", "", 5*time.Second)
	summary, err := client.SummarizeText(context.Background(), "Very long thread")
	if err != nil {
		t.Fatalf("SummarizeText() error = %v", err)
	}
	if summary != "Short operational summary." {
		t.Fatalf("summary = %q", summary)
	}
}

func TestNewClientFromConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Intake.CheapCognition.Enabled = true
	cfg.Intake.CheapCognition.BaseURL = "http://127.0.0.1:8000/v1"
	cfg.Intake.CheapCognition.APIKey = "dummy"
	cfg.Intake.CheapCognition.Model = "tencent/Youtu-LLM-2B"
	cfg.Intake.CheapCognition.TimeoutSeconds = 12

	client, err := NewClientFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewClientFromConfig() error = %v", err)
	}
	if client.baseURL != "http://127.0.0.1:8000/v1" {
		t.Fatalf("baseURL = %q", client.baseURL)
	}
	if client.model != "tencent/Youtu-LLM-2B" {
		t.Fatalf("model = %q", client.model)
	}
}

func TestNewClientFromConfig_Disabled(t *testing.T) {
	cfg := config.DefaultConfig()
	_, err := NewClientFromConfig(cfg)
	if err == nil {
		t.Fatal("expected error when cheap cognition is disabled")
	}
}
