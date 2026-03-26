// Spiderweb - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 Spiderweb contributors

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/JustSebNL/Spiderweb/cmd/spiderweb/internal"
	"github.com/JustSebNL/Spiderweb/cmd/spiderweb/internal/agent"
	"github.com/JustSebNL/Spiderweb/cmd/spiderweb/internal/auth"
	"github.com/JustSebNL/Spiderweb/cmd/spiderweb/internal/cron"
	"github.com/JustSebNL/Spiderweb/cmd/spiderweb/internal/gateway"
	"github.com/JustSebNL/Spiderweb/cmd/spiderweb/internal/migrate"
	"github.com/JustSebNL/Spiderweb/cmd/spiderweb/internal/openclaw"
	"github.com/JustSebNL/Spiderweb/cmd/spiderweb/internal/skills"
	"github.com/JustSebNL/Spiderweb/cmd/spiderweb/internal/status"
	"github.com/JustSebNL/Spiderweb/cmd/spiderweb/internal/transfer"
	"github.com/JustSebNL/Spiderweb/cmd/spiderweb/internal/version"
	"github.com/JustSebNL/Spiderweb/cmd/spiderweb/internal/wakeup"
)

func NewSpiderwebCommand() *cobra.Command {
	short := fmt.Sprintf("%s sweb - Intake Patch for OpenClaw v%s\n\n", internal.Logo, internal.GetVersion())

	cmd := &cobra.Command{
		Use:     "sweb",
		Aliases: []string{"spiderweb"},
		Short:   short,
		Example: "sweb status",
	}

	cmd.AddCommand(
		wakeup.NewWakeupCommand(),
		agent.NewAgentCommand(),
		auth.NewAuthCommand(),
		gateway.NewGatewayCommand(),
		status.NewStatusCommand(),
		cron.NewCronCommand(),
		migrate.NewMigrateCommand(),
		openclaw.NewOpenClawCommand(),
		skills.NewSkillsCommand(),
		transfer.NewTransferCommand(),
		version.NewVersionCommand(),
	)

	return cmd
}

func main() {
	cmd := NewSpiderwebCommand()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
