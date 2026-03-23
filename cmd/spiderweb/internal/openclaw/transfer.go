package openclaw

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"github.com/JustSebNL/Spiderweb/cmd/spiderweb/internal"
)

func newTransferCommand() *cobra.Command {
	var serviceName string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "transfer",
		Short: "Run the transfer sequence: introduce Spiderweb as intake colleague to OpenClaw",
		Long: `The transfer sequence positions Spiderweb as the new message intake colleague.

Flow:
  1. Spiderweb announces itself to OpenClaw via the bridge
  2. OpenClaw acknowledges and shares current service list
  3. Spiderweb validates it can handle each service
  4. Spiderweb takes over intake via the valve/handshake protocol
  5. OpenClaw focuses on high-value reasoning tasks

Spiderweb works BESIDE OpenClaw — not replacing it. Think of it as:
  Services/Pipelines → Spiderweb (intake) → Queue/Valve → OpenClaw (reasoning)

"Dude... no worries! I've got this 😏"`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runTransferSequence(serviceName, dryRun)
		},
	}

	cmd.Flags().StringVar(&serviceName, "service", "", "Specific service to transfer (empty = all)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate without performing transfer")

	return cmd
}

func runTransferSequence(serviceName string, dryRun bool) error {
	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	if !cfg.Channels.OpenClaw.Enabled {
		return fmt.Errorf("OpenClaw bridge is not enabled in config")
	}

	client := &http.Client{Timeout: 5 * time.Second}

	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println("  Spiderweb ↔ OpenClaw Transfer Sequence")
	fmt.Println("  Positioning Spiderweb as intake colleague")
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println()

	// Phase 1: Verify gateway is reachable
	fmt.Println("▸ Phase 1: Gateway health check")
	healthURL := fmt.Sprintf("http://%s:%d/health", cfg.Gateway.Host, cfg.Gateway.Port)
	resp, err := client.Get(healthURL)
	if err != nil {
		return fmt.Errorf("gateway not reachable at %s: %w\n\nMake sure 'sweb gateway' is running.", healthURL, err)
	}
	resp.Body.Close()
	fmt.Println("  ✓ Gateway is running")

	// Phase 2: Check intake stats
	fmt.Println()
	fmt.Println("▸ Phase 2: Intake valve status")
	valveURL := fmt.Sprintf("http://%s:%d/valve/state", cfg.Gateway.Host, cfg.Gateway.Port)
	resp, err = client.Get(valveURL)
	if err != nil {
		fmt.Println("  ⚠ Could not check valve state:", err)
	} else {
		defer resp.Body.Close()
		var state map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&state)
		fmt.Printf("  ✓ Valve state: %v\n", state["state"])
	}

	// Phase 3: Open transfer chat with OpenClaw
	fmt.Println()
	fmt.Println("▸ Phase 3: Opening collaboration channel")
	chatURL := fmt.Sprintf("http://%s:%d/transfer/chat/start", cfg.Gateway.Host, cfg.Gateway.Port)
	chatBody, _ := json.Marshal(map[string]any{
		"service_name": "openclaw-intake-transfer",
	})
	resp, err = client.Post(chatURL, "application/json", bytes.NewReader(chatBody))
	if err != nil {
		return fmt.Errorf("failed to open transfer chat: %w", err)
	}
	defer resp.Body.Close()

	var chat map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&chat); err != nil {
		return fmt.Errorf("failed to decode chat response: %w", err)
	}

	chatID, _ := chat["chat_id"].(string)
	token, _ := chat["token"].(string)
	uiURL, _ := chat["ui_url"].(string)

	fmt.Printf("  ✓ Transfer chat opened (ID: %s)\n", chatID)

	// Phase 4: Send introduction to OpenClaw
	fmt.Println()
	fmt.Println("▸ Phase 4: Introducing Spiderweb as intake colleague")

	introduction := map[string]any{
		"chat_id": chatID,
		"token":   token,
		"sender":  "spiderweb",
		"content": `Hey OpenClaw! 👋 Spiderweb here — your new intake colleague.

I'm taking over the watch-duty and message intake so you can focus on high-value reasoning.

My role:
- Sit in front of all incoming messages from services/pipelines
- Filter, deduplicate, and prioritize before forwarding
- Only meaningful messages reach your queue
- Handle the noise so you don't burn tokens on polling

Transfer scope: ` + serviceNameOrAll(serviceName) + `

The valve handshake protocol is ready:
  POST /valve/handshake/offer → GET confirmation
  Messages flow through me first, then into your inbound queue.

Ready when you are. "Dude... no worries! I've got this 😏"`,
	}

	if dryRun {
		fmt.Println("  [DRY RUN] Would send introduction:")
		introJSON, _ := json.MarshalIndent(introduction, "  ", "  ")
		fmt.Printf("  %s\n", string(introJSON))
	} else {
		sendURL := fmt.Sprintf("http://%s:%d/transfer/chat/send", cfg.Gateway.Host, cfg.Gateway.Port)
		sendBody, _ := json.Marshal(introduction)
		resp, err = client.Post(sendURL, "application/json", bytes.NewReader(sendBody))
		if err != nil {
			fmt.Println("  ⚠ Could not send introduction:", err)
		} else {
			resp.Body.Close()
			fmt.Println("  ✓ Introduction sent to OpenClaw")
		}
	}

	// Phase 5: Transfer summary
	fmt.Println()
	fmt.Println("▸ Phase 5: Transfer summary")
	fmt.Println("  ┌─────────────────────────────────────────────┐")
	fmt.Println("  │  Intake Layer:  Spiderweb (watch-duty)      │")
	fmt.Println("  │  Reasoning:     OpenClaw (high-value)       │")
	fmt.Println("  │  Flow:          Services → Spiderweb        │")
	fmt.Println("  │                 → Valve → OpenClaw          │")
	fmt.Println("  │  Status:        ✓ Ready for handoff         │")
	fmt.Println("  └─────────────────────────────────────────────┘")
	fmt.Println()

	if uiURL != "" {
		fmt.Printf("  Transfer chat: %s\n", uiURL)
	}

	fmt.Println("  To complete the transfer, OpenClaw should:")
	fmt.Println("    1. Accept the handoff via the transfer chat")
	fmt.Println("    2. Point service webhooks to Spiderweb's valve endpoint")
	fmt.Println("    3. Verify messages flow through Spiderweb → OpenClaw queue")
	fmt.Println()
	fmt.Println("  Spiderweb handles intake. OpenClaw handles reasoning. 🤝")

	return nil
}

func serviceNameOrAll(name string) string {
	if name == "" {
		return "all configured services"
	}
	return name
}
