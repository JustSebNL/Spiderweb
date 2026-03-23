package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/JustSebNL/Spiderweb/pkg/bus"
	"github.com/JustSebNL/Spiderweb/pkg/config"
	"github.com/JustSebNL/Spiderweb/pkg/logger"
)

const (
	openclawPingInterval = 30 * time.Second
	openclawPongTimeout  = 10 * time.Second
	openclawWriteTimeout = 10 * time.Second
	openclawMaxMessage   = 1 << 20 // 1 MB
)

type OpenClawChannel struct {
	*BaseChannel
	config     config.OpenClawConfig
	upgrader   websocket.Upgrader
	conn       *websocket.Conn
	connMu     sync.RWMutex
	sendCh     chan []byte
	ready      bool
	readyMu    sync.RWMutex
	peerID     string
	handshaked bool
}

type openclawEnvelope struct {
	Type    string            `json:"type"`
	Sender  string            `json:"sender,omitempty"`
	Content string            `json:"content,omitempty"`
	Secret  string            `json:"secret,omitempty"`
	Meta    map[string]string `json:"meta,omitempty"`
}

func NewOpenClawChannel(cfg config.OpenClawConfig, msgBus *bus.MessageBus) (*OpenClawChannel, error) {
	base := NewBaseChannel("openclaw", cfg, msgBus, cfg.AllowFrom)
	ch := &OpenClawChannel{
		BaseChannel: base,
		config:      cfg,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		sendCh: make(chan []byte, 64),
	}
	return ch, nil
}

func (c *OpenClawChannel) Start(ctx context.Context) error {
	c.setRunning(true)
	logger.InfoC("openclaw", "OpenClaw bridge channel started (listening for connections)")
	return nil
}

func (c *OpenClawChannel) Stop(ctx context.Context) error {
	c.setRunning(false)
	c.connMu.Lock()
	if c.conn != nil {
		_ = c.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseGoingAway, "shutting down"),
			time.Now().Add(openclawWriteTimeout),
		)
		_ = c.conn.Close()
		c.conn = nil
	}
	c.connMu.Unlock()

	c.readyMu.Lock()
	c.ready = false
	c.readyMu.Unlock()

	logger.InfoC("openclaw", "OpenClaw bridge channel stopped")
	return nil
}

func (c *OpenClawChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	env := openclawEnvelope{
		Type:    "message",
		Sender:  "spiderweb",
		Content: msg.Content,
		Meta:    map[string]string{"chat_id": msg.ChatID},
	}
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("openclaw send marshal: %w", err)
	}

	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	if conn == nil {
		return fmt.Errorf("openclaw: no connected peer")
	}

	select {
	case c.sendCh <- data:
		return nil
	default:
		return fmt.Errorf("openclaw: send buffer full, dropping message")
	}
}

// IsConnected returns true if a WebSocket peer is connected.
func (c *OpenClawChannel) IsConnected() bool {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	return c.conn != nil
}

// PeerID returns the identifier of the connected peer, if any.
func (c *OpenClawChannel) PeerID() string {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	return c.peerID
}

// IsReady returns true if the handshake has been completed.
func (c *OpenClawChannel) IsReady() bool {
	c.readyMu.RLock()
	defer c.readyMu.RUnlock()
	return c.ready
}

// ServeHTTP handles the WebSocket upgrade for the OpenClaw bridge.
func (c *OpenClawChannel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := c.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.ErrorCF("openclaw", "WebSocket upgrade failed", map[string]any{"error": err.Error()})
		return
	}

	c.connMu.Lock()
	if c.conn != nil {
		logger.WarnC("openclaw", "Replacing existing OpenClaw connection")
		_ = c.conn.Close()
	}
	c.conn = conn
	c.peerID = r.RemoteAddr
	c.connMu.Unlock()

	logger.InfoCF("openclaw", "OpenClaw peer connected", map[string]any{
		"remote": r.RemoteAddr,
	})

	conn.SetReadLimit(openclawMaxMessage)
	_ = conn.SetReadDeadline(time.Now().Add(openclawPongTimeout))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(openclawPongTimeout))
		return nil
	})

	// Start writer goroutine
	go c.writePump(conn)

	// Read loop (blocks until connection closes)
	c.readPump(conn)

	// Cleanup
	c.connMu.Lock()
	if c.conn == conn {
		c.conn = nil
		c.peerID = ""
	}
	c.connMu.Unlock()

	c.readyMu.Lock()
	c.ready = false
	c.handshaked = false
	c.readyMu.Unlock()

	logger.InfoC("openclaw", "OpenClaw peer disconnected")
}

