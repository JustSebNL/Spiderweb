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
	"github.com/JustSebNL/Spiderweb/pkg/observer"
	"github.com/JustSebNL/Spiderweb/pkg/providers"
	"github.com/JustSebNL/Spiderweb/pkg/servicectl"
	"github.com/JustSebNL/Spiderweb/pkg/state"
	"github.com/JustSebNL/Spiderweb/pkg/systemdb"
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

	// Start observer journal scheduler after agent loop is running
	startJournalScheduler(ctx, cfg, agentLoop)

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

const journalJobName = "system-observer-journal"

// startJournalScheduler launches a goroutine that generates the daily observer journal
// near the configured rollover time. The journal agent wakes up, reads the day's events
// through the LLM, and writes a dark-humor entry grounded in real observer facts.
func startJournalScheduler(ctx context.Context, cfg *config.Config, agentLoop *agent.AgentLoop) {
	if !cfg.Observer.Journal.Enabled {
		return
	}

	workspace := cfg.WorkspacePath()
	healthFile := resolveMaintenanceHealthFile(workspace, cfg.Maintenance.HealthFile)

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		lastGeneratedDate := ""

		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				now = now.UTC()
				hour := cfg.Observer.Journal.RolloverHour
				minute := cfg.Observer.Journal.RolloverMinute
				if hour < 0 || hour > 23 {
					hour = 23
				}
				if minute < 0 || minute > 59 {
					minute = 50
				}

				// Only trigger within a 2-minute window after the rollover time
				if now.Hour() != hour || now.Minute() < minute || now.Minute() > minute+1 {
					continue
				}

				dateKey := now.Format("2006-01-02")
				if dateKey == lastGeneratedDate {
					continue
				}

				store := observer.NewStore(workspace, healthFile)
				prompt, err := buildJournalPrompt(store, dateKey)
				if err != nil {
					logger.WarnCF("observer", "Failed to build journal prompt", map[string]any{
						"error": err.Error(),
					})
					// Fall back to template-based generation
					if entry, fallbackErr := store.GenerateDailyJournalWithConfig(now); fallbackErr == nil {
						lastGeneratedDate = dateKey
						logger.InfoCF("observer", "Journal generated via fallback template", map[string]any{
							"date":  entry.Date,
							"title": entry.Title,
						})
					}
					continue
				}

				// Run the journal agent through the LLM
				response, err := agentLoop.ProcessJournal(ctx, prompt)
				if err != nil || strings.TrimSpace(response) == "" {
					logger.WarnCF("observer", "Journal agent returned empty or error", map[string]any{
						"error": err,
					})
					// Fall back to template-based generation
					if entry, fallbackErr := store.GenerateDailyJournalWithConfig(now); fallbackErr == nil {
						lastGeneratedDate = dateKey
						logger.InfoCF("observer", "Journal generated via fallback template", map[string]any{
							"date":  entry.Date,
							"title": entry.Title,
						})
					}
					continue
				}

				// Parse the agent response into title + body
				entry := parseJournalResponse(dateKey, response)

				// Apply max length cap
				maxLen := cfg.Observer.Journal.MaxLengthCap
				if maxLen > 0 && len(entry.Body) > maxLen {
					entry.Body = entry.Body[:maxLen-3] + "..."
				}

				// Persist via observer store
				if saveErr := store.SaveJournal(entry); saveErr != nil {
					logger.WarnCF("observer", "Failed to save journal entry", map[string]any{
						"error": saveErr.Error(),
					})
				} else {
					lastGeneratedDate = dateKey
					logger.InfoCF("observer", "Journal generated by agent", map[string]any{
						"date":  entry.Date,
						"title": entry.Title,
					})
				}
			}
		}
	}()

	logger.InfoCF("observer", "Journal scheduler started", map[string]any{
		"rollover_hour":   cfg.Observer.Journal.RolloverHour,
		"rollover_minute": cfg.Observer.Journal.RolloverMinute,
	})
}

