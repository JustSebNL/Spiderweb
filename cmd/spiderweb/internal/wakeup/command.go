package wakeup

import (
	"embed"

	"github.com/spf13/cobra"
)

//go:generate cp -r ../../../../workspace .
//go:generate cp -r ../../../../agents .
//go:embed workspace agents
var embeddedFiles embed.FS

func NewWakeupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "wakeup",
		Aliases: []string{"wake", "o"},
		Short:   "Initialize Spiderweb configuration and workspace",
		Run: func(cmd *cobra.Command, args []string) {
			runWakeup()
		},
	}

	return cmd
}
