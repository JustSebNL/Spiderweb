package health

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/JustSebNL/Spiderweb/pkg/observer"
)

func (s *Server) observerStore() *observer.FileStore {
	s.mu.RLock()
	workspace := s.workspace
	s.mu.RUnlock()
	return observer.NewFileStore(workspace)
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

func (s *Server) writeObserverError(w http.ResponseWriter, err error) {
	status := http.StatusServiceUnavailable
	switch {
	case errors.Is(err, observer.ErrWorkspaceNotSet):
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
