package spiderweb

import (
	"bufio"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/JustSebNL/Spiderweb/cmd/spiderweb/internal"
)

//go:embed LAUNCH_SEQUENCE.md
var embeddedFiles embed.FS

type launchState struct {
	Version     int    `json:"version"`
	RunID       string `json:"run_id"`
	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at,omitempty"`
	Completed   bool   `json:"completed"`
}

func newWakeupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "wakeup",
		Aliases: []string{"wake"},
		Short: "Run the Spiderweb launch sequence (point of no return)",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				return fmt.Errorf("error loading config: %w", err)
			}
			workspace := cfg.WorkspacePath()
			statePath := filepath.Join(workspace, "state", "spiderweb_launch.json")
			if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
				return fmt.Errorf("failed to create state dir: %w", err)
			}

			prev, _ := readState(statePath)
			if prev != nil && !prev.Completed {
				fmt.Println("WARNING: Previous launch sequence was not completed.")
				fmt.Println("It will restart from Chapter 1 on this launch.")
				fmt.Println("")
			}

			fmt.Println("Spiderweb Launch Sequence — Point of No Return")
			fmt.Println("")
			fmt.Println("###############################")
			fmt.Println("|     !!!  Spiderweb Wakeup Warning   !!!    |")
			fmt.Println("|                         Caution !!                          |")
			fmt.Println("|          Please read the following text         |")
			fmt.Println("|               with the utter most care             |")
			fmt.Println("|                 before you continue                 |")
			fmt.Println("################################")
			fmt.Println("Starting this process will spin up a transfer process that cannot be stopped.")
			fmt.Println("Even killing the Spiderweb app will just cause the Spiderweb configuration process to resume on the next restart.")
			fmt.Println("If you're ready to let the Spiderweb configuration process to run its course, please type in \"Wakeup Spiderweb\" and it will begin.")
			fmt.Println("Pressing \"Enter\" without any text will cancel the setup.")
			fmt.Println("")
			fmt.Print("> ")
			reader := bufio.NewReader(os.Stdin)
			line, _ := reader.ReadString('\n')
			line = strings.TrimSpace(line)
			if line == "" {
				fmt.Println("Aborted.")
				return nil
			}
			if line != "Wakeup Spiderweb" {
				fmt.Println("Aborted.")
				return nil
			}

			runID := randomHex(8)
			now := time.Now()
			st := &launchState{
				Version:   1,
				RunID:     runID,
				StartedAt: now.Format(time.RFC3339),
				Completed: false,
			}
			if err := writeState(statePath, st); err != nil {
				return err
			}

			if err := os.MkdirAll(filepath.Join(workspace, "transfer-logs"), 0o755); err != nil {
				return fmt.Errorf("failed to create transfer-logs: %w", err)
			}
			if err := os.MkdirAll(filepath.Join(workspace, "transfers"), 0o755); err != nil {
				return fmt.Errorf("failed to create transfers: %w", err)
			}

			b, err := embeddedFiles.ReadFile("LAUNCH_SEQUENCE.md")
			if err != nil {
				return fmt.Errorf("failed to load launch sequence: %w", err)
			}

			fmt.Println("")
			fmt.Println(string(b))

			st.Completed = true
			st.CompletedAt = time.Now().Format(time.RFC3339)
			if err := writeState(statePath, st); err != nil {
				return err
			}

			fmt.Println("")
			fmt.Printf("Launch sequence completed (run_id=%s)\n", runID)
			fmt.Printf("State: %s\n", statePath)
			fmt.Printf("Logs:  %s\n", filepath.Join(workspace, "transfer-logs"))
			return nil
		},
	}

	return cmd
}

func randomHex(nBytes int) string {
	if nBytes <= 0 {
		nBytes = 8
	}
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func readState(path string) (*launchState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var st launchState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

func writeState(path string, st *launchState) error {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize state: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}
	return nil
}
