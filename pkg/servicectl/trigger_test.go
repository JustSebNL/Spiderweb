package servicectl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JustSebNL/Spiderweb/pkg/config"
)

func TestTriggerControllerStart_NoWorkspace_NoOp(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	controller := NewTriggerController(repoRoot, config.TriggerConfig{
		Enabled:   true,
		AutoStart: true,
	})

	if err := controller.Start(); err != nil {
		t.Fatalf("expected missing optional trigger workspace to be a no-op, got %v", err)
	}
}

func TestTriggerControllerResolveWorkdir_RequiresPackageJSON(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	triggerDir := filepath.Join(repoRoot, "trigger")
	if err := os.MkdirAll(triggerDir, 0o755); err != nil {
		t.Fatalf("mkdir trigger dir: %v", err)
	}

	controller := NewTriggerController(repoRoot, config.TriggerConfig{
		Enabled:   true,
		AutoStart: true,
	})
	if workdir := controller.resolveWorkdir(); workdir != "" {
		t.Fatalf("expected empty workdir without package.json, got %q", workdir)
	}
}
