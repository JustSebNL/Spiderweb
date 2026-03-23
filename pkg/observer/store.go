package observer

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/JustSebNL/Spiderweb/pkg/maintenance"
)

var ErrWorkspaceNotSet = errors.New("workspace not set")

type FileStore struct {
	workspace string
}

type Overview struct {
	GeneratedAt           time.Time         `json:"generated_at"`
	Status                string            `json:"status"`
	Score                 int               `json:"score"`
	Summary               string            `json:"summary"`
	Deferred              bool              `json:"deferred"`
	ComparedTo            string            `json:"compared_to,omitempty"`
	Recommendations       []string          `json:"recommendations,omitempty"`
	LastRemediationAt     time.Time         `json:"last_remediation_at,omitempty"`
	LastRemediationAction string            `json:"last_remediation_action,omitempty"`
	Services              []ServiceStatus   `json:"services"`
	CheapCognition        any               `json:"cheap_cognition"`
	Baseline              *BenchmarkSummary `json:"baseline,omitempty"`
}

type BenchmarkSummary struct {
	Timestamp time.Time `json:"timestamp"`
	Score     int       `json:"score"`
	Summary   string    `json:"summary"`
}

type ServiceStatus struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	State   string `json:"state"`
	Running bool   `json:"running"`
	Owned   bool   `json:"owned"`
	Action  string `json:"action,omitempty"`
	Message string `json:"message,omitempty"`
	PIDFile string `json:"pid_file,omitempty"`
}

type BenchmarksResponse struct {
	GeneratedAt time.Time                   `json:"generated_at"`
	Current     *maintenance.HealthSnapshot `json:"current,omitempty"`
	Baseline    *maintenance.HealthSnapshot `json:"baseline,omitempty"`
}

func NewFileStore(workspace string) *FileStore {
	return &FileStore{workspace: strings.TrimSpace(workspace)}
}

func (s *FileStore) Overview() (*Overview, error) {
	current, err := s.ReadCurrent()
	if err != nil {
		return nil, err
	}
	baseline, _ := s.ReadBaseline()

	out := &Overview{
		GeneratedAt:           time.Now().UTC(),
		Status:                current.Summary,
		Score:                 current.Score,
		Summary:               current.Summary,
		Deferred:              current.Deferred,
		ComparedTo:            current.ComparedTo,
		Recommendations:       current.Recommendations,
		LastRemediationAt:     current.LastRemediationAt,
		LastRemediationAction: current.LastRemediationAction,
		Services:              summarizeServices(current),
		CheapCognition:        current.CheapCognition,
	}
	if baseline != nil {
		out.Baseline = &BenchmarkSummary{
			Timestamp: baseline.Timestamp,
			Score:     baseline.Score,
			Summary:   baseline.Summary,
		}
	}

	return out, nil
}

func (s *FileStore) Benchmarks() (*BenchmarksResponse, error) {
	current, err := s.ReadCurrent()
	if err != nil {
		return nil, err
	}
	baseline, _ := s.ReadBaseline()
	return &BenchmarksResponse{
		GeneratedAt: time.Now().UTC(),
		Current:     current,
		Baseline:    baseline,
	}, nil
}

func (s *FileStore) Services() ([]ServiceStatus, error) {
	current, err := s.ReadCurrent()
	if err != nil {
		return nil, err
	}
	return summarizeServices(current), nil
}

func (s *FileStore) ReadCurrent() (*maintenance.HealthSnapshot, error) {
	return s.readSnapshot(s.healthFilePath())
}

func (s *FileStore) ReadBaseline() (*maintenance.HealthSnapshot, error) {
	return s.readSnapshot(s.baselineFilePath())
}

func (s *FileStore) healthFilePath() string {
	return filepath.Join(s.workspace, "state", "runtime-health.json")
}

func (s *FileStore) baselineFilePath() string {
	return s.healthFilePath() + ".baseline"
}

func (s *FileStore) readSnapshot(path string) (*maintenance.HealthSnapshot, error) {
	if s.workspace == "" {
		return nil, ErrWorkspaceNotSet
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var snapshot maintenance.HealthSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func summarizeServices(snapshot *maintenance.HealthSnapshot) []ServiceStatus {
	if snapshot == nil || len(snapshot.Processes) == 0 {
		return nil
	}

	out := make([]ServiceStatus, 0, len(snapshot.Processes))
	for id, proc := range snapshot.Processes {
		state := "offline"
		if proc.Running {
			state = "online"
		}
		if proc.Action == "restart_requested" {
			state = "restarting"
		} else if proc.Action == "restart_failed" {
			state = "degraded"
		}
		out = append(out, ServiceStatus{
			ID:      id,
			Name:    proc.Name,
			State:   state,
			Running: proc.Running,
			Owned:   proc.Owned,
			Action:  proc.Action,
			Message: proc.Message,
			PIDFile: proc.PIDFile,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}
