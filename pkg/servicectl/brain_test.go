package servicectl

import (
	"path/filepath"
	"testing"

	"github.com/JustSebNL/Spiderweb/pkg/config"
)

func TestBrainControllerStart_NoOpWhenDisabled(t *testing.T) {
	t.Parallel()

	controller := NewBrainController(t.TempDir(), config.CheapCognitionConfig{
		Enabled: false,
		Runtime: "vllm",
	})
	if err := controller.Start(); err != nil {
		t.Fatalf("expected disabled controller start to no-op, got %v", err)
	}
}

func TestBrainControllerStop_NoOpWhenNotStarted(t *testing.T) {
	t.Parallel()

	controller := NewBrainController(t.TempDir(), config.CheapCognitionConfig{
		Enabled: true,
		Runtime: "vllm",
	})
	if err := controller.Stop(); err != nil {
		t.Fatalf("expected stop without started process to no-op, got %v", err)
	}
}

func TestBrainControllerResolveBrainDir_PrefersBRAINDir(t *testing.T) {
	repoRoot := t.TempDir()
	t.Setenv("BRAIN_DIR", "/tmp/brain-primary")

	controller := NewBrainController(repoRoot, config.CheapCognitionConfig{})
	if got := controller.resolveBrainDir(); got != "/tmp/brain-primary" {
		t.Fatalf("resolveBrainDir() = %q, want %q", got, "/tmp/brain-primary")
	}
}

func TestBrainControllerResolveBrainDir_DefaultsToRepoBrain(t *testing.T) {
	repoRoot := t.TempDir()
	t.Setenv("BRAIN_DIR", "")

	controller := NewBrainController(repoRoot, config.CheapCognitionConfig{})
	want := filepath.Join(repoRoot, "brain")
	if got := controller.resolveBrainDir(); got != want {
		t.Fatalf("resolveBrainDir() = %q, want %q", got, want)
	}
}

func TestBrainControllerResolvePIDFile_PrefersBRAINPID(t *testing.T) {
	repoRoot := t.TempDir()
	t.Setenv("BRAIN_VLLM_PID_FILE", "/tmp/brain-vllm.pid")
	t.Setenv("YOUTU_VLLM_PID_FILE", "/tmp/youtu-vllm.pid")

	controller := NewBrainController(repoRoot, config.CheapCognitionConfig{})
	if got := controller.resolvePIDFile(); got != "/tmp/brain-vllm.pid" {
		t.Fatalf("resolvePIDFile() = %q, want %q", got, "/tmp/brain-vllm.pid")
	}
}

func TestBrainControllerResolvePIDFile_FallsBackToYOUTUPID(t *testing.T) {
	repoRoot := t.TempDir()
	t.Setenv("BRAIN_VLLM_PID_FILE", "")
	t.Setenv("YOUTU_VLLM_PID_FILE", "/tmp/youtu-vllm.pid")

	controller := NewBrainController(repoRoot, config.CheapCognitionConfig{})
	if got := controller.resolvePIDFile(); got != "/tmp/youtu-vllm.pid" {
		t.Fatalf("resolvePIDFile() = %q, want %q", got, "/tmp/youtu-vllm.pid")
	}
}

func TestBrainControllerResolvePIDFile_DefaultsUnderBrainDir(t *testing.T) {
	repoRoot := t.TempDir()
	t.Setenv("BRAIN_DIR", filepath.Join(repoRoot, "brain-custom"))
	t.Setenv("BRAIN_VLLM_PID_FILE", "")
	t.Setenv("YOUTU_VLLM_PID_FILE", "")

	controller := NewBrainController(repoRoot, config.CheapCognitionConfig{})
	want := filepath.Join(repoRoot, "brain-custom", "brain-vllm.pid")
	if got := controller.resolvePIDFile(); got != want {
		t.Fatalf("resolvePIDFile() = %q, want %q", got, want)
	}
}
