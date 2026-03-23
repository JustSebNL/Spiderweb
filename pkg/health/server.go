package health

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/JustSebNL/Spiderweb/pkg/bus"
	"github.com/JustSebNL/Spiderweb/pkg/constants"
)

type Server struct {
	server    *http.Server
	mu        sync.RWMutex
	ready     bool
	checks    map[string]Check
	startTime time.Time
	msgBus    *bus.MessageBus
	workspace string
	nextID    uint64
	offers    map[uint64]offerEntry
	receipts  map[uint64]time.Time
	chats     map[string]chatEntry
	wsHandler map[string]http.Handler
}

type offerEntry struct {
	queue   string
	expires time.Time
}

type chatEntry struct {
	path    string
	token   string
	created time.Time
}

type transferChatMeta struct {
	ChatID      string `json:"chat_id"`
	Token       string `json:"token"`
	FilePath    string `json:"file_path"`
	CreatedAt   string `json:"created_at"`
	ServiceName string `json:"service_name"`
}

type Check struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type StatusResponse struct {
	Status string           `json:"status"`
	Uptime string           `json:"uptime"`
	Checks map[string]Check `json:"checks,omitempty"`
}

type ValveResponse struct {
	State      int    `json:"state"`
	Message    string `json:"message,omitempty"`
	OfferID    uint64 `json:"offer_id,omitempty"`
	ReceiptID  uint64 `json:"receipt_id,omitempty"`
	InboundLen int    `json:"inbound_len,omitempty"`
	InboundCap int    `json:"inbound_cap,omitempty"`
}

type ValveOfferRequest struct {
	Channel    string            `json:"channel"`
	SenderID   string            `json:"sender_id"`
	ChatID     string            `json:"chat_id"`
	Content    string            `json:"content"`
	Media      []string          `json:"media,omitempty"`
	SessionKey string            `json:"session_key,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type ValveTransferRequest struct {
	OfferID    uint64            `json:"offer_id"`
	Channel    string            `json:"channel"`
	SenderID   string            `json:"sender_id"`
	ChatID     string            `json:"chat_id"`
	Content    string            `json:"content"`
	Media      []string          `json:"media,omitempty"`
	SessionKey string            `json:"session_key,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type TransferChatStartRequest struct {
	ServiceName string `json:"service_name"`
}

type TransferChatSendRequest struct {
	ChatID  string `json:"chat_id"`
	Token   string `json:"token"`
	Sender  string `json:"sender"`
	Content string `json:"content"`
}

type TransferChatTailResponse struct {
	ChatID   string `json:"chat_id"`
	Content  string `json:"content"`
	FilePath string `json:"file_path,omitempty"`
}

func NewServer(host string, port int) *Server {
	mux := http.NewServeMux()
	s := &Server{
		ready:     false,
		checks:    make(map[string]Check),
		startTime: time.Now(),
		offers:    make(map[uint64]offerEntry),
		receipts:  make(map[uint64]time.Time),
		chats:     make(map[string]chatEntry),
		wsHandler: make(map[string]http.Handler),
	}

	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/ready", s.readyHandler)
	mux.HandleFunc("/valve/state", s.valveStateHandler)
	mux.HandleFunc("/valve/offer", s.valveOfferHandler)
	mux.HandleFunc("/valve/handshake/offer", s.valveHandshakeOfferHandler)
	mux.HandleFunc("/valve/handshake/transfer", s.valveHandshakeTransferHandler)
	mux.HandleFunc("/valve/handshake/confirm", s.valveHandshakeConfirmHandler)
	mux.HandleFunc("/intake/stats", s.intakeStatsHandler)
	mux.HandleFunc("/transfer/chat/start", s.transferChatStartHandler)
	mux.HandleFunc("/transfer/chat/send", s.transferChatSendHandler)
	mux.HandleFunc("/transfer/chat/tail", s.transferChatTailHandler)
	mux.HandleFunc("/transfer/chat", s.transferChatUIHandler)
	mux.HandleFunc("/bridge/", s.bridgeRouter)

	addr := fmt.Sprintf("%s:%d", host, port)
	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	return s
}

