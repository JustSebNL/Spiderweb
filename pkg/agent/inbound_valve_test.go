package agent

import (
	"context"
	"testing"
	"time"

	"github.com/JustSebNL/Spiderweb/pkg/bus"
	"github.com/JustSebNL/Spiderweb/pkg/config"
)

func TestInboundValve_CoalescesBurst(t *testing.T) {
	v := newInboundValve(config.IntakeConfig{
		Enabled:             true,
		CoalesceWindowMs:    50,
		MaxBatchMessages:    10,
		MaxBatchChars:       0,
		DedupeWindowSeconds: 0,
	})

	in := make(chan bus.InboundMessage, 10)
	out := make(chan bus.InboundMessage, 10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go v.run(ctx, in, out)

	in <- bus.InboundMessage{Channel: "telegram", SenderID: "u1", ChatID: "c1", Content: "hello"}
	in <- bus.InboundMessage{Channel: "telegram", SenderID: "u1", ChatID: "c1", Content: "world"}
	close(in)

	select {
	case got := <-out:
		if got.Content != "hello\n\nworld" {
			t.Fatalf("Content = %q, want %q", got.Content, "hello\n\nworld")
		}
		if got.Metadata == nil || got.Metadata["intake_batched"] != "true" ||
			got.Metadata["intake_batch_count"] != "2" {
			t.Fatalf("Metadata = %#v, want intake_batched=true and intake_batch_count=2", got.Metadata)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("timed out waiting for batched message")
	}

	select {
	case got := <-out:
		t.Fatalf("unexpected extra message: %#v", got)
	default:
	}
}

func TestInboundValve_DedupesExactRepeats(t *testing.T) {
	v := newInboundValve(config.IntakeConfig{
		Enabled:             true,
		CoalesceWindowMs:    0,
		MaxBatchMessages:    1,
		MaxBatchChars:       0,
		DedupeWindowSeconds: 1,
	})

	in := make(chan bus.InboundMessage, 10)
	out := make(chan bus.InboundMessage, 10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go v.run(ctx, in, out)

	in <- bus.InboundMessage{Channel: "slack", SenderID: "u1", ChatID: "c1", Content: "ping"}
	in <- bus.InboundMessage{Channel: "slack", SenderID: "u1", ChatID: "c1", Content: "ping"}
	close(in)

	select {
	case got := <-out:
		if got.Content != "ping" {
			t.Fatalf("Content = %q, want %q", got.Content, "ping")
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("timed out waiting for message")
	}

	select {
	case got := <-out:
		t.Fatalf("unexpected extra message: %#v", got)
	default:
	}
}
