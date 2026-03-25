package channels

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/JustSebNL/Spiderweb/pkg/bus"
	"github.com/JustSebNL/Spiderweb/pkg/config"
)

func TestOpenClawHandshake_AcceptsAndPublishesInboundAfterHandshake(t *testing.T) {
	t.Parallel()

	msgBus := bus.NewMessageBus()
	ch, err := NewOpenClawChannel(config.OpenClawConfig{
		Enabled:       true,
		SharedSecret:  "top-secret",
		AutoHandshake: true,
		IntakeEnabled: true,
	}, msgBus)
	if err != nil {
		t.Fatalf("new openclaw channel: %v", err)
	}
	peer := ch.registerPeer(nil, "test-openclaw")

	ch.handleHandshake(peer, openclawEnvelope{
		Type:    "handshake",
		Sender:  "openclaw",
		Secret:  "top-secret",
		Content: "requesting intake handoff",
	})

	ack := readQueuedEnvelope(t, peer.sendCh)
	if ack.Type != "handshake_ack" {
		t.Fatalf("expected handshake_ack, got %+v", ack)
	}
	if ack.Meta["peer"] != "openclaw" {
		t.Fatalf("expected ack peer=openclaw, got %+v", ack.Meta)
	}

	intro := readQueuedEnvelope(t, peer.sendCh)
	if intro.Type != "message" {
		t.Fatalf("expected transfer introduction message, got %+v", intro)
	}
	if intro.Meta["type"] != "transfer_introduction" {
		t.Fatalf("expected transfer introduction meta, got %+v", intro.Meta)
	}

	if !ch.IsReady() {
		t.Fatalf("channel should be ready after successful handshake")
	}

	ch.handleIncomingMessage(peer, openclawEnvelope{
		Type:    "message",
		Sender:  "openclaw",
		Content: "hello from peer",
		Meta: map[string]string{
			"chat_id": "handoff-1",
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	msg, ok := msgBus.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("expected inbound message on bus")
	}
	if msg.Channel != "openclaw" {
		t.Fatalf("channel = %q, want %q", msg.Channel, "openclaw")
	}
	if msg.SenderID != "openclaw" {
		t.Fatalf("sender_id = %q, want %q", msg.SenderID, "openclaw")
	}
	if msg.ChatID != "handoff-1" {
		t.Fatalf("chat_id = %q, want %q", msg.ChatID, "handoff-1")
	}
	if msg.Content != "hello from peer" {
		t.Fatalf("content = %q, want %q", msg.Content, "hello from peer")
	}
	if msg.Metadata["source"] != "openclaw-bridge" {
		t.Fatalf("metadata = %+v", msg.Metadata)
	}
}

func TestOpenClawHandshake_RejectsInvalidSecret(t *testing.T) {
	t.Parallel()

	msgBus := bus.NewMessageBus()
	ch, err := NewOpenClawChannel(config.OpenClawConfig{
		Enabled:      true,
		SharedSecret: "top-secret",
	}, msgBus)
	if err != nil {
		t.Fatalf("new openclaw channel: %v", err)
	}
	peer := ch.registerPeer(nil, "test-openclaw")

	ch.handleHandshake(peer, openclawEnvelope{
		Type:    "handshake",
		Sender:  "openclaw",
		Secret:  "wrong-secret",
		Content: "requesting intake handoff",
	})

	resp := readQueuedEnvelope(t, peer.sendCh)
	if resp.Type != "handshake_nack" {
		t.Fatalf("expected handshake_nack, got %+v", resp)
	}
	if resp.Meta["error"] != "invalid secret" {
		t.Fatalf("expected invalid secret error, got %+v", resp.Meta)
	}
	if ch.IsReady() {
		t.Fatalf("channel should not be ready after invalid handshake")
	}

	ch.handleIncomingMessage(peer, openclawEnvelope{
		Type:    "message",
		Sender:  "openclaw",
		Content: "should be dropped",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	if _, ok := msgBus.ConsumeInbound(ctx); ok {
		t.Fatal("unexpected inbound message after rejected handshake")
	}
}

func TestOpenClawSend_BroadcastsToAllReadyPeers(t *testing.T) {
	t.Parallel()

	msgBus := bus.NewMessageBus()
	ch, err := NewOpenClawChannel(config.OpenClawConfig{Enabled: true}, msgBus)
	if err != nil {
		t.Fatalf("new openclaw channel: %v", err)
	}

	peerA := ch.registerPeer(nil, "peer-a")
	peerB := ch.registerPeer(nil, "peer-b")
	ch.handleHandshake(peerA, openclawEnvelope{Type: "handshake", Sender: "peer-a"})
	ch.handleHandshake(peerB, openclawEnvelope{Type: "handshake", Sender: "peer-b"})

	_ = readQueuedEnvelope(t, peerA.sendCh)
	_ = readQueuedEnvelope(t, peerB.sendCh)

	if err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "fanout-1",
		Content: "fanout hello",
	}); err != nil {
		t.Fatalf("send fanout: %v", err)
	}

	msgA := readQueuedEnvelope(t, peerA.sendCh)
	msgB := readQueuedEnvelope(t, peerB.sendCh)

	if msgA.Content != "fanout hello" || msgB.Content != "fanout hello" {
		t.Fatalf("expected both peers to receive the outbound message, got A=%+v B=%+v", msgA, msgB)
	}
}

func readQueuedEnvelope(t *testing.T, sendCh <-chan []byte) openclawEnvelope {
	t.Helper()
	select {
	case data := <-sendCh:
		var env openclawEnvelope
		if err := json.Unmarshal(data, &env); err != nil {
			t.Fatalf("unmarshal envelope: %v", err)
		}
		return env
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for queued envelope")
	}
	return openclawEnvelope{}
}
