package openclaw

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"github.com/JustSebNL/Spiderweb/cmd/spiderweb/internal"
)

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check OpenClaw bridge connection status",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				return fmt.Errorf("error loading config: %w", err)
			}

			if !cfg.Channels.OpenClaw.Enabled {
				fmt.Println("OpenClaw bridge: DISABLED")
				fmt.Println()
				fmt.Println("Enable it in config.json:")
				fmt.Println(`  "channels": { "openclaw": { "enabled": true } }`)
				return nil
			}

			client := &http.Client{Timeout: 3 * time.Second}
			url := fmt.Sprintf("http://%s:%d/health", cfg.Gateway.Host, cfg.Gateway.Port)
			resp, err := client.Get(url)
			if err != nil {
				return fmt.Errorf("gateway not reachable at %s: %w", url, err)
			}
			defer resp.Body.Close()

			var health map[string]any
			_ = json.NewDecoder(resp.Body).Decode(&health)

			fmt.Println("OpenClaw bridge: ENABLED")
			fmt.Printf("Gateway: %s:%d\n", cfg.Gateway.Host, cfg.Gateway.Port)
			fmt.Printf("WebSocket path: %s\n", cfg.Channels.OpenClaw.WebhookPath)
			fmt.Printf("Auto-handshake: %v\n", cfg.Channels.OpenClaw.AutoHandshake)
			fmt.Printf("Intake enabled: %v\n", cfg.Channels.OpenClaw.IntakeEnabled)
			fmt.Printf("Gateway status: %s\n", health["status"])

			return nil
		},
	}
}
