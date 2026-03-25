package gateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/JustSebNL/Spiderweb/cmd/spiderweb/internal"
	"github.com/JustSebNL/Spiderweb/pkg/agent"
	"github.com/JustSebNL/Spiderweb/pkg/bus"
	"github.com/JustSebNL/Spiderweb/pkg/channels"
	"github.com/JustSebNL/Spiderweb/pkg/config"
	"github.com/JustSebNL/Spiderweb/pkg/cron"
	"github.com/JustSebNL/Spiderweb/pkg/devices"
	"github.com/JustSebNL/Spiderweb/pkg/health"
	"github.com/JustSebNL/Spiderweb/pkg/heartbeat"
	"github.com/JustSebNL/Spiderweb/pkg/logger"
	"github.com/JustSebNL/Spiderweb/pkg/maintenance"
	"github.com/JustSebNL/Spiderweb/pkg/providers"
	"github.com/JustSebNL/Spiderweb/pkg/servicectl"
	"github.com/JustSebNL/Spiderweb/pkg/state"
	"github.com/JustSebNL/Spiderweb/pkg/tools"
	"github.com/JustSebNL/Spiderweb/pkg/voice"
)

func gatewayCmd(debug bool) error {
	if debug {
		logger.SetLevel(logger.DEBUG)
		fmt.Println("🔍 Debug mode enabled")
	}

	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	provider, modelID, err := providers.CreateProvider(cfg)
	if err != nil {
		return fmt.Errorf("error creating provider: %w", err)
	}

	repoRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error determining repo root: %w", err)
	}
	triggerCtl := servicectl.NewTriggerController(repoRoot, cfg.Trigger)
	if err := triggerCtl.Start(); err != nil {
		logger.WarnCF("gateway", "Trigger worker start failed; continuing without Trigger", map[string]any{
			"error": err.Error(),
		})
	}
	brainCtl := servicectl.NewBrainController(repoRoot, cfg.Intake.CheapCognition)
	if err := brainCtl.Start(); err != nil {
		return fmt.Errorf("error starting cheap cognition runtime: %w", err)
	}

	// Use the resolved model ID from provider creation
	if modelID != "" {
		cfg.Agents.Defaults.ModelName = modelID
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)

	// Print agent startup info
	fmt.Println("\n📦 Agent Status:")
	startupInfo := agentLoop.GetStartupInfo()
	toolsInfo := startupInfo["tools"].(map[string]any)
	skillsInfo := startupInfo["skills"].(map[string]any)
	fmt.Printf("  • Tools: %d loaded\n", toolsInfo["count"])
	fmt.Printf("  • Skills: %d/%d available\n",
		skillsInfo["available"],
		skillsInfo["total"])

	// Log to file as well
	logger.InfoCF("agent", "Agent initialized",
		map[string]any{
			"tools_count":      toolsInfo["count"],
			"skills_total":     skillsInfo["total"],
			"skills_available": skillsInfo["available"],
		})

	// Setup cron tool and service
	execTimeout := time.Duration(cfg.Tools.Cron.ExecTimeoutMinutes) * time.Minute
	cronService := setupCronTool(
		agentLoop,
		msgBus,
		cfg.WorkspacePath(),
		cfg.Agents.Defaults.RestrictToWorkspace,
		execTimeout,
		cfg,
	)

	heartbeatService := heartbeat.NewHeartbeatService(
		cfg.WorkspacePath(),
		cfg.Heartbeat.Interval,
		cfg.Heartbeat.Enabled,
	)
	heartbeatService.SetBus(msgBus)
	heartbeatService.SetHandler(func(prompt, channel, chatID string) *tools.ToolResult {
		// Use cli:direct as fallback if no valid channel
		if channel == "" || chatID == "" {
			channel, chatID = "cli", "direct"
		}
		// Use ProcessHeartbeat - no session history, each heartbeat is independent
		var response string
		response, err = agentLoop.ProcessHeartbeat(context.Background(), prompt, channel, chatID)
		if err != nil {
			return tools.ErrorResult(fmt.Sprintf("Heartbeat error: %v", err))
		}
		if response == "HEARTBEAT_OK" {
			return tools.SilentResult("Heartbeat OK")
		}
		// For heartbeat, always return silent - the subagent result will be
		// sent to user via processSystemMessage when the async task completes
		return tools.SilentResult(response)
	})

	channelManager, err := channels.NewManager(cfg, msgBus)
	if err != nil {
		return fmt.Errorf("error creating channel manager: %w", err)
	}

	// Inject channel manager into agent loop for command handling
	agentLoop.SetChannelManager(channelManager)

	var transcriber *voice.GroqTranscriber
	groqAPIKey := cfg.Providers.Groq.APIKey
	if groqAPIKey == "" {
		for _, mc := range cfg.ModelList {
			if strings.HasPrefix(mc.Model, "groq/") && mc.APIKey != "" {
				groqAPIKey = mc.APIKey
				break
			}
		}
	}
	if groqAPIKey != "" {
		transcriber = voice.NewGroqTranscriber(groqAPIKey)
		logger.InfoC("voice", "Groq voice transcription enabled")
	}

	if transcriber != nil {
		if telegramChannel, ok := channelManager.GetChannel("telegram"); ok {
			if tc, ok := telegramChannel.(*channels.TelegramChannel); ok {
				tc.SetTranscriber(transcriber)
				logger.InfoC("voice", "Groq transcription attached to Telegram channel")
			}
		}
		if discordChannel, ok := channelManager.GetChannel("discord"); ok {
			if dc, ok := discordChannel.(*channels.DiscordChannel); ok {
				dc.SetTranscriber(transcriber)
				logger.InfoC("voice", "Groq transcription attached to Discord channel")
			}
		}
		if slackChannel, ok := channelManager.GetChannel("slack"); ok {
			if sc, ok := slackChannel.(*channels.SlackChannel); ok {
				sc.SetTranscriber(transcriber)
				logger.InfoC("voice", "Groq transcription attached to Slack channel")
			}
		}
	}

	enabledChannels := channelManager.GetEnabledChannels()
	if len(enabledChannels) > 0 {
		fmt.Printf("✓ Channels enabled: %s\n", enabledChannels)
	} else {
		fmt.Println("⚠ Warning: No channels enabled")
	}

	fmt.Printf("✓ Gateway started on %s:%d\n", cfg.Gateway.Host, cfg.Gateway.Port)
	fmt.Println("Press Ctrl+C to stop")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cronService.Start(); err != nil {
		fmt.Printf("Error starting cron service: %v\n", err)
	}
	fmt.Println("✓ Cron service started")

	if err := heartbeatService.Start(); err != nil {
		fmt.Printf("Error starting heartbeat service: %v\n", err)
	}
	fmt.Println("✓ Heartbeat service started")

	stateManager := state.NewManager(cfg.WorkspacePath())
	deviceService := devices.NewService(devices.Config{
		Enabled:    cfg.Devices.Enabled,
		MonitorUSB: cfg.Devices.MonitorUSB,
	}, stateManager)
	deviceService.SetBus(msgBus)
	if err := deviceService.Start(ctx); err != nil {
		fmt.Printf("Error starting device service: %v\n", err)
	} else if cfg.Devices.Enabled {
		fmt.Println("✓ Device event service started")
	}

	if err := channelManager.StartAll(ctx); err != nil {
		fmt.Printf("Error starting channels: %v\n", err)
	}

	healthServer := health.NewServer(cfg.Gateway.Host, cfg.Gateway.Port)
	healthServer.SetMessageBus(msgBus)
	healthServer.SetWorkspace(cfg.WorkspacePath())
	healthServer.SetObserverHealthFile(resolveMaintenanceHealthFile(cfg.WorkspacePath(), cfg.Maintenance.HealthFile))
	healthServer.SetObserverRestartFunc(func(ctx context.Context, service string) (map[string]any, error) {
		switch strings.ToLower(strings.TrimSpace(service)) {
		case "trigger":
			if err := triggerCtl.Restart(); err != nil {
				return nil, err
			}
			return map[string]any{
				"ok":      true,
				"action":  "restart",
				"message": "Trigger worker restart requested",
			}, nil
		case "brain", "brain-vllm", "cheap_cognition_vllm":
			if err := brainCtl.Restart(); err != nil {
				return nil, err
			}
			return map[string]any{
				"ok":      true,
				"action":  "restart",
				"message": "Brain runtime restart requested",
			}, nil
		default:
			return nil, fmt.Errorf("unsupported restart target: %s", service)
		}
	})
	// Register OpenClaw WebSocket bridge if enabled
	if openclawCh, ok := channelManager.GetChannel("openclaw"); ok {
		if oc, ok := openclawCh.(*channels.OpenClawChannel); ok {
			wsPath := cfg.Channels.OpenClaw.WebhookPath
			if wsPath == "" {
				wsPath = "/bridge/openclaw"
			}
			healthServer.RegisterWSHandler(wsPath, oc)
			logger.InfoCF("channels", "OpenClaw WebSocket bridge registered", map[string]any{
				"path": wsPath,
			})
		}
	}

	go func() {
		if err := healthServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.ErrorCF("health", "Health server error", map[string]any{"error": err.Error()})
		}
	}()
	fmt.Printf("✓ Health endpoints available at http://%s:%d/health and /ready\n", cfg.Gateway.Host, cfg.Gateway.Port)
	fmt.Printf(
		"✓ Valve endpoints available at http://%s:%d/valve/state and /valve/offer\n",
		cfg.Gateway.Host,
		cfg.Gateway.Port,
	)
	if cfg.Channels.OpenClaw.Enabled {
		wsPath := cfg.Channels.OpenClaw.WebhookPath
		if wsPath == "" {
			wsPath = "/bridge/openclaw"
		}
		fmt.Printf(
			"✓ OpenClaw bridge available at ws://%s:%d%s\n",
			cfg.Gateway.Host,
			cfg.Gateway.Port,
			wsPath,
		)
	}

	maintenanceService := maintenance.NewService(
		cfg.WorkspacePath(),
		cfg.Maintenance,
		cfg.Intake.CheapCognition,
		cfg.Trigger,
		agentLoop,
		triggerCtl,
		brainCtl,
	)
	healthServer.SetObserverSelfCareFunc(func(ctx context.Context) (map[string]any, error) {
		maintenanceService.RunOnce(ctx)
		return map[string]any{
			"ok":      true,
			"action":  "self_care_run",
			"message": "Self-care run requested",
		}, nil
	})
	if err := maintenanceService.Start(); err != nil {
		fmt.Printf("Error starting maintenance service: %v\n", err)
	} else if cfg.Maintenance.Enabled {
		fmt.Println("✓ Runtime maintenance service started")
	}

	go agentLoop.Run(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan

	fmt.Println("\nShutting down...")
	if cp, ok := provider.(providers.StatefulProvider); ok {
		cp.Close()
	}
	cancel()
	healthServer.Stop(context.Background())
	maintenanceService.Stop()
	if err := brainCtl.Stop(); err != nil {
		fmt.Printf("Error stopping cheap cognition runtime: %v\n", err)
	}
	if err := triggerCtl.Stop(); err != nil {
		fmt.Printf("Error stopping trigger worker: %v\n", err)
	}
	deviceService.Stop()
	heartbeatService.Stop()
	cronService.Stop()
	agentLoop.Stop()
	channelManager.StopAll(ctx)
	fmt.Println("✓ Gateway stopped")

	return nil
}

func resolveMaintenanceHealthFile(workspace, healthFile string) string {
	healthFile = strings.TrimSpace(healthFile)
	if healthFile != "" {
		return expandHome(healthFile)
	}
	return filepath.Join(workspace, "state", "runtime-health.json")
}

func expandHome(path string) string {
	if path == "" {
		return path
	}
	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		if len(path) > 1 && path[1] == '/' {
			return home + path[1:]
		}
		return home
	}
	return path
}

func setupCronTool(
	agentLoop *agent.AgentLoop,
	msgBus *bus.MessageBus,
	workspace string,
	restrict bool,
	execTimeout time.Duration,
	cfg *config.Config,
) *cron.CronService {
	cronStorePath := filepath.Join(workspace, "cron", "jobs.json")

	// Create cron service
	cronService := cron.NewCronService(cronStorePath, nil)

	// Create and register CronTool
	cronTool := tools.NewCronTool(cronService, agentLoop, msgBus, workspace, restrict, execTimeout, cfg)
	agentLoop.RegisterTool(cronTool)

	// Set the onJob handler
	cronService.SetOnJob(func(job *cron.CronJob) (string, error) {
		result := cronTool.ExecuteJob(context.Background(), job)
		return result, nil
	})

	return cronService
}