func (s *Server) nextToken() uint64 {
	return atomic.AddUint64(&s.nextID, 1)
}

func (s *Server) pruneHandshake(now time.Time) {
	for k, v := range s.offers {
		if now.After(v.expires) {
			delete(s.offers, k)
		}
	}
	for k, t := range s.receipts {
		if now.Sub(t) > 10*time.Minute {
			delete(s.receipts, k)
		}
	}
}

func (s *Server) SetMessageBus(msgBus *bus.MessageBus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgBus = msgBus
}

func (s *Server) SetWorkspace(workspace string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workspace = workspace
}

// RegisterWSHandler registers a WebSocket handler at the given path prefix.
func (s *Server) RegisterWSHandler(path string, handler http.Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.wsHandler[path] = handler
}

func (s *Server) bridgeRouter(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	handler, ok := s.wsHandler[r.URL.Path]
	s.mu.RUnlock()

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{"error": "bridge not found"})
		return
	}

	handler.ServeHTTP(w, r)
}

func (s *Server) Start() error {
	s.mu.Lock()
	s.ready = true
	s.mu.Unlock()
	return s.server.ListenAndServe()
}

func (s *Server) StartContext(ctx context.Context) error {
	s.mu.Lock()
	s.ready = true
	s.mu.Unlock()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return s.server.Shutdown(context.Background())
	}
}

func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	s.ready = false
	s.mu.Unlock()
	return s.server.Shutdown(ctx)
}

func (s *Server) SetReady(ready bool) {
	s.mu.Lock()
	s.ready = ready
	s.mu.Unlock()
}

func (s *Server) RegisterCheck(name string, checkFn func() (bool, string)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	status, msg := checkFn()
	s.checks[name] = Check{
		Name:      name,
		Status:    statusString(status),
		Message:   msg,
		Timestamp: time.Now(),
	}
}

func (s *Server) valveState(queue string) (state constants.ValveState, inboundLen int, inboundCap int) {
	s.mu.RLock()
	ready := s.ready
	msgBus := s.msgBus
	s.mu.RUnlock()

	if !ready {
		return constants.ValveStateReject, 0, 0
	}
	if msgBus == nil {
		return constants.ValveStateSystemError, 0, 0
	}

	highLen, highCap, lowLen, lowCap, closed := msgBus.InboundStatsByQueue()
	if closed {
		return constants.ValveStateReject, highLen + lowLen, highCap + lowCap
	}

	if queue == "interrupt" {
		if highCap > 0 && highLen >= highCap {
			return constants.ValveStateBusy, highLen, highCap
		}
		return constants.ValveStateAccept, highLen, highCap
	}

	if lowCap > 0 && lowLen >= lowCap {
		return constants.ValveStateBusy, lowLen, lowCap
	}
	return constants.ValveStateAccept, lowLen, lowCap
}

func (s *Server) valveStateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	queue := r.URL.Query().Get("name")
	if queue == "" {
		queue = "normal"
	}
	verbose := r.URL.Query().Get("verbose") == "1"
	state, inLen, inCap := s.valveState(queue)
	message := ""
	if verbose {
		message = constants.ValveStateString(state)
	}
	json.NewEncoder(w).Encode(ValveResponse{
		State:      int(state),
		Message:    message,
		InboundLen: inLen,
		InboundCap: inCap,
	})
}

