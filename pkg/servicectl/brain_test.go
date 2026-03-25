package servicectl

import (
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
