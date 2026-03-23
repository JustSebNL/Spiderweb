package transfer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func newListCommand(transfersDirFn func() (string, error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List transfer documents",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			dir, err := transfersDirFn()
			if err != nil {
				return err
			}

			entries, err := os.ReadDir(dir)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("(no transfer docs)")
					return nil
				}
				return fmt.Errorf("failed to read transfers dir: %w", err)
			}

			var names []string
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				name := e.Name()
				if !strings.HasSuffix(strings.ToLower(name), ".md") {
					continue
				}
				names = append(names, name)
			}
			sort.Strings(names)
			if len(names) == 0 {
				fmt.Println("(no transfer docs)")
				return nil
			}

			for _, n := range names {
				fmt.Println(filepath.Join(dir, n))
			}
			return nil
		},
	}

	return cmd
}