func (s *Server) valveOfferHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	verbose := r.URL.Query().Get("verbose") == "1"

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		msg := ""
		if verbose {
			msg = constants.ValveStateString(constants.ValveStateReject)
		}
		json.NewEncoder(w).Encode(ValveResponse{State: int(constants.ValveStateReject), Message: msg})
		return
	}

	s.mu.RLock()
	msgBus := s.msgBus
	s.mu.RUnlock()
	if msgBus == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		msg := ""
		if verbose {
			msg = constants.ValveStateString(constants.ValveStateSystemError)
		}
		json.NewEncoder(w).Encode(ValveResponse{State: int(constants.ValveStateSystemError), Message: msg})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req ValveOfferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		msg := ""
		if verbose {
			msg = constants.ValveStateString(constants.ValveStateUnknown)
		}
		json.NewEncoder(w).Encode(ValveResponse{State: int(constants.ValveStateUnknown), Message: msg})
		return
	}

	if req.Channel == "" || req.ChatID == "" || req.SenderID == "" || req.Content == "" {
		w.WriteHeader(http.StatusBadRequest)
		msg := ""
		if verbose {
			msg = constants.ValveStateString(constants.ValveStateUnknown)
		}
		json.NewEncoder(w).Encode(ValveResponse{State: int(constants.ValveStateUnknown), Message: msg})
		return
	}

	queue := r.URL.Query().Get("name")
	if queue == "" {
		queue = "normal"
	}
	if req.Metadata != nil && req.Metadata["valve"] == "interrupt" {
		queue = "interrupt"
	}
	if req.Metadata != nil && req.Metadata["priority"] == "high" {
		queue = "interrupt"
	}

	state, inLen, inCap := s.valveState(queue)
	if state != constants.ValveStateAccept {
		if queue == "normal" {
			allowInterrupt := false
			if req.Metadata != nil && req.Metadata["allow_interrupt"] == "true" {
				allowInterrupt = true
			}
			if req.Metadata != nil && req.Metadata["priority"] == "high" {
				allowInterrupt = true
			}
			if r.URL.Query().Get("allow_interrupt") == "1" {
				allowInterrupt = true
			}

			interruptState, interruptLen, interruptCap := s.valveState("interrupt")
			if allowInterrupt && interruptState == constants.ValveStateAccept {
				if req.Metadata == nil {
					req.Metadata = make(map[string]string)
				}
				req.Metadata["valve"] = "interrupt"
				queue = "interrupt"
				state = constants.ValveStateInterruptAccepted
				inLen = interruptLen
				inCap = interruptCap
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
				msg := ""
				if verbose {
					msg = constants.ValveStateString(state)
				}
				json.NewEncoder(w).Encode(ValveResponse{
					State:      int(state),
					Message:    msg,
					InboundLen: inLen,
					InboundCap: inCap,
				})
				return
			}
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			msg := ""
			if verbose {
				msg = constants.ValveStateString(state)
			}
			json.NewEncoder(w).
				Encode(ValveResponse{State: int(state), Message: msg, InboundLen: inLen, InboundCap: inCap})
			return
		}
	}

	ok := msgBus.TryPublishInbound(bus.InboundMessage{
		Channel:    req.Channel,
		SenderID:   req.SenderID,
		ChatID:     req.ChatID,
		Content:    req.Content,
		Media:      req.Media,
		SessionKey: req.SessionKey,
		Metadata:   req.Metadata,
	})
	if !ok {
		s2, l2, c2 := s.valveState(queue)
		if s2 == constants.ValveStateAccept {
			s2 = constants.ValveStateBusy
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		msg := ""
		if verbose {
			msg = constants.ValveStateString(s2)
		}
		json.NewEncoder(w).Encode(ValveResponse{State: int(s2), Message: msg, InboundLen: l2, InboundCap: c2})
		return
	}

	if state == constants.ValveStateInterruptAccepted {
		msg := ""
		if verbose {
			msg = constants.ValveStateString(state)
		}
		json.NewEncoder(w).Encode(ValveResponse{
			State:      int(state),
			Message:    msg,
			InboundLen: inLen,
			InboundCap: inCap,
		})
		return
	}
	state, inLen, inCap = s.valveState(queue)
	msg := ""
	if verbose {
		msg = constants.ValveStateString(state)
	}
	json.NewEncoder(w).Encode(ValveResponse{State: int(state), Message: msg, InboundLen: inLen, InboundCap: inCap})
}

