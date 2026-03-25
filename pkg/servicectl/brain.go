package servicectl

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/JustSebNL/Spiderweb/pkg/config"
)

type BrainController struct {
	repoRoot string
	cfg      config.CheapCognitionConfig
	started  bool
}

func NewBrainController(repoRoot string, cfg config.CheapCognitionConfig) *BrainController {
	return &BrainController{repoRoot: repoRoot, cfg: cfg}
}

func (c *BrainController) Start() error {
	if c == nil || !c.cfg.Enabled || strings.ToLower(strings.TrimSpace(c.cfg.Runtime)) != "vllm" {
		return nil
	}

	cmd := exec.Command("bash", filepath.Join(c.repoRoot, "scripts", "start_brain_vllm.sh"))
	cmd.Env = append(os.Environ(),
		"BRAIN_DIR="+c.resolveBrainDir(),
		"BRAIN_VLLM_PID_FILE="+c.resolvePIDFile(),
	)
	cmd.Dir = c.repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("start brain vllm: %w: %s", err, string(output))
	}
	if strings.Contains(string(output), "already running") {
		c.started = false
		return nil
	}
	c.started = true
	return nil
}

func (c *BrainController) Stop() error {
	if c == nil || strings.ToLower(strings.TrimSpace(c.cfg.Runtime)) != "vllm" {
		return nil
	}
	scriptPath := filepath.Join(c.repoRoot, "scripts", "stop_brain_vllm.sh")
	if _, err := os.Stat(scriptPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	cmd := exec.Command("bash", scriptPath)
	cmd.Env = append(os.Environ(),
		"BRAIN_DIR="+c.resolveBrainDir(),
		"BRAIN_VLLM_PID_FILE="+c.resolvePIDFile(),
	)
	cmd.Dir = c.repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("stop brain vllm: %w: %s", err, string(output))
	}
	c.started = false
	return nil
}

func (c *BrainController) Restart() error {
	if c == nil {
		return nil
	}
	if err := c.Stop(); err != nil {
		return err
	}
	return c.Start()
}

func (c *BrainController) resolveBrainDir() string {
	if c == nil {
		return filepath.Join(c.repoRoot, "brain")
	}
	if dir := os.Getenv("BRAIN_DIR"); dir != "" {
		return dir
	}
	if dir := os.Getenv("YOUTU_DIR"); dir != "" {
		return dir
	}
	return filepath.Join(c.repoRoot, "brain")
}

func (c *BrainController) resolvePIDFile() string {
	if c == nil {
		return filepath.Join(c.resolveBrainDir(), "brain-vllm.pid")
	}
	if path := os.Getenv("BRAIN_VLLM_PID_FILE"); path != "" {
		return path
	}
	if path := os.Getenv("YOUTU_VLLM_PID_FILE"); path != "" {
		return path
	}
	return filepath.Join(c.resolveBrainDir(), "brain-vllm.pid")
}
