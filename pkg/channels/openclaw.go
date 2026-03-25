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
	config   config.OpenClawConfig
	upgrader websocket.Upgrader
	peers    map[*websocket.Conn]*openclawPeer
	peersMu  sync.RWMutex
}

type openclawEnvelope struct {
	Type    string            `json:"type"`
	Sender  string            `json:"sender,omitempty"`
	Content string            `json:"content,omitempty"`
	Secret  string            `json:"secret,omitempty"`
	Meta    map[string]string `json:"meta,omitempty"`
}

type openclawPeer struct {
	conn       *websocket.Conn
	remoteAddr string
	peerID     string
	sendCh     chan []byte
	ready      bool
	handshaked bool
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
		peers: make(map[*websocket.Conn]*openclawPeer),
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
	c.peersMu.Lock()
	for conn, peer := range c.peers {
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseGoingAway, "shutting down"),
			time.Now().Add(openclawWriteTimeout),
		)
		close(peer.sendCh)
		_ = conn.Close()
		delete(c.peers, conn)
	}
	c.peersMu.Unlock()

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

	readyPeers := c.readyPeers()
	if len(readyPeers) == 0 {
		return fmt.Errorf("openclaw: no ready peer")
	}

	sent := 0
	for _, peer := range readyPeers {
		select {
		case peer.sendCh <- data:
			sent++
		default:
			logger.WarnCF("openclaw", "Peer send buffer full, dropping outbound message", map[string]any{
				"peer": peer.peerID,
			})
		}
	}
	if sent == 0 {
		return fmt.Errorf("openclaw: all peer send buffers full, dropping message")
	}
	return nil
}

// IsConnected returns true if a WebSocket peer is connected.
func (c *OpenClawChannel) IsConnected() bool {
	c.peersMu.RLock()
	defer c.peersMu.RUnlock()
	return len(c.peers) > 0
}

// PeerID returns the identifier of the connected peer, if any.
func (c *OpenClawChannel) PeerID() string {
	c.peersMu.RLock()
	defer c.peersMu.RUnlock()
	for _, peer := range c.peers {
		if peer.peerID != "" {
			return peer.peerID
		}
	}
	return ""
}

// IsReady returns true if the handshake has been completed.
func (c *OpenClawChannel) IsReady() bool {
	c.peersMu.RLock()
	defer c.peersMu.RUnlock()
	for _, peer := range c.peers {
		if peer.ready {
			return true
		}
	}
	return false
}

// ServeHTTP handles the WebSocket upgrade for the OpenClaw bridge.
func (c *OpenClawChannel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := c.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.ErrorCF("openclaw", "WebSocket upgrade failed", map[string]any{"error": err.Error()})
		return
	}

	peer := c.registerPeer(conn, r.RemoteAddr)

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
	go c.writePump(peer)

	// Read loop (blocks until connection closes)
	c.readPump(peer)

	// Cleanup
	c.unregisterPeer(conn)

	logger.InfoC("openclaw", "OpenClaw peer disconnected")
}

func (c *OpenClawChannel) readPump(peer *openclawPeer) {
	defer peer.conn.Close()

	for {
		_, message, err := peer.conn.ReadMessage()
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
			c.handleHandshake(peer, env)
		case "message":
			c.handleIncomingMessage(peer, env)
		case "ping":
			c.sendPong(peer)
		default:
			logger.DebugCF("openclaw", "Unknown envelope type", map[string]any{"type": env.Type})
		}
	}
}

func (c *OpenClawChannel) handleHandshake(peer *openclawPeer, env openclawEnvelope) {
	if c.config.SharedSecret != "" && env.Secret != c.config.SharedSecret {
		logger.WarnC("openclaw", "Handshake rejected: invalid shared secret")
		nack, _ := json.Marshal(openclawEnvelope{
			Type:   "handshake_nack",
			Sender: "spiderweb",
			Meta:   map[string]string{"error": "invalid secret"},
		})
		c.queueToPeer(peer, nack)
		return
	}

	peerName := env.Sender
	if peerName == "" {
		peerName = "openclaw"
	}

	c.peersMu.Lock()
	peer.ready = true
	peer.handshaked = true
	peer.peerID = peerName
	c.peersMu.Unlock()

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
	if !c.queueToPeer(peer, ack) {
		logger.WarnC("openclaw", "Could not queue handshake ack (buffer full)")
	}

	// If auto-handshake is enabled, send the transfer introduction
	if c.config.AutoHandshake {
		c.sendTransferIntroduction(peer, peerName)
	}
}

func (c *OpenClawChannel) sendTransferIntroduction(peer *openclawPeer, peerName string) {
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
			"type":           "transfer_introduction",
			"intake_ready":   "true",
			"valve_endpoint": "/valve/offer",
			"chat_endpoint":  "/transfer/chat",
		},
	}

	data, _ := json.Marshal(intro)
	if c.queueToPeer(peer, data) {
		logger.InfoC("openclaw", "Transfer introduction sent to OpenClaw peer")
	} else {
		logger.WarnC("openclaw", "Could not queue transfer introduction (buffer full)")
	}
}

func (c *OpenClawChannel) handleIncomingMessage(peer *openclawPeer, env openclawEnvelope) {
	c.peersMu.RLock()
	ready := peer.ready
	c.peersMu.RUnlock()
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

func (c *OpenClawChannel) writePump(peer *openclawPeer) {
	ticker := time.NewTicker(openclawPingInterval)
	defer func() {
		ticker.Stop()
		_ = peer.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-peer.sendCh:
			_ = peer.conn.SetWriteDeadline(time.Now().Add(openclawWriteTimeout))
			if !ok {
				_ = peer.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := peer.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				logger.ErrorCF("openclaw", "WebSocket write error", map[string]any{"error": err.Error()})
				return
			}
		case <-ticker.C:
			_ = peer.conn.SetWriteDeadline(time.Now().Add(openclawWriteTimeout))
			if err := peer.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *OpenClawChannel) sendPong(peer *openclawPeer) {
	env := openclawEnvelope{Type: "pong", Sender: "spiderweb"}
	data, _ := json.Marshal(env)
	c.queueToPeer(peer, data)
}

func (c *OpenClawChannel) registerPeer(conn *websocket.Conn, remoteAddr string) *openclawPeer {
	if conn == nil {
		conn = &websocket.Conn{}
	}
	peer := &openclawPeer{
		conn:       conn,
		remoteAddr: remoteAddr,
		peerID:     remoteAddr,
		sendCh:     make(chan []byte, 64),
	}
	c.peersMu.Lock()
	c.peers[conn] = peer
	c.peersMu.Unlock()
	return peer
}

func (c *OpenClawChannel) unregisterPeer(conn *websocket.Conn) {
	c.peersMu.Lock()
	peer, ok := c.peers[conn]
	if ok {
		delete(c.peers, conn)
	}
	c.peersMu.Unlock()
	if ok {
		close(peer.sendCh)
	}
}

func (c *OpenClawChannel) readyPeers() []*openclawPeer {
	c.peersMu.RLock()
	defer c.peersMu.RUnlock()
	peers := make([]*openclawPeer, 0, len(c.peers))
	for _, peer := range c.peers {
		if peer.ready {
			peers = append(peers, peer)
		}
	}
	return peers
}

func (c *OpenClawChannel) queueToPeer(peer *openclawPeer, data []byte) bool {
	if peer == nil {
		return false
	}
	select {
	case peer.sendCh <- data:
		return true
	default:
		return false
	}
}