func (s *Server) valveHandshakeOfferHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	verbose := r.URL.Query().Get("verbose") == "1"

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		msg := ""
		if verbose {
			msg = constants.ValveStateString(constants.ValveStateReject)
		}
		json.NewEncoder(w).Encode(ValveResponse{State: int(constants.ValveStateReject), Message: msg})
		return
	}

	queue := r.URL.Query().Get("name")
	if queue == "" {
		queue = "normal"
	}
	state, inLen, inCap := s.valveState(queue)
	if state != constants.ValveStateAccept {
		w.WriteHeader(http.StatusServiceUnavailable)
		msg := ""
		if verbose {
			msg = constants.ValveStateString(state)
		}
		json.NewEncoder(w).Encode(ValveResponse{State: int(state), Message: msg, InboundLen: inLen, InboundCap: inCap})
		return
	}

	now := time.Now()
	offerID := s.nextToken()
	s.mu.Lock()
	s.pruneHandshake(now)
	s.offers[offerID] = offerEntry{queue: queue, expires: now.Add(30 * time.Second)}
	s.mu.Unlock()

	msg := ""
	if verbose {
		msg = constants.ValveStateString(constants.ValveStateAccept)
	}
	json.NewEncoder(w).Encode(ValveResponse{State: int(constants.ValveStateAccept), Message: msg, OfferID: offerID, InboundLen: inLen, InboundCap: inCap})
}

func (s *Server) valveHandshakeTransferHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	verbose := r.URL.Query().Get("verbose") == "1"

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		msg := ""
		if verbose {
			msg = constants.ValveStateString(constants.ValveStateReject)
		}
		json.NewEncoder(w).Encode(ValveResponse{State: int(constants.ValveStateReject), Message: msg})
		return
	}

	s.mu.RLock()
	msgBus := s.msgBus
	s.mu.RUnlock()
	if msgBus == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		msg := ""
		if verbose {
			msg = constants.ValveStateString(constants.ValveStateSystemError)
		}
		json.NewEncoder(w).Encode(ValveResponse{State: int(constants.ValveStateSystemError), Message: msg})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req ValveTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		msg := ""
		if verbose {
			msg = constants.ValveStateString(constants.ValveStateUnknown)
		}
		json.NewEncoder(w).Encode(ValveResponse{State: int(constants.ValveStateUnknown), Message: msg})
		return
	}

	if req.OfferID == 0 || req.Channel == "" || req.ChatID == "" || req.SenderID == "" || req.Content == "" {
		w.WriteHeader(http.StatusBadRequest)
		msg := ""
		if verbose {
			msg = constants.ValveStateString(constants.ValveStateUnknown)
		}
		json.NewEncoder(w).Encode(ValveResponse{State: int(constants.ValveStateUnknown), Message: msg})
		return
	}

	now := time.Now()
	queue := ""
	s.mu.Lock()
	s.pruneHandshake(now)
	if offer, ok := s.offers[req.OfferID]; ok && now.Before(offer.expires) {
		queue = offer.queue
		delete(s.offers, req.OfferID)
	}
	s.mu.Unlock()

	if queue == "" {
		w.WriteHeader(http.StatusServiceUnavailable)
		msg := ""
		if verbose {
			msg = constants.ValveStateString(constants.ValveStateUnknown)
		}
		json.NewEncoder(w).Encode(ValveResponse{State: int(constants.ValveStateUnknown), Message: msg})
		return
	}

	state, inLen, inCap := s.valveState(queue)
	if state != constants.ValveStateAccept {
		w.WriteHeader(http.StatusServiceUnavailable)
		msg := ""
		if verbose {
			msg = constants.ValveStateString(state)
		}
		json.NewEncoder(w).Encode(ValveResponse{State: int(state), Message: msg, InboundLen: inLen, InboundCap: inCap})
		return
	}

	if req.Metadata == nil {
		req.Metadata = make(map[string]string)
	}
	req.Metadata["offer_id"] = fmt.Sprintf("%d", req.OfferID)

	ok := msgBus.TryPublishInbound(bus.InboundMessage{
		Channel:    req.Channel,
		SenderID:   req.SenderID,
		ChatID:     req.ChatID,
		Content:    req.Content,
		Media:      req.Media,
		SessionKey: req.SessionKey,
		Metadata:   req.Metadata,
	})
	if !ok {
		w.WriteHeader(http.StatusServiceUnavailable)
		msg := ""
		if verbose {
			msg = constants.ValveStateString(constants.ValveStateBusy)
		}
		json.NewEncoder(w).Encode(ValveResponse{State: int(constants.ValveStateBusy), Message: msg, InboundLen: inLen, InboundCap: inCap})
		return
	}

	receiptID := s.nextToken()
	s.mu.Lock()
	s.pruneHandshake(now)
	s.receipts[receiptID] = now
	s.mu.Unlock()

	msg := ""
	if verbose {
		msg = constants.ValveStateString(constants.ValveStateAccept)
	}
	json.NewEncoder(w).Encode(ValveResponse{State: int(constants.ValveStateAccept), Message: msg, ReceiptID: receiptID, InboundLen: inLen, InboundCap: inCap})
}

func (s *Server) valveHandshakeConfirmHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	verbose := r.URL.Query().Get("verbose") == "1"
	receipt := r.URL.Query().Get("receipt_id")
	if receipt == "" {
		w.WriteHeader(http.StatusBadRequest)
		msg := ""
		if verbose {
			msg = constants.ValveStateString(constants.ValveStateUnknown)
		}
		json.NewEncoder(w).Encode(ValveResponse{State: int(constants.ValveStateUnknown), Message: msg})
		return
	}

	var receiptID uint64
	_, err := fmt.Sscanf(receipt, "%d", &receiptID)
	if err != nil || receiptID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		msg := ""
		if verbose {
			msg = constants.ValveStateString(constants.ValveStateUnknown)
		}
		json.NewEncoder(w).Encode(ValveResponse{State: int(constants.ValveStateUnknown), Message: msg})
		return
	}

	now := time.Now()
	found := false
	s.mu.Lock()
	s.pruneHandshake(now)
	if _, ok := s.receipts[receiptID]; ok {
		found = true
	}
	s.mu.Unlock()

	state := constants.ValveStateReject
	if found {
		state = constants.ValveStateAccept
	}
	msg := ""
	if verbose {
		msg = constants.ValveStateString(state)
	}
	json.NewEncoder(w).Encode(ValveResponse{State: int(state), Message: msg, ReceiptID: receiptID})
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	uptime := time.Since(s.startTime)
	resp := StatusResponse{
		Status: "ok",
		Uptime: uptime.String(),
	}

	json.NewEncoder(w).Encode(resp)
}

func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	s.mu.RLock()
	ready := s.ready
	checks := make(map[string]Check)
	for k, v := range s.checks {
		checks[k] = v
	}
	s.mu.RUnlock()

	if !ready {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(StatusResponse{
			Status: "not ready",
			Checks: checks,
		})
		return
	}

	for _, check := range checks {
		if check.Status == "fail" {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(StatusResponse{
				Status: "not ready",
				Checks: checks,
			})
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	uptime := time.Since(s.startTime)
	json.NewEncoder(w).Encode(StatusResponse{
		Status: "ready",
		Uptime: uptime.String(),
		Checks: checks,
	})
}

func (s *Server) intakeStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	days := 7
	if v := r.URL.Query().Get("days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			days = n
		}
	}
	if days <= 0 {
		days = 1
	}
	if days > 30 {
		days = 30
	}

	s.mu.RLock()
	msgBus := s.msgBus
	s.mu.RUnlock()
	if msgBus == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]any{"error": "message bus not set"})
		return
	}

	json.NewEncoder(w).Encode(msgBus.InboundUsageSnapshot(days))
}

