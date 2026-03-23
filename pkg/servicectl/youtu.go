package servicectl

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/JustSebNL/Spiderweb/pkg/config"
)

type YoutuController struct {
	repoRoot string
	cfg      config.CheapCognitionConfig
	started  bool
}

func NewYoutuController(repoRoot string, cfg config.CheapCognitionConfig) *YoutuController {
	return &YoutuController{repoRoot: repoRoot, cfg: cfg}
}

func (c *YoutuController) Start() error {
	if c == nil || !c.cfg.Enabled || strings.ToLower(strings.TrimSpace(c.cfg.Runtime)) != "vllm" {
		return nil
	}

	cmd := exec.Command("bash", filepath.Join(c.repoRoot, "scripts", "start_youtu_vllm.sh"))
	cmd.Env = append(os.Environ(),
		"YOUTU_DIR="+c.resolveYoutuDir(),
		"YOUTU_VLLM_PID_FILE="+c.resolvePIDFile(),
	)
	cmd.Dir = c.repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("start youtu vllm: %w: %s", err, string(output))
	}
	if strings.Contains(string(output), "already running") {
		c.started = false
		return nil
	}
	c.started = true
	return nil
}

func (c *YoutuController) Stop() error {
	if c == nil || !c.started || strings.ToLower(strings.TrimSpace(c.cfg.Runtime)) != "vllm" {
		return nil
	}

	cmd := exec.Command("bash", filepath.Join(c.repoRoot, "scripts", "stop_youtu_vllm.sh"))
	cmd.Env = append(os.Environ(),
		"YOUTU_DIR="+c.resolveYoutuDir(),
		"YOUTU_VLLM_PID_FILE="+c.resolvePIDFile(),
	)
	cmd.Dir = c.repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("stop youtu vllm: %w: %s", err, string(output))
	}
	c.started = false
	return nil
}

func (c *YoutuController) resolveYoutuDir() string {
	if c == nil {
		return filepath.Join(c.repoRoot, "youtu-llm")
	}
	if dir := os.Getenv("YOUTU_DIR"); dir != "" {
		return dir
	}
	return filepath.Join(c.repoRoot, "youtu-llm")
}

func (c *YoutuController) resolvePIDFile() string {
	if c == nil {
		return filepath.Join(c.resolveYoutuDir(), "youtu-vllm.pid")
	}
	if path := os.Getenv("YOUTU_VLLM_PID_FILE"); path != "" {
		return path
	}
	return filepath.Join(c.resolveYoutuDir(), "youtu-vllm.pid")
}
