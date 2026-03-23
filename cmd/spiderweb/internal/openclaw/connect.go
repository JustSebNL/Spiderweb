package openclaw

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"

	"github.com/JustSebNL/Spiderweb/cmd/spiderweb/internal"
)

func newConnectCommand() *cobra.Command {
	var peerName string

	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect to Spiderweb as an OpenClaw peer (for testing)",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				return fmt.Errorf("error loading config: %w", err)
			}

			wsPath := cfg.Channels.OpenClaw.WebhookPath
			if wsPath == "" {
				wsPath = "/bridge/openclaw"
			}

			u := url.URL{
				Scheme: "ws",
				Host:   fmt.Sprintf("%s:%d", cfg.Gateway.Host, cfg.Gateway.Port),
				Path:   wsPath,
			}

			fmt.Printf("Connecting to %s as %s ...\n", u.String(), peerName)

			conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
			if err != nil {
				return fmt.Errorf("dial failed: %w", err)
			}
			defer conn.Close()

			// Send handshake
			secret := cfg.Channels.OpenClaw.SharedSecret
			handshake := map[string]any{
				"type":    "handshake",
				"sender":  peerName,
				"secret":  secret,
				"content": "requesting intake handoff",
			}
			handshakeJSON, _ := json.Marshal(handshake)
			if err := conn.WriteMessage(websocket.TextMessage, handshakeJSON); err != nil {
				return fmt.Errorf("handshake write failed: %w", err)
			}

			// Read handshake ack
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return fmt.Errorf("read ack failed: %w", err)
			}
			fmt.Printf("< %s\n", string(msg))

			// Ctrl+C to exit
			interrupt := make(chan os.Signal, 1)
			signal.Notify(interrupt, os.Interrupt)

			fmt.Println("Connected. Type messages and press Enter. Ctrl+C to disconnect.")

			// Read goroutine
			go func() {
				for {
					_, message, err := conn.ReadMessage()
					if err != nil {
						log.Printf("Read error: %v", err)
						interrupt <- os.Interrupt
						return
					}
					fmt.Printf("< %s\n", string(message))
				}
			}()

			// Write loop
			go func() {
				for {
					var input string
					if _, err := fmt.Scanln(&input); err != nil {
						continue
					}
					if input == "" {
						continue
					}
					env := map[string]any{
						"type":    "message",
						"sender":  peerName,
						"content": input,
					}
					data, _ := json.Marshal(env)
					if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
						log.Printf("Write error: %v", err)
						return
					}
				}
			}()

			<-interrupt
			fmt.Println("\nDisconnecting...")
			_ = conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
				time.Now().Add(5*time.Second),
			)
			return nil
		},
	}

	cmd.Flags().StringVar(&peerName, "name", "openclaw", "Peer name for the handshake")

	return cmd
}