func (s *Server) transferChatStartHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]any{"error": "method not allowed"})
		return
	}

	s.mu.RLock()
	workspace := s.workspace
	s.mu.RUnlock()
	if workspace == "" {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]any{"error": "workspace not set"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req TransferChatStartRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.ServiceName == "" {
		req.ServiceName = "UNKNOWN"
	}

	now := time.Now()
	chatID := fmt.Sprintf("%d", s.nextToken())
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]any{"error": "failed to generate token"})
		return
	}
	token := hex.EncodeToString(b)

	dir := filepath.Join(workspace, "transfer-logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]any{"error": "failed to create transfer-logs directory"})
		return
	}
	path := filepath.Join(dir, "chat_"+chatID+".md")
	metaPath := filepath.Join(dir, "chat_"+chatID+".meta.json")

	header := fmt.Sprintf(
		"# Transfer Chat %s\n\nService: %s\nCreated: %s\n\n---\n",
		chatID,
		req.ServiceName,
		now.Format(time.RFC3339),
	)
	if err := os.WriteFile(path, []byte(header), 0o644); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]any{"error": "failed to write chat file"})
		return
	}

	meta := transferChatMeta{
		ChatID:      chatID,
		Token:       token,
		FilePath:    path,
		CreatedAt:   now.Format(time.RFC3339),
		ServiceName: req.ServiceName,
	}
	bMeta, err := json.Marshal(meta)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]any{"error": "failed to create chat meta"})
		return
	}
	if err := os.WriteFile(metaPath, bMeta, 0o644); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]any{"error": "failed to write chat meta"})
		return
	}

	s.mu.Lock()
	s.chats[chatID] = chatEntry{path: path, token: token, created: now}
	s.mu.Unlock()

	baseURL := requestBaseURL(r)
	uiURL := baseURL + "/transfer/chat?chat_id=" + chatID + "&token=" + token

	json.NewEncoder(w).Encode(map[string]any{
		"chat_id":  chatID,
		"token":    token,
		"file":     path,
		"send_url": "/transfer/chat/send",
		"tail_url": "/transfer/chat/tail?chat_id=" + chatID,
		"ui_url":   uiURL,
	})
}

func requestBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	} else if v := r.Header.Get("X-Forwarded-Proto"); v != "" {
		scheme = v
	}
	host := r.Host
	if host == "" {
		host = "localhost"
	}
	return scheme + "://" + host
}

func (s *Server) transferChatUIHandler(w http.ResponseWriter, r *http.Request) {
	chatID := r.URL.Query().Get("chat_id")
	token := r.URL.Query().Get("token")
	if chatID == "" || token == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "chat_id and token required")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!doctype html>