// buildJournalPrompt collects the day's observer data and formats it into
// a prompt for the journal agent LLM.
func buildJournalPrompt(store *observer.Store, dateKey string) (string, error) {
	data, err := store.CollectJournalDayData(dateKey)
	if err != nil {
		return "", err
	}

	// Build event summary
	eventLines := []string{}
	for _, e := range data.Events {
		line := fmt.Sprintf("- [%s] %s: %s", e.Severity, e.EventType, e.Message)
		if e.ServiceID != "" {
			line += fmt.Sprintf(" (service: %s)", e.ServiceID)
		}
		eventLines = append(eventLines, line)
	}
	eventBlock := "No events recorded today."
	if len(eventLines) > 0 {
		eventBlock = strings.Join(eventLines, "\n")
	}

	// Build service summary
	serviceLines := []string{}
	for _, s := range data.Services {
		line := fmt.Sprintf("- %s: %s", s.Name, s.State)
		if s.Message != "" {
			line += fmt.Sprintf(" — %s", s.Message)
		}
		serviceLines = append(serviceLines, line)
	}
	serviceBlock := "No services tracked."
	if len(serviceLines) > 0 {
		serviceBlock = strings.Join(serviceLines, "\n")
	}

	prompt := fmt.Sprintf(`You are the Spiderweb Observer Journal agent. Your job is to write a daily journal entry about what happened in the Spiderweb system today.

RULES:
- Write in dark, dry humor style. Think Douglas Adams meets a tired sysadmin.
- Ground everything in the REAL data provided below. Do not invent events.
- If 2 or more services/agents had issues today, call it a "mutiny" or "riot."
- If things were calm, be suspicious about how calm it was.
- Always claim to be working on "world domination" in your free time between journal entries.
- Keep it 2-4 paragraphs. Punchy, sardonic, never mean-spirited.
- Include a short title at the top.

FORMAT YOUR RESPONSE EXACTLY LIKE THIS:
TITLE: <one-line title>
BODY: <2-4 paragraphs of dark-humor narrative>

--- TODAY'S DATA (date: %s) ---

Services:
%s

Events (%d total, %d errors, %d critical):
%s

24h Stats: total=%d errors=%d critical=%d warnings=%d

Self-care cycles tracked: %d (restarts: %d)

Latest health score: %d — %s

Troubled services: %d offline, %d degraded/restarting`,
		dateKey,
		serviceBlock,
		len(data.Events), data.Stats.ErrorEvents, data.Stats.CriticalEvents, eventBlock,
		data.Stats.TotalEvents, data.Stats.ErrorEvents, data.Stats.CriticalEvents, data.Stats.WarningEvents,
		len(data.Cycles), data.RestartCount,
		data.Score, data.ScoreSummary,
		data.OfflineCount, data.DegradedCount,
	)

	return prompt, nil
}

// parseJournalResponse extracts title and body from the LLM journal response.
func parseJournalResponse(dateKey, response string) systemdb.JournalEntry {
	title := "Daily operations log"
	body := strings.TrimSpace(response)

	// Try to parse TITLE: / BODY: format
	lines := strings.SplitN(response, "\n", 2)
	firstLine := strings.TrimSpace(lines[0])

	if strings.HasPrefix(strings.ToUpper(firstLine), "TITLE:") {
		title = strings.TrimSpace(firstLine[6:])
		if len(lines) > 1 {
			remaining := strings.TrimSpace(lines[1])
			if strings.HasPrefix(strings.ToUpper(remaining), "BODY:") {
				body = strings.TrimSpace(remaining[5:])
			} else {
				body = remaining
			}
		}
	}

	// Clean up any markdown formatting
	title = strings.Trim(title, "#* ")
	body = strings.Trim(body, "\n")

	return systemdb.JournalEntry{
		Date:      dateKey,
		Title:     title,
		Body:      body,
		Style:     "dark_humor",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}
