package spiderweb

import (
	"github.com/spf13/cobra"
)

func NewSpiderwebCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sweb",
		Aliases: []string{"spiderweb"},
		Short:   "Spiderweb interface and launch sequence",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newWakeupCommand())
	return cmd
}
