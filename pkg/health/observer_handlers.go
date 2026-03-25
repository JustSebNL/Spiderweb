package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/JustSebNL/Spiderweb/pkg/observer"
)

func (s *Server) observerStore() *observer.Store {
	s.mu.RLock()
	workspace := s.workspace
	healthFile := s.healthFile
	s.mu.RUnlock()
	return observer.NewStore(workspace, healthFile)
}

func (s *Server) observerOverviewHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	resp, err := s.observerStore().Overview()
	if err != nil {
		s.writeObserverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) observerBenchmarksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	resp, err := s.observerStore().Benchmarks()
	if err != nil {
		s.writeObserverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) observerServicesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	resp, err := s.observerStore().Services()
	if err != nil {
		s.writeObserverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"generated_at": time.Now().UTC(),
		"services":     resp,
	})
}

func (s *Server) observerAgentsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	resp, err := s.observerStore().AgentsFiltered(
		r.URL.Query().Get("state"),
		r.URL.Query().Get("pipeline"),
		r.URL.Query().Get("manager"),
	)
	if err != nil {
		s.writeObserverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"generated_at": time.Now().UTC(),
		"agents":       resp,
	})
}

func (s *Server) observerAgentsSummaryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	resp, err := s.observerStore().AgentSummary()
	if err != nil {
		s.writeObserverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) observerStats24hHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	resp, err := s.observerStore().Stats24h()
	if err != nil {
		s.writeObserverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) observerEventsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	resp, err := s.observerStore().Events(observerLimit(r, 24))
	if err != nil {
		s.writeObserverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) observerSelfCareCyclesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	resp, err := s.observerStore().SelfCareCycles(observerLimit(r, 12))
	if err != nil {
		s.writeObserverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) observerDashboardHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	resp, err := s.observerStore().Dashboard(observerLimit(r, 12))
	if err != nil {
		s.writeObserverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// requireLocalhost returns true if the request is from localhost (IPv4 or IPv6)
func (s *Server) requireLocalhost(r *http.Request) bool {
	host := r.RemoteAddr
	if host == "" {
		return false
	}
	// Strip port if present
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return host == "127.0.0.1" || host == "::1" || host == "localhost"
}

// requireObserverAuth checks if the request is authorized for observer actions
// Returns true if authorized (from localhost or if auth is disabled)
func (s *Server) requireObserverAuth(r *http.Request) bool {
	// For now, only allow localhost
	// Future: add API key or token-based auth for non-localhost access
	return s.requireLocalhost(r)
}

func (s *Server) observerRestartHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if !s.requireObserverAuth(r) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "observer actions require localhost access"})
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Service string `json:"service"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "invalid json body"})
		return
	}
	if req.Service == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "service is required"})
		return
	}

	s.mu.RLock()
	restartFn := s.restartFn
	s.mu.RUnlock()
	if restartFn == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "restart actions are not configured"})
		return
	}

	resp, err := restartFn(context.Background(), req.Service)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"service": req.Service,
			"error":   err.Error(),
		})
		return
	}
	if resp == nil {
		resp = map[string]any{}
	}
	resp["service"] = req.Service
	resp["requested_at"] = time.Now().UTC()
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) observerGenerateReportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if !s.requireObserverAuth(r) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "observer actions require localhost access"})
		return
	}

	w.Header().Set("Content-Type", "application/json")

	resp, err := s.observerStore().GenerateHTMLReport(observerLimit(r, 24))
	if err != nil {
		s.writeObserverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) observerSelfCareRunHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if !s.requireObserverAuth(r) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "observer actions require localhost access"})
		return
	}

	w.Header().Set("Content-Type", "application/json")

	s.mu.RLock()
	selfCareFn := s.selfCareFn
	s.mu.RUnlock()
	if selfCareFn == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "self-care action is not configured"})
		return
	}

	resp, err := selfCareFn(context.Background())
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
		return
	}
	if resp == nil {
		resp = map[string]any{}
	}
	resp["requested_at"] = time.Now().UTC()
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) observerLatestReportHandler(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "html" {
		data, info, err := s.observerStore().LatestReportHTML()
		if err != nil {
			s.writeObserverError(w, err)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Disposition", "inline; filename=\""+info.Filename+"\"")
		_, _ = w.Write(data)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	resp, err := s.observerStore().LatestReport()
	if err != nil {
		s.writeObserverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) observerJournalGenerateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	entry, err := s.observerStore().GenerateDailyJournal(time.Now().UTC())
	if err != nil {
		s.writeObserverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"generated_at": time.Now().UTC(),
		"entry":        entry,
	})
}

func (s *Server) observerJournalLatestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	resp, err := s.observerStore().LatestJournal()
	if err != nil {
		s.writeObserverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) observerClearEventsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if !s.requireObserverAuth(r) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "observer actions require localhost access"})
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req struct {
		RetentionDays int `json:"retention_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Use default if body is empty or invalid
		req.RetentionDays = 30
	}

	resp, err := s.observerStore().ClearOldEvents(req.RetentionDays)
	if err != nil {
		s.writeObserverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) observerResetBaselineHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if !s.requireObserverAuth(r) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "observer actions require localhost access"})
		return
	}

	w.Header().Set("Content-Type", "application/json")

	resp, err := s.observerStore().ResetBaseline()
	if err != nil {
		s.writeObserverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) observerConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	store := s.observerStore()

	if r.Method == http.MethodGet {
		config, err := store.LoadConfig()
		if err != nil {
			s.writeObserverError(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(config)
		return
	}

	if r.Method == http.MethodPost {
		if !s.requireObserverAuth(r) {
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "observer actions require localhost access"})
			return
		}

		var config observer.ObserverConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "invalid json body"})
			return
		}

		if err := store.SaveConfig(config); err != nil {
			s.writeObserverError(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(config)
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (s *Server) observerJournalConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	store := s.observerStore()

	if r.Method == http.MethodGet {
		journalConfig := store.GetJournalConfig()
		_ = json.NewEncoder(w).Encode(journalConfig)
		return
	}

	if r.Method == http.MethodPost {
		if !s.requireObserverAuth(r) {
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "observer actions require localhost access"})
			return
		}

		var journalConfig observer.JournalConfig
		if err := json.NewDecoder(r.Body).Decode(&journalConfig); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "invalid json body"})
			return
		}

		config, err := store.LoadConfig()
		if err != nil {
			config = observer.DefaultObserverConfig()
		}
		config.Journal = journalConfig

		if err := store.SaveConfig(config); err != nil {
			s.writeObserverError(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(journalConfig)
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

func observerLimit(r *http.Request, fallback int) int {
	value := r.URL.Query().Get("limit")
	if value == "" {
		return fallback
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit <= 0 {
		return fallback
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func (s *Server) writeObserverError(w http.ResponseWriter, err error) {
	status := http.StatusServiceUnavailable
	switch {
	case errors.Is(err, observer.ErrWorkspaceNotSet), errors.Is(err, observer.ErrHealthFileNotSet):
		status = http.StatusServiceUnavailable
	case errors.Is(err, os.ErrNotExist):
		status = http.StatusNotFound
	default:
		status = http.StatusInternalServerError
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": err.Error(),
	})
}