<html>
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Transfer Chat %s</title>
  <style>
    body { font-family: ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, Arial, sans-serif; margin: 24px; }
    .meta { color: #555; margin-bottom: 12px; }
    pre { white-space: pre-wrap; background: #0b1020; color: #e6e6e6; padding: 12px; border-radius: 8px; min-height: 320px; }
    .row { display: flex; gap: 8px; margin-top: 10px; }
    input, textarea, button { font-size: 14px; padding: 10px; border-radius: 8px; border: 1px solid #ccc; }
    input { width: 180px; }
    textarea { flex: 1; min-height: 44px; }
    button { cursor: pointer; background: #111827; color: #fff; border: 1px solid #111827; }
    button:disabled { opacity: 0.5; cursor: default; }
  </style>
</head>
<body>
  <h1>Transfer Chat</h1>
  <div class="meta">Chat ID: <code>%s</code> · Logs: <code>transfer-logs/chat_%s.md</code></div>
  <pre id="log">Loading…</pre>
  <div class="row">
    <input id="sender" placeholder="Sender" value="OpenClaw" />
    <textarea id="msg" placeholder="Message"></textarea>
    <button id="send">Send</button>
  </div>
  <script>
    const chatId = %q;
    const token = %q;
    const logEl = document.getElementById("log");
    const senderEl = document.getElementById("sender");
    const msgEl = document.getElementById("msg");
    const sendBtn = document.getElementById("send");

    async function refresh() {
      try {
        const url = "/transfer/chat/tail?chat_id=" + encodeURIComponent(chatId) + "&token=" + encodeURIComponent(token) + "&lines=200";
        const res = await fetch(url);
        const json = await res.json();
        if (!res.ok) throw new Error(json.error || res.statusText);
        logEl.textContent = json.content || "";
      } catch (e) {
        logEl.textContent = String(e);
      }
    }

    async function send() {
      const sender = senderEl.value.trim() || "OpenClaw";
      const content = msgEl.value.trim();
      if (!content) return;
      sendBtn.disabled = true;
      try {
        const res = await fetch("/transfer/chat/send", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ chat_id: chatId, token, sender, content }),
        });
        const json = await res.json();
        if (!res.ok) throw new Error(json.error || res.statusText);
        msgEl.value = "";
        await refresh();
      } catch (e) {
        alert(String(e));
      } finally {
        sendBtn.disabled = false;
      }
    }

    sendBtn.addEventListener("click", send);
    msgEl.addEventListener("keydown", (e) => {
      if ((e.ctrlKey || e.metaKey) && e.key === "Enter") send();
    });

    refresh();
    setInterval(refresh, 1500);
  </script>
</body>
</html>`, chatID, chatID, chatID, chatID, token)
}

func (s *Server) loadChat(chatID string) (chatEntry, bool) {
	s.mu.RLock()
	chat, ok := s.chats[chatID]
	workspace := s.workspace
	s.mu.RUnlock()
	if ok {
		return chat, true
	}
	if workspace == "" {
		return chatEntry{}, false
	}

	metaPath := filepath.Join(workspace, "transfer-logs", "chat_"+chatID+".meta.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return chatEntry{}, false
	}
	var meta transferChatMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return chatEntry{}, false
	}
	if meta.FilePath == "" || meta.Token == "" {
		return chatEntry{}, false
	}
	created := time.Time{}
	if meta.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, meta.CreatedAt); err == nil {
			created = t
		}
	}
	chat = chatEntry{path: meta.FilePath, token: meta.Token, created: created}
	s.mu.Lock()
	s.chats[chatID] = chat
	s.mu.Unlock()
	return chat, true
}

func (s *Server) transferChatSendHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]any{"error": "method not allowed"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req TransferChatSendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"error": "invalid json"})
		return
	}
	if req.ChatID == "" || req.Token == "" || req.Sender == "" || req.Content == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"error": "chat_id, token, sender, content required"})
		return
	}

	now := time.Now()
	chat, ok := s.loadChat(req.ChatID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{"error": "chat not found"})
		return
	}
	if chat.token != req.Token {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{"error": "invalid token"})
		return
	}

	line := fmt.Sprintf("\n[%s] %s: %s\n", now.Format(time.RFC3339), req.Sender, req.Content)
	f, err := os.OpenFile(chat.path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]any{"error": "failed to open chat file"})
		return
	}
	_, _ = f.WriteString(line)
	_ = f.Close()

	json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func (s *Server) transferChatTailHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	chatID := r.URL.Query().Get("chat_id")
	token := r.URL.Query().Get("token")
	if chatID == "" || token == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"error": "chat_id and token required"})
		return
	}
	maxLines := 80
	if v := r.URL.Query().Get("lines"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			maxLines = n
		}
	}
	if maxLines <= 0 {
		maxLines = 20
	}
	if maxLines > 500 {
		maxLines = 500
	}

	chat, ok := s.loadChat(chatID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{"error": "chat not found"})
		return
	}
	if chat.token != token {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{"error": "invalid token"})
		return
	}

	data, err := os.ReadFile(chat.path)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]any{"error": "failed to read chat file"})
		return
	}
	lines := splitLines(string(data))
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	json.NewEncoder(w).Encode(TransferChatTailResponse{
		ChatID:   chatID,
		Content:  joinLines(lines),
		FilePath: chat.path,
	})
}

func splitLines(s string) []string {
	if s == "" {
		return []string{}
	}
	out := make([]string, 0, 128)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start <= len(s) {
		out = append(out, s[start:])
	}
	return out
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	n := 0
	for i := range lines {
		n += len(lines[i]) + 1
	}
	b := make([]byte, 0, n)
	for i := range lines {
		b = append(b, lines[i]...)
		if i != len(lines)-1 {
			b = append(b, '\n')
		}
	}
	return string(b)
}

func statusString(ok bool) string {
	if ok {
		return "ok"
	}
	return "fail"
}
