package servicectl

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/JustSebNL/Spiderweb/pkg/config"
)

type TriggerController struct {
	repoRoot string
	cfg      config.TriggerConfig
	started  bool
}

func NewTriggerController(repoRoot string, cfg config.TriggerConfig) *TriggerController {
	return &TriggerController{repoRoot: repoRoot, cfg: cfg}
}

func (c *TriggerController) Start() error {
	if c == nil || !c.cfg.Enabled || !c.cfg.AutoStart {
		return nil
	}

	workdir := c.resolveWorkdir()
	if workdir == "" {
		return nil
	}
	pidFile := c.resolvePIDFile(workdir)
	logFile := c.resolveLogFile(workdir)

	cmd := exec.Command("bash", filepath.Join(c.repoRoot, "scripts", "start_trigger_worker.sh"))
	cmd.Env = append(os.Environ(),
		"TRIGGER_DIR="+workdir,
		"TRIGGER_PID_FILE="+pidFile,
		"TRIGGER_LOG_FILE="+logFile,
		fmt.Sprintf("TRIGGER_HOST=%s", c.cfg.Host),
		fmt.Sprintf("TRIGGER_PORT=%d", c.cfg.Port),
	)
	cmd.Dir = c.repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("start trigger worker: %w: %s", err, string(output))
	}
	if strings.Contains(string(output), "already running") {
		c.started = false
		return nil
	}
	c.started = true
	return nil
}

func (c *TriggerController) Stop() error {
	if c == nil || !c.cfg.Enabled {
		return nil
	}

	cmd := exec.Command("bash", filepath.Join(c.repoRoot, "scripts", "stop_trigger_worker.sh"))
	workdir := c.resolveWorkdir()
	cmd.Env = append(os.Environ(),
		"TRIGGER_DIR="+workdir,
		"TRIGGER_PID_FILE="+c.resolvePIDFile(workdir),
	)
	cmd.Dir = c.repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("stop trigger worker: %w: %s", err, string(output))
	}
	c.started = false
	return nil
}

func (c *TriggerController) Restart() error {
	if c == nil {
		return nil
	}
	if err := c.Stop(); err != nil {
		return err
	}
	return c.Start()
}

func (c *TriggerController) resolveWorkdir() string {
	if c == nil {
		return ""
	}
	if workdir := expandHome(c.cfg.Workdir); workdir != "" {
		if stat, err := os.Stat(workdir); err == nil && stat.IsDir() {
			return workdir
		}
		return ""
	}
	fallback := filepath.Join(c.repoRoot, "trigger")
	if stat, err := os.Stat(fallback); err == nil && stat.IsDir() {
		if _, err := os.Stat(filepath.Join(fallback, "package.json")); err == nil {
			return fallback
		}
	}
	return ""
}

func (c *TriggerController) resolvePIDFile(workdir string) string {
	if c == nil {
		return filepath.Join(workdir, ".trigger.pid")
	}
	if path := expandHome(c.cfg.PIDFile); path != "" {
		return path
	}
	return filepath.Join(workdir, ".trigger.pid")
}

func (c *TriggerController) resolveLogFile(workdir string) string {
	if c == nil {
		return filepath.Join(workdir, ".trigger.log")
	}
	if path := expandHome(c.cfg.LogFile); path != "" {
		return path
	}
	return filepath.Join(workdir, ".trigger.log")
}

func expandHome(path string) string {
	if path == "" {
		return path
	}
	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		if len(path) > 1 && path[1] == '/' {
			return home + path[1:]
		}
		return home
	}
	return path
}
