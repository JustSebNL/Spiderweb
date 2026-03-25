package agent

import (
	"errors"
	"strings"
	"testing"

	"github.com/JustSebNL/Spiderweb/pkg/bus"
	"github.com/JustSebNL/Spiderweb/pkg/cognition"
	"github.com/JustSebNL/Spiderweb/pkg/config"
)

func TestForwardBlockReason_PreventsOpenClawLoop(t *testing.T) {
	t.Parallel()

	al := &AgentLoop{cfg: &config.Config{}}
	blocked, reason := al.forwardBlockReason(bus.InboundMessage{Channel: "openclaw"})
	if !blocked || reason != "openclaw_loop_prevented" {
		t.Fatalf("expected openclaw loop prevention, got blocked=%t reason=%q", blocked, reason)
	}
}

func TestForwardBlockReason_RespectsChannelAllowList(t *testing.T) {
	t.Parallel()

	al := &AgentLoop{cfg: &config.Config{
		Intake: config.IntakeConfig{
			ForwardAllowChannels: []string{"slack", "telegram"},
		},
	}}

	blocked, reason := al.forwardBlockReason(bus.InboundMessage{Channel: "discord"})
	if !blocked || reason != "channel_not_allowed" {
		t.Fatalf("expected channel_not_allowed, got blocked=%t reason=%q", blocked, reason)
	}

	blocked, reason = al.forwardBlockReason(bus.InboundMessage{Channel: "slack"})
	if blocked {
		t.Fatalf("expected slack to be allowed, got reason=%q", reason)
	}
}

func TestForwardBlockReason_RespectsChannelDenyList(t *testing.T) {
	t.Parallel()

	al := &AgentLoop{cfg: &config.Config{
		Intake: config.IntakeConfig{
			ForwardDenyChannels: []string{"discord"},
		},
	}}

	blocked, reason := al.forwardBlockReason(bus.InboundMessage{Channel: "discord"})
	if !blocked || reason != "channel_denied" {
		t.Fatalf("expected channel_denied, got blocked=%t reason=%q", blocked, reason)
	}
}

func TestForwardBlockReason_RespectsServicePolicies(t *testing.T) {
	t.Parallel()

	al := &AgentLoop{cfg: &config.Config{
		Intake: config.IntakeConfig{
			ForwardAllowServices: []string{"inbox", "alerts"},
			ForwardDenyServices:  []string{"noisy-service"},
		},
	}}

	blocked, reason := al.forwardBlockReason(bus.InboundMessage{
		Channel: "slack",
		Metadata: map[string]string{
			"service": "billing",
		},
	})
	if !blocked || reason != "service_not_allowed" {
		t.Fatalf("expected service_not_allowed, got blocked=%t reason=%q", blocked, reason)
	}

	blocked, reason = al.forwardBlockReason(bus.InboundMessage{
		Channel: "slack",
		Metadata: map[string]string{
			"service_name": "noisy-service",
		},
	})
	if !blocked || reason != "service_denied" {
		t.Fatalf("expected service_denied, got blocked=%t reason=%q", blocked, reason)
	}

	blocked, reason = al.forwardBlockReason(bus.InboundMessage{
		Channel: "slack",
		Metadata: map[string]string{
			"pipeline": "alerts",
		},
	})
	if blocked {
		t.Fatalf("expected alerts pipeline to be allowed, got reason=%q", reason)
	}
}

func TestAnnotateForwardMetadata_DegradedWhenCheapCognitionUnavailable(t *testing.T) {
	t.Parallel()

	metadata := annotateForwardMetadata(
		map[string]string{"source": "openclaw-bridge"},
		nil,
		"",
		"unavailable",
		errors.New("vllm health check failed"),
	)

	if metadata["routing_mode"] != "degraded" {
		t.Fatalf("expected degraded routing mode, got %q", metadata["routing_mode"])
	}
	if metadata["cheap_cognition"] != "unavailable" {
		t.Fatalf("expected unavailable cheap cognition, got %q", metadata["cheap_cognition"])
	}
	if metadata["intake_forward_decision"] != "forward_degraded" {
		t.Fatalf("expected forward_degraded decision, got %q", metadata["intake_forward_decision"])
	}
	if metadata["fallback_reason"] != "vllm health check failed" {
		t.Fatalf("expected fallback reason to be preserved, got %q", metadata["fallback_reason"])
	}
}

func TestAnnotateForwardMetadata_UsesCheapClassificationWhenAvailable(t *testing.T) {
	t.Parallel()

	metadata := annotateForwardMetadata(
		map[string]string{},
		&cognition.ClassificationResult{
			Priority:         "high",
			Category:         "alert",
			EscalationNeeded: true,
			OneLineSummary:   "important event",
		},
		"forward",
		"ok",
		nil,
	)

	if metadata["routing_mode"] != "cheap_cognition" {
		t.Fatalf("expected cheap_cognition routing mode, got %q", metadata["routing_mode"])
	}
	if metadata["cheap_cognition_priority"] != "high" {
		t.Fatalf("expected priority to be preserved, got %q", metadata["cheap_cognition_priority"])
	}
	if metadata["intake_forward_decision"] != "forward" {
		t.Fatalf("expected forward decision, got %q", metadata["intake_forward_decision"])
	}
}

func TestBuildInboundUserMessage_AddsTriageNote(t *testing.T) {
	t.Parallel()

	msg := bus.InboundMessage{
		Content: "original body",
		Metadata: map[string]string{
			"routing_mode":             "cheap_cognition",
			"cheap_cognition_priority": "high",
			"cheap_cognition_category": "alert",
			"cheap_cognition_summary":  "important event",
		},
	}

	built := buildInboundUserMessage(msg)
	if built == msg.Content {
		t.Fatal("expected intake note to be added")
	}
	if !strings.Contains(built, "priority=high") || !strings.Contains(built, "important event") {
		t.Fatalf("expected triage note in user message, got %q", built)
	}
}

func TestBuildInboundUserMessage_AddsDegradedNote(t *testing.T) {
	t.Parallel()

	msg := bus.InboundMessage{
		Content: "original body",
		Metadata: map[string]string{
			"routing_mode":    "degraded",
			"fallback_reason": "vllm health check failed",
		},
	}

	built := buildInboundUserMessage(msg)
	if !strings.Contains(built, "Intake degraded mode") || !strings.Contains(built, "vllm health check failed") {
		t.Fatalf("expected degraded note in user message, got %q", built)
	}
}
