package onboard

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/JustSebNL/Spiderweb/cmd/spiderweb/internal"
	"github.com/JustSebNL/Spiderweb/pkg/config"
)

func onboard() {
	configPath := internal.GetConfigPath()

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config already exists at %s\n", configPath)
		fmt.Print("Overwrite? (y/n): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" {
			fmt.Println("Aborted.")
			return
		}
	}

	cfg := config.DefaultConfig()
	registerAgentsFromProfiles(cfg)
	if err := config.SaveConfig(configPath, cfg); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		os.Exit(1)
	}

	workspace := cfg.WorkspacePath()
	ensureWorkspaceTemplates(workspace)

	fmt.Printf("%s spiderweb is ready!\n", internal.Logo)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Add your API key to", configPath)
	fmt.Println("")
	fmt.Println("     Recommended:")
	fmt.Println("     - OpenRouter: https://openrouter.ai/keys (access 100+ models)")
	fmt.Println("     - Ollama:     https://ollama.com (local, free)")
	fmt.Println("")
	fmt.Println("     See README.md for 17+ supported providers.")
	fmt.Println("")
	fmt.Println("  2. Chat: sweb agent -m \"Hello!\"")
}

func ensureWorkspaceTemplates(workspace string) {
	err := copyEmbeddedDir("workspace", workspace)
	if err != nil {
		fmt.Printf("Error copying workspace templates: %v\n", err)
	}

	agentsTarget := filepath.Join(workspace, "agents")
	if err := copyEmbeddedDir("agents", agentsTarget); err != nil {
		fmt.Printf("Error copying agent profiles: %v\n", err)
	}
}

func copyEmbeddedDir(sourceRoot, targetDir string) error {
	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("Failed to create target directory: %w", err)
	}

	// Walk through all files in embed.FS
	err := fs.WalkDir(embeddedFiles, sourceRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Read embedded file
		data, err := embeddedFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("Failed to read embedded file %s: %w", path, err)
		}

		new_path, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return fmt.Errorf("Failed to get relative path for %s: %v\n", path, err)
		}

		// Build target file path
		targetPath := filepath.Join(targetDir, new_path)

		// Ensure target file's directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("Failed to create directory %s: %w", filepath.Dir(targetPath), err)
		}

		if _, err := os.Stat(targetPath); err == nil {
			return nil
		}

		// Write file
		if err := os.WriteFile(targetPath, data, 0o644); err != nil {
			return fmt.Errorf("Failed to write file %s: %w", targetPath, err)
		}

		return nil
	})

	return err
}

func EnsureInitialized() (*config.Config, error) {
	configPath := internal.GetConfigPath()
	if _, err := os.Stat(configPath); err == nil {
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			return nil, err
		}
		if registerAgentsFromProfiles(cfg) {
			if err := config.SaveConfig(configPath, cfg); err != nil {
				return nil, err
			}
		}
		ensureWorkspaceTemplates(cfg.WorkspacePath())
		return cfg, nil
	}

	cfg := config.DefaultConfig()
	registerAgentsFromProfiles(cfg)
	if err := config.SaveConfig(configPath, cfg); err != nil {
		return nil, err
	}
	ensureWorkspaceTemplates(cfg.WorkspacePath())
	return cfg, nil
}

func registerAgentsFromProfiles(cfg *config.Config) bool {
	profiles := listEmbeddedAgentProfiles()
	if len(profiles) == 0 {
		return false
	}

	existing := make(map[string]struct{}, len(cfg.Agents.List))
	for _, agent := range cfg.Agents.List {
		if agent.ID != "" {
			existing[agent.ID] = struct{}{}
		}
	}

	changed := false
	for _, profile := range profiles {
		id := agentIDFromFilename(profile)
		if id == "" {
			continue
		}
		if _, ok := existing[id]; ok {
			continue
		}
		cfg.Agents.List = append(cfg.Agents.List, config.AgentConfig{
			ID:   id,
			Name: agentNameFromID(id),
		})
		existing[id] = struct{}{}
		changed = true
	}
	return changed
}

func listEmbeddedAgentProfiles() []string {
	var files []string
	_ = fs.WalkDir(embeddedFiles, "agents", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := strings.ToLower(d.Name())
		if strings.HasSuffix(name, ".md") {
			files = append(files, d.Name())
		}
		return nil
	})
	return files
}

func agentIDFromFilename(filename string) string {
	name := strings.TrimSuffix(strings.ToLower(filename), ".md")
	name = strings.TrimPrefix(name, "spiderweb-")
	name = strings.TrimSpace(name)
	return name
}

func agentNameFromID(id string) string {
	parts := strings.FieldsFunc(id, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	if len(parts) == 0 {
		return id
	}
	return strings.Join(parts, " ")
}
