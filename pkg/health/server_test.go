package health

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JustSebNL/Spiderweb/pkg/bus"
	"github.com/JustSebNL/Spiderweb/pkg/constants"
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
