package transfer

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newPathCommand(transfersDirFn func() (string, error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "path",
		Short: "Print transfers directory path",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			dir, err := transfersDirFn()
			if err != nil {
				return err
			}
			fmt.Println(dir)
			return nil
		},
	}
	return cmd
}

