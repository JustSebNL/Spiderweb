package transfer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newInitCommand(transfersDirFn func() (string, error)) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init <service-name>",
		Short: "Create a new transfer document from template",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			dir, err := transfersDirFn()
			if err != nil {
				return err
			}
			serviceName := strings.TrimSpace(args[0])
			if serviceName == "" {
				return fmt.Errorf("service-name is required")
			}

			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("failed to create transfers dir: %w", err)
			}

			slug := slugify(serviceName)
			path := filepath.Join(dir, slug+".md")
			if !force {
				if _, err := os.Stat(path); err == nil {
					return fmt.Errorf("transfer doc already exists: %s", path)
				}
			}

			tpl, err := loadTemplate()
			if err != nil {
				return fmt.Errorf("failed to load template: %w", err)
			}

			content := renderTemplate(tpl, serviceName)
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				return fmt.Errorf("failed to write transfer doc: %w", err)
			}

			fmt.Printf("✓ Created transfer doc: %s\n", path)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite if file exists")

	return cmd
}

