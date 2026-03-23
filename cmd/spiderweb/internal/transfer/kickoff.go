package transfer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/JustSebNL/Spiderweb/cmd/spiderweb/internal"
	"github.com/JustSebNL/Spiderweb/pkg/bus"
)

func newKickoffCommand(transfersDirFn func() (string, error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kickoff <service-name>",
		Short: "Create doc + chat + show stats for a transfer",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			serviceName := strings.TrimSpace(args[0])
			if serviceName == "" {
				return fmt.Errorf("service-name is required")
			}

			cfg, err := internal.LoadConfig()
			if err != nil {
				return fmt.Errorf("error loading config: %w", err)
			}

			transfersDir, err := transfersDirFn()
			if err != nil {
				return err
			}

			if err := os.MkdirAll(transfersDir, 0o755); err != nil {
				return fmt.Errorf("failed to create transfers dir: %w", err)
			}

			docSlug := slugify(serviceName)
			docPath := filepath.Join(transfersDir, docSlug+".md")
			if _, err := os.Stat(docPath); os.IsNotExist(err) {
				tpl, err := loadTemplate()
				if err != nil {
					return fmt.Errorf("failed to load template: %w", err)
				}
				content := renderTemplate(tpl, serviceName)
				if err := os.WriteFile(docPath, []byte(content), 0o644); err != nil {
					return fmt.Errorf("failed to write transfer doc: %w", err)
				}
			}

			startURL := fmt.Sprintf("http://%s:%d/transfer/chat/start", cfg.Gateway.Host, cfg.Gateway.Port)
			startBody, _ := json.Marshal(map[string]any{"service_name": serviceName})
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Post(startURL, "application/json", bytes.NewReader(startBody))
			if err != nil {
				return fmt.Errorf("failed to call %s: %w", startURL, err)
			}
			defer resp.Body.Close()
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				b, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("gateway returned %s: %s", resp.Status, strings.TrimSpace(string(b)))
			}

			var chat map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&chat); err != nil {
				return fmt.Errorf("failed to decode chat response: %w", err)
			}

			chatID, _ := chat["chat_id"].(string)
			token, _ := chat["token"].(string)
			chatFile, _ := chat["file"].(string)
			uiURL, _ := chat["ui_url"].(string)

			statsURL := fmt.Sprintf("http://%s:%d/intake/stats?days=7", cfg.Gateway.Host, cfg.Gateway.Port)
			statsResp, err := client.Get(statsURL)
			if err == nil {
				defer statsResp.Body.Close()
				if statsResp.StatusCode >= 200 && statsResp.StatusCode < 300 {
					var snap bus.InboundUsageSnapshot
					_ = json.NewDecoder(statsResp.Body).Decode(&snap)
					fmt.Printf("Intake: %d msgs last %d day(s)\n", snap.Total.Messages, snap.WindowDays)
				}
			}

			fmt.Println()
			fmt.Printf("Transfer doc: %s\n", docPath)
			fmt.Printf("Transfer logs: %s\n", filepath.Join(cfg.WorkspacePath(), "transfer-logs"))
			if chatFile != "" {
				fmt.Printf("Chat file: %s\n", chatFile)
			}
			if uiURL != "" {
				fmt.Printf("Chat URL: %s\n", uiURL)
			}

			fmt.Println()
			fmt.Printf("Chat ID: %s\n", chatID)
			fmt.Printf("Token: %s\n", token)
			fmt.Println()
			fmt.Println("OpenClaw invite payload:")
			fmt.Printf("  POST http://%s:%d/transfer/chat/send\n", cfg.Gateway.Host, cfg.Gateway.Port)
			fmt.Printf("  {\"chat_id\":\"%s\",\"token\":\"%s\",\"sender\":\"OpenClaw\",\"content\":\"<message>\"}\n", chatID, token)
			fmt.Println()
			fmt.Println("Tail:")
			fmt.Printf("  sweb transfer chat tail %s --token %s\n", chatID, token)

			return nil
		},
	}
	return cmd
}
