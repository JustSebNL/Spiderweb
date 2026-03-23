package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/JustSebNL/Spiderweb/pkg/config"
)

const Logo = "🕸️"

var (
	version   = "dev"
	gitCommit string
	buildTime string
	goVersion string
)

func GetConfigPath() string {
	home, _ := os.UserHomeDir()
	newPath := filepath.Join(home, ".spiderweb", "config.json")
	oldPath := filepath.Join(home, ".spiderweb", "config.json")

	if _, err := os.Stat(newPath); err == nil {
		return newPath
	}
	if _, err := os.Stat(oldPath); err == nil {
		return oldPath
	}
	return newPath
}

func LoadConfig() (*config.Config, error) {
	return config.LoadConfig(GetConfigPath())
}

// FormatVersion returns the version string with optional git commit
func FormatVersion() string {
	v := version
	if gitCommit != "" {
		v += fmt.Sprintf(" (git: %s)", gitCommit)
	}
	return v
}

// FormatBuildInfo returns build time and go version info
func FormatBuildInfo() (build string, goVer string) {
	if buildTime != "" {
		build = buildTime
	}
	goVer = goVersion
	if goVer == "" {
		goVer = runtime.Version()
	}
	return
}

// GetVersion returns the version string
func GetVersion() string {
	return version
}