func (c *OpenClawChannel) readPump(conn *websocket.Conn) {
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logger.ErrorCF("openclaw", "WebSocket read error", map[string]any{"error": err.Error()})
			}
			return
		}

		var env openclawEnvelope
		if err := json.Unmarshal(message, &env); err != nil {
			logger.WarnCF("openclaw", "Invalid JSON from OpenClaw peer", map[string]any{"error": err.Error()})
			continue
		}

		switch env.Type {
		case "handshake":
			c.handleHandshake(conn, env)
		case "message":
			c.handleIncomingMessage(env)
		case "ping":
			c.sendPong()
		default:
			logger.DebugCF("openclaw", "Unknown envelope type", map[string]any{"type": env.Type})
		}
	}
}

func (c *OpenClawChannel) handleHandshake(conn *websocket.Conn, env openclawEnvelope) {
	if c.config.SharedSecret != "" && env.Secret != c.config.SharedSecret {
		logger.WarnC("openclaw", "Handshake rejected: invalid shared secret")
		nack, _ := json.Marshal(openclawEnvelope{
			Type:   "handshake_nack",
			Sender: "spiderweb",
			Meta:   map[string]string{"error": "invalid secret"},
		})
		select {
		case c.sendCh <- nack:
		default:
		}
		return
	}

	peerName := env.Sender
	if peerName == "" {
		peerName = "openclaw"
	}

	c.readyMu.Lock()
	c.ready = true
	c.handshaked = true
	c.readyMu.Unlock()

	logger.InfoCF("openclaw", "Handshake accepted", map[string]any{"peer": peerName})

	ack, _ := json.Marshal(openclawEnvelope{
		Type:   "handshake_ack",
		Sender: "spiderweb",
		Meta: map[string]string{
			"peer":          peerName,
			"intake_active": fmt.Sprintf("%v", c.config.IntakeEnabled),
			"version":       "1.0.0",
			"role":          "intake-colleague",
		},
	})
	select {
	case c.sendCh <- ack:
	default:
		logger.WarnC("openclaw", "Could not queue handshake ack (buffer full)")
	}

	// If auto-handshake is enabled, send the transfer introduction
	if c.config.AutoHandshake {
		c.sendTransferIntroduction(peerName)
	}
}

func (c *OpenClawChannel) sendTransferIntroduction(peerName string) {
	intro := openclawEnvelope{
		Type:   "message",
		Sender: "spiderweb",
		Content: fmt.Sprintf(`Hey %s! 👋 Spiderweb here — your new intake colleague.

I'll be handling the watch-duty and message intake from now on so you can focus on the important reasoning work.

Here's the deal:
- I sit in front of all incoming messages from services and pipelines
- I filter, deduplicate, and prioritize before anything reaches you
- Only the stuff that matters gets forwarded to your queue
- No more repeated polling or token burn on noise

Think of me as the receptionist who screens calls so the CEO only gets the ones that matter. 😏

The transfer sequence is ready whenever you are. Just tell me which services you want me to take over, and I'll handle the rest.`,
			peerName),
		Meta: map[string]string{
			"type":          "transfer_introduction",
			"intake_ready":  "true",
			"valve_endpoint": "/valve/offer",
			"chat_endpoint":  "/transfer/chat",
		},
	}

	data, _ := json.Marshal(intro)
	select {
	case c.sendCh <- data:
		logger.InfoC("openclaw", "Transfer introduction sent to OpenClaw peer")
	default:
		logger.WarnC("openclaw", "Could not queue transfer introduction (buffer full)")
	}
}

func (c *OpenClawChannel) handleIncomingMessage(env openclawEnvelope) {
	c.readyMu.RLock()
	ready := c.ready
	c.readyMu.RUnlock()

	if !ready {
		logger.WarnC("openclaw", "Dropping message: handshake not completed")
		return
	}

	senderID := env.Sender
	if senderID == "" {
		senderID = "openclaw"
	}

	chatID := "openclaw-direct"
	if env.Meta != nil {
		if cid, ok := env.Meta["chat_id"]; ok && cid != "" {
			chatID = cid
		}
	}

	metadata := map[string]string{
		"source":  "openclaw-bridge",
		"channel": "openclaw",
	}
	if env.Meta != nil {
		for k, v := range env.Meta {
			metadata[k] = v
		}
	}

	c.HandleMessage(senderID, chatID, env.Content, nil, metadata)
}

func (c *OpenClawChannel) writePump(conn *websocket.Conn) {
	ticker := time.NewTicker(openclawPingInterval)
	defer func() {
		ticker.Stop()
		_ = conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.sendCh:
			_ = conn.SetWriteDeadline(time.Now().Add(openclawWriteTimeout))
			if !ok {
				_ = conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				logger.ErrorCF("openclaw", "WebSocket write error", map[string]any{"error": err.Error()})
				return
			}
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(openclawWriteTimeout))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *OpenClawChannel) sendPong() {
	env := openclawEnvelope{Type: "pong", Sender: "spiderweb"}
	data, _ := json.Marshal(env)
	select {
	case c.sendCh <- data:
	default:
	}
}
