package transfer

import (
	"embed"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/JustSebNL/Spiderweb/cmd/spiderweb/internal"
)

//go:embed TRANSFER_SHEET_TEMPLATE.md
var embeddedFiles embed.FS

type deps struct {
	transfersDir string
}

func NewTransferCommand() *cobra.Command {
	var d deps

	cmd := &cobra.Command{
		Use:   "transfer",
		Short: "Manage service transfer documents",
		Args:  cobra.NoArgs,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				return fmt.Errorf("error loading config: %w", err)
			}
			d.transfersDir = filepath.Join(cfg.WorkspacePath(), "transfers")
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	transfersDirFn := func() (string, error) {
		if d.transfersDir == "" {
			return "", fmt.Errorf("transfers directory is not initialized")
		}
		return d.transfersDir, nil
	}

	cmd.AddCommand(
		newPathCommand(transfersDirFn),
		newListCommand(transfersDirFn),
		newInitCommand(transfersDirFn),
		newKickoffCommand(transfersDirFn),
		newStatsCommand(),
		newChatCommand(),
	)

	return cmd
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "unknown"
	}
	s = strings.ReplaceAll(s, " ", "-")
	re := regexp.MustCompile(`[^a-z0-9._-]+`)
	s = re.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-._")
	if s == "" {
		return "unknown"
	}
	return s
}

func loadTemplate() (string, error) {
	b, err := embeddedFiles.ReadFile("TRANSFER_SHEET_TEMPLATE.md")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func renderTemplate(template string, serviceName string) string {
	out := template
	out = strings.ReplaceAll(out, "<name>", serviceName)
	out = strings.ReplaceAll(out, "<YYYY-MM-DD>", time.Now().Format("2006-01-02"))
	return out
}
