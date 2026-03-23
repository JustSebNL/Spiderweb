package transfer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"github.com/JustSebNL/Spiderweb/cmd/spiderweb/internal"
	"github.com/JustSebNL/Spiderweb/pkg/bus"
)

func newStatsCommand() *cobra.Command {
	var days int

	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show recent intake usage stats from the local gateway",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				return fmt.Errorf("error loading config: %w", err)
			}
			if days <= 0 {
				days = 7
			}
			if days > 30 {
				days = 30
			}

			url := fmt.Sprintf("http://%s:%d/intake/stats?days=%d", cfg.Gateway.Host, cfg.Gateway.Port, days)

			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Get(url)
			if err != nil {
				return fmt.Errorf("failed to fetch %s: %w", url, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return fmt.Errorf("gateway returned %s", resp.Status)
			}

			var snap bus.InboundUsageSnapshot
			if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}

			fmt.Printf("Window: %d day(s)\n", snap.WindowDays)
			fmt.Printf("Total: %d msgs (high=%d, low=%d)\n", snap.Total.Messages, snap.Total.High, snap.Total.Low)
			for _, d := range snap.Days {
				fmt.Printf("%s: %d msgs (high=%d, low=%d)\n", d.Date, d.Messages, d.High, d.Low)
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&days, "days", 7, "Days of history to show (1-30)")

	return cmd
}

