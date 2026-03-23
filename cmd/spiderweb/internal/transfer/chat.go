package transfer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/JustSebNL/Spiderweb/cmd/spiderweb/internal"
)

func newChatCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Create and use a transfer discussion chat area",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(
		newChatStartCommand(),
		newChatSendCommand(),
		newChatTailCommand(),
	)

	return cmd
}

func newChatStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start <service-name>",
		Short: "Create a new transfer chat and print an OpenClaw invite payload",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				return fmt.Errorf("error loading config: %w", err)
			}

			serviceName := strings.TrimSpace(args[0])
			if serviceName == "" {
				return fmt.Errorf("service-name is required")
			}

			url := fmt.Sprintf("http://%s:%d/transfer/chat/start", cfg.Gateway.Host, cfg.Gateway.Port)
			body, _ := json.Marshal(map[string]any{"service_name": serviceName})

			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Post(url, "application/json", bytes.NewReader(body))
			if err != nil {
				return fmt.Errorf("failed to call %s: %w", url, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				b, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("gateway returned %s: %s", resp.Status, strings.TrimSpace(string(b)))
			}

			var out map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}

			chatID, _ := out["chat_id"].(string)
			token, _ := out["token"].(string)
			file, _ := out["file"].(string)
			uiURL, _ := out["ui_url"].(string)

			fmt.Printf("Chat ID: %s\n", chatID)
			fmt.Printf("Token: %s\n", token)
			if file != "" {
				fmt.Printf("File: %s\n", file)
			}
			if uiURL != "" {
				fmt.Printf("URL: %s\n", uiURL)
			}
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

func newChatSendCommand() *cobra.Command {
	var token string
	var sender string
	var message string

	cmd := &cobra.Command{
		Use:   "send <chat-id>",
		Short: "Send a message into a transfer chat",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				return fmt.Errorf("error loading config: %w", err)
			}
			chatID := strings.TrimSpace(args[0])
			if chatID == "" {
				return fmt.Errorf("chat-id is required")
			}
			if token == "" {
				return fmt.Errorf("--token is required")
			}
			if sender == "" {
				sender = "spiderweb"
			}
			if strings.TrimSpace(message) == "" {
				return fmt.Errorf("--message is required")
			}

			url := fmt.Sprintf("http://%s:%d/transfer/chat/send", cfg.Gateway.Host, cfg.Gateway.Port)
			body, _ := json.Marshal(map[string]any{
				"chat_id":  chatID,
				"token":    token,
				"sender":   sender,
				"content":  message,
			})

			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Post(url, "application/json", bytes.NewReader(body))
			if err != nil {
				return fmt.Errorf("failed to call %s: %w", url, err)
			}
			defer resp.Body.Close()
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				b, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("gateway returned %s: %s", resp.Status, strings.TrimSpace(string(b)))
			}
			fmt.Println("✓ sent")
			return nil
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "Chat token")
	cmd.Flags().StringVar(&sender, "sender", "spiderweb", "Sender label")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Message text")

	return cmd
}

func newChatTailCommand() *cobra.Command {
	var token string
	var lines int

	cmd := &cobra.Command{
		Use:   "tail <chat-id>",
		Short: "Print the last lines of a transfer chat",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				return fmt.Errorf("error loading config: %w", err)
			}
			chatID := strings.TrimSpace(args[0])
			if chatID == "" {
				return fmt.Errorf("chat-id is required")
			}
			if token == "" {
				return fmt.Errorf("--token is required")
			}
			if lines <= 0 {
				lines = 80
			}

			url := fmt.Sprintf(
				"http://%s:%d/transfer/chat/tail?chat_id=%s&token=%s&lines=%d",
				cfg.Gateway.Host,
				cfg.Gateway.Port,
				chatID,
				token,
				lines,
			)

			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Get(url)
			if err != nil {
				return fmt.Errorf("failed to call %s: %w", url, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				b, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("gateway returned %s: %s", resp.Status, strings.TrimSpace(string(b)))
			}

			var out map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}
			content, _ := out["content"].(string)
			if content != "" {
				fmt.Println(content)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "Chat token")
	cmd.Flags().IntVar(&lines, "lines", 80, "Number of lines to show (max 500)")

	return cmd
}
