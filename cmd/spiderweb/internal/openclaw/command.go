package openclaw

import (
	"github.com/spf13/cobra"
)

func NewOpenClawCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "openclaw",
		Short: "Manage Spiderweb ↔ OpenClaw bridge and transfer",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(
		newStatusCommand(),
		newConnectCommand(),
		newTransferCommand(),
	)

	return cmd
}
