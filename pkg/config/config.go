package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/caarlos0/env/v11"
)

// rrCounter is a global counter for round-robin load balancing across models.
var rrCounter atomic.Uint64

// FlexibleStringSlice is a []string that also accepts JSON numbers,
// so allow_from can contain both "123" and 123.
type FlexibleStringSlice []string

func (f *FlexibleStringSlice) UnmarshalJSON(data []byte) error {
	// Try []string first
	var ss []string
	if err := json.Unmarshal(data, &ss); err == nil {
		*f = ss
		return nil
	}

	// Try []interface{} to handle mixed types
	var raw []any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	result := make([]string, 0, len(raw))
	for _, v := range raw {
		switch val := v.(type) {
		case string:
			result = append(result, val)
		case float64:
			result = append(result, fmt.Sprintf("%.0f", val))
		default:
			result = append(result, fmt.Sprintf("%v", val))
		}
	}
	*f = result
	return nil
}

type Config struct {
	Agents      AgentsConfig      `json:"agents"`
	Bindings    []AgentBinding    `json:"bindings,omitempty"`
	Session     SessionConfig     `json:"session,omitempty"`
	Channels    ChannelsConfig    `json:"channels"`
	Providers   ProvidersConfig   `json:"providers,omitempty"`
	ModelList   []ModelConfig     `json:"model_list"` // New model-centric provider configuration
	Gateway     GatewayConfig     `json:"gateway"`
	Tools       ToolsConfig       `json:"tools"`
	Intake      IntakeConfig      `json:"intake"`
	Trigger     TriggerConfig     `json:"trigger"`
	Maintenance MaintenanceConfig `json:"maintenance"`
	Heartbeat   HeartbeatConfig   `json:"heartbeat"`
	Devices     DevicesConfig     `json:"devices"`
	Observer    ObserverConfig    `json:"observer"`
}

// MarshalJSON implements custom JSON marshaling for Config
// to omit providers section when empty and session when empty
func (c Config) MarshalJSON() ([]byte, error) {
	type Alias Config
	aux := &struct {
		Providers *ProvidersConfig `json:"providers,omitempty"`
		Session   *SessionConfig   `json:"session,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(&c),
	}

	// Only include providers if not empty
	if !c.Providers.IsEmpty() {
		aux.Providers = &c.Providers
	}

	// Only include session if not empty
	if c.Session.DMScope != "" || len(c.Session.IdentityLinks) > 0 {
		aux.Session = &c.Session
	}

	return json.Marshal(aux)
}

type AgentsConfig struct {
	Defaults AgentDefaults `json:"defaults"`
	List     []AgentConfig `json:"list,omitempty"`
}

// AgentModelConfig supports both string and structured model config.
// String format: "gpt-4" (just primary, no fallbacks)
// Object format: {"primary": "gpt-4", "fallbacks": ["claude-haiku"]}
type AgentModelConfig struct {
	Primary   string   `json:"primary,omitempty"`
	Fallbacks []string `json:"fallbacks,omitempty"`
}

func (m *AgentModelConfig) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		m.Primary = s
		m.Fallbacks = nil
		return nil
	}
	type raw struct {
		Primary   string   `json:"primary"`
		Fallbacks []string `json:"fallbacks"`
	}
	var r raw
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}
	m.Primary = r.Primary
	m.Fallbacks = r.Fallbacks
	return nil
}

func (m AgentModelConfig) MarshalJSON() ([]byte, error) {
	if len(m.Fallbacks) == 0 && m.Primary != "" {
		return json.Marshal(m.Primary)
	}
	type raw struct {
		Primary   string   `json:"primary,omitempty"`
		Fallbacks []string `json:"fallbacks,omitempty"`
	}
	return json.Marshal(raw{Primary: m.Primary, Fallbacks: m.Fallbacks})
}

type AgentConfig struct {
	ID        string            `json:"id"`
	Default   bool              `json:"default,omitempty"`
	Name      string            `json:"name,omitempty"`
	Workspace string            `json:"workspace,omitempty"`
	Model     *AgentModelConfig `json:"model,omitempty"`
	Skills    []string          `json:"skills,omitempty"`
	Subagents *SubagentsConfig  `json:"subagents,omitempty"`
}

type SubagentsConfig struct {
	AllowAgents []string          `json:"allow_agents,omitempty"`
	Model       *AgentModelConfig `json:"model,omitempty"`
}

type PeerMatch struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type BindingMatch struct {
	Channel   string     `json:"channel"`
	AccountID string     `json:"account_id,omitempty"`
	Peer      *PeerMatch `json:"peer,omitempty"`
	GuildID   string     `json:"guild_id,omitempty"`
	TeamID    string     `json:"team_id,omitempty"`
}

type AgentBinding struct {
	AgentID string       `json:"agent_id"`
	Match   BindingMatch `json:"match"`
}

type SessionConfig struct {
	DMScope                 string              `json:"dm_scope,omitempty"`
	IdentityLinks           map[string][]string `json:"identity_links,omitempty"`
	AutoSummarize           bool                `json:"auto_summarize"            env:"SPIDERWEB_SESSION_AUTO_SUMMARIZE"`
	CompactPrompt           bool                `json:"compact_prompt"            env:"SPIDERWEB_SESSION_COMPACT_PROMPT"`
	IncludeBootstrap        bool                `json:"include_bootstrap"         env:"SPIDERWEB_SESSION_INCLUDE_BOOTSTRAP"`
	IncludeSkillsSummary    bool                `json:"include_skills_summary"    env:"SPIDERWEB_SESSION_INCLUDE_SKILLS_SUMMARY"`
	IncludeToolDescriptions bool                `json:"include_tool_descriptions" env:"SPIDERWEB_SESSION_INCLUDE_TOOL_DESCRIPTIONS"`
	IncludeMemoryContext    bool                `json:"include_memory_context"    env:"SPIDERWEB_SESSION_INCLUDE_MEMORY_CONTEXT"`
	MemoryContextMaxChars   int                 `json:"memory_context_max_chars"  env:"SPIDERWEB_SESSION_MEMORY_CONTEXT_MAX_CHARS"`
	RecentDailyNotesDays    int                 `json:"recent_daily_notes_days"   env:"SPIDERWEB_SESSION_RECENT_DAILY_NOTES_DAYS"`
	ToolResultMaxChars      int                 `json:"tool_result_max_chars"     env:"SPIDERWEB_SESSION_TOOL_RESULT_MAX_CHARS"`
}

type AgentDefaults struct {
	Workspace           string   `json:"workspace"                       env:"SPIDERWEB_AGENTS_DEFAULTS_WORKSPACE"`
	RestrictToWorkspace bool     `json:"restrict_to_workspace"           env:"SPIDERWEB_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE"`
	Provider            string   `json:"provider"                        env:"SPIDERWEB_AGENTS_DEFAULTS_PROVIDER"`
	ModelName           string   `json:"model_name,omitempty"            env:"SPIDERWEB_AGENTS_DEFAULTS_MODEL_NAME"`
	Model               string   `json:"model,omitempty"                 env:"SPIDERWEB_AGENTS_DEFAULTS_MODEL"` // Deprecated: use model_name instead
	ModelFallbacks      []string `json:"model_fallbacks,omitempty"`
	ImageModel          string   `json:"image_model,omitempty"           env:"SPIDERWEB_AGENTS_DEFAULTS_IMAGE_MODEL"`
	ImageModelFallbacks []string `json:"image_model_fallbacks,omitempty"`
	MaxTokens           int      `json:"max_tokens"                      env:"SPIDERWEB_AGENTS_DEFAULTS_MAX_TOKENS"`
	Temperature         *float64 `json:"temperature,omitempty"           env:"SPIDERWEB_AGENTS_DEFAULTS_TEMPERATURE"`
	MaxToolIterations   int      `json:"max_tool_iterations"             env:"SPIDERWEB_AGENTS_DEFAULTS_MAX_TOOL_ITERATIONS"`
}

// GetModelName returns the effective model name for the agent defaults.
// It prefers the new "model_name" field but falls back to "model" for backward compatibility.
func (d *AgentDefaults) GetModelName() string {
	if d.ModelName != "" {
		return d.ModelName
	}
	return d.Model
}

type ChannelsConfig struct {
	WhatsApp WhatsAppConfig `json:"whatsapp"`
	Telegram TelegramConfig `json:"telegram"`
	Feishu   FeishuConfig   `json:"feishu"`
	Discord  DiscordConfig  `json:"discord"`
	MaixCam  MaixCamConfig  `json:"maixcam"`
	QQ       QQConfig       `json:"qq"`
	DingTalk DingTalkConfig `json:"dingtalk"`
	Slack    SlackConfig    `json:"slack"`
	LINE     LINEConfig     `json:"line"`
	OneBot   OneBotConfig   `json:"onebot"`
	WeCom    WeComConfig    `json:"wecom"`
	WeComApp WeComAppConfig `json:"wecom_app"`
	OpenClaw OpenClawConfig `json:"openclaw"`
}

type WhatsAppConfig struct {
	Enabled   bool                `json:"enabled"    env:"SPIDERWEB_CHANNELS_WHATSAPP_ENABLED"`
	BridgeURL string              `json:"bridge_url" env:"SPIDERWEB_CHANNELS_WHATSAPP_BRIDGE_URL"`
	AllowFrom FlexibleStringSlice `json:"allow_from" env:"SPIDERWEB_CHANNELS_WHATSAPP_ALLOW_FROM"`
}

type TelegramConfig struct {
	Enabled   bool                `json:"enabled"    env:"SPIDERWEB_CHANNELS_TELEGRAM_ENABLED"`
	Token     string              `json:"token"      env:"SPIDERWEB_CHANNELS_TELEGRAM_TOKEN"`
	Proxy     string              `json:"proxy"      env:"SPIDERWEB_CHANNELS_TELEGRAM_PROXY"`
	AllowFrom FlexibleStringSlice `json:"allow_from" env:"SPIDERWEB_CHANNELS_TELEGRAM_ALLOW_FROM"`
}

type FeishuConfig struct {
	Enabled           bool                `json:"enabled"            env:"SPIDERWEB_CHANNELS_FEISHU_ENABLED"`
	AppID             string              `json:"app_id"             env:"SPIDERWEB_CHANNELS_FEISHU_APP_ID"`
	AppSecret         string              `json:"app_secret"         env:"SPIDERWEB_CHANNELS_FEISHU_APP_SECRET"`
	EncryptKey        string              `json:"encrypt_key"        env:"SPIDERWEB_CHANNELS_FEISHU_ENCRYPT_KEY"`
	VerificationToken string              `json:"verification_token" env:"SPIDERWEB_CHANNELS_FEISHU_VERIFICATION_TOKEN"`
	AllowFrom         FlexibleStringSlice `json:"allow_from"         env:"SPIDERWEB_CHANNELS_FEISHU_ALLOW_FROM"`
}

type DiscordConfig struct {
	Enabled     bool                `json:"enabled"      env:"SPIDERWEB_CHANNELS_DISCORD_ENABLED"`
	Token       string              `json:"token"        env:"SPIDERWEB_CHANNELS_DISCORD_TOKEN"`
	AllowFrom   FlexibleStringSlice `json:"allow_from"   env:"SPIDERWEB_CHANNELS_DISCORD_ALLOW_FROM"`
	MentionOnly bool                `json:"mention_only" env:"SPIDERWEB_CHANNELS_DISCORD_MENTION_ONLY"`
}

type MaixCamConfig struct {
	Enabled   bool                `json:"enabled"    env:"SPIDERWEB_CHANNELS_MAIXCAM_ENABLED"`
	Host      string              `json:"host"       env:"SPIDERWEB_CHANNELS_MAIXCAM_HOST"`
	Port      int                 `json:"port"       env:"SPIDERWEB_CHANNELS_MAIXCAM_PORT"`
	AllowFrom FlexibleStringSlice `json:"allow_from" env:"SPIDERWEB_CHANNELS_MAIXCAM_ALLOW_FROM"`
}

type QQConfig struct {
	Enabled   bool                `json:"enabled"    env:"SPIDERWEB_CHANNELS_QQ_ENABLED"`
	AppID     string              `json:"app_id"     env:"SPIDERWEB_CHANNELS_QQ_APP_ID"`
	AppSecret string              `json:"app_secret" env:"SPIDERWEB_CHANNELS_QQ_APP_SECRET"`
	AllowFrom FlexibleStringSlice `json:"allow_from" env:"SPIDERWEB_CHANNELS_QQ_ALLOW_FROM"`
}

type DingTalkConfig struct {
	Enabled      bool                `json:"enabled"       env:"SPIDERWEB_CHANNELS_DINGTALK_ENABLED"`
	ClientID     string              `json:"client_id"     env:"SPIDERWEB_CHANNELS_DINGTALK_CLIENT_ID"`
	ClientSecret string              `json:"client_secret" env:"SPIDERWEB_CHANNELS_DINGTALK_CLIENT_SECRET"`
	AllowFrom    FlexibleStringSlice `json:"allow_from"    env:"SPIDERWEB_CHANNELS_DINGTALK_ALLOW_FROM"`
}

type SlackConfig struct {
	Enabled   bool                `json:"enabled"    env:"SPIDERWEB_CHANNELS_SLACK_ENABLED"`
	BotToken  string              `json:"bot_token"  env:"SPIDERWEB_CHANNELS_SLACK_BOT_TOKEN"`
	AppToken  string              `json:"app_token"  env:"SPIDERWEB_CHANNELS_SLACK_APP_TOKEN"`
	AllowFrom FlexibleStringSlice `json:"allow_from" env:"SPIDERWEB_CHANNELS_SLACK_ALLOW_FROM"`
}

type LINEConfig struct {
	Enabled            bool                `json:"enabled"              env:"SPIDERWEB_CHANNELS_LINE_ENABLED"`
	ChannelSecret      string              `json:"channel_secret"       env:"SPIDERWEB_CHANNELS_LINE_CHANNEL_SECRET"`
	ChannelAccessToken string              `json:"channel_access_token" env:"SPIDERWEB_CHANNELS_LINE_CHANNEL_ACCESS_TOKEN"`
	WebhookHost        string              `json:"webhook_host"         env:"SPIDERWEB_CHANNELS_LINE_WEBHOOK_HOST"`
	WebhookPort        int                 `json:"webhook_port"         env:"SPIDERWEB_CHANNELS_LINE_WEBHOOK_PORT"`
	WebhookPath        string              `json:"webhook_path"         env:"SPIDERWEB_CHANNELS_LINE_WEBHOOK_PATH"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"           env:"SPIDERWEB_CHANNELS_LINE_ALLOW_FROM"`
}

type OneBotConfig struct {
	Enabled            bool                `json:"enabled"              env:"SPIDERWEB_CHANNELS_ONEBOT_ENABLED"`
	WSUrl              string              `json:"ws_url"               env:"SPIDERWEB_CHANNELS_ONEBOT_WS_URL"`
	AccessToken        string              `json:"access_token"         env:"SPIDERWEB_CHANNELS_ONEBOT_ACCESS_TOKEN"`
	ReconnectInterval  int                 `json:"reconnect_interval"   env:"SPIDERWEB_CHANNELS_ONEBOT_RECONNECT_INTERVAL"`
	GroupTriggerPrefix []string            `json:"group_trigger_prefix" env:"SPIDERWEB_CHANNELS_ONEBOT_GROUP_TRIGGER_PREFIX"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"           env:"SPIDERWEB_CHANNELS_ONEBOT_ALLOW_FROM"`
}

type WeComConfig struct {
	Enabled        bool                `json:"enabled"          env:"SPIDERWEB_CHANNELS_WECOM_ENABLED"`
	Token          string              `json:"token"            env:"SPIDERWEB_CHANNELS_WECOM_TOKEN"`
	EncodingAESKey string              `json:"encoding_aes_key" env:"SPIDERWEB_CHANNELS_WECOM_ENCODING_AES_KEY"`
	WebhookURL     string              `json:"webhook_url"      env:"SPIDERWEB_CHANNELS_WECOM_WEBHOOK_URL"`
	WebhookHost    string              `json:"webhook_host"     env:"SPIDERWEB_CHANNELS_WECOM_WEBHOOK_HOST"`
	WebhookPort    int                 `json:"webhook_port"     env:"SPIDERWEB_CHANNELS_WECOM_WEBHOOK_PORT"`
	WebhookPath    string              `json:"webhook_path"     env:"SPIDERWEB_CHANNELS_WECOM_WEBHOOK_PATH"`
	AllowFrom      FlexibleStringSlice `json:"allow_from"       env:"SPIDERWEB_CHANNELS_WECOM_ALLOW_FROM"`
	ReplyTimeout   int                 `json:"reply_timeout"    env:"SPIDERWEB_CHANNELS_WECOM_REPLY_TIMEOUT"`
}

type WeComAppConfig struct {
	Enabled        bool                `json:"enabled"          env:"SPIDERWEB_CHANNELS_WECOM_APP_ENABLED"`
	CorpID         string              `json:"corp_id"          env:"SPIDERWEB_CHANNELS_WECOM_APP_CORP_ID"`
	CorpSecret     string              `json:"corp_secret"      env:"SPIDERWEB_CHANNELS_WECOM_APP_CORP_SECRET"`
	AgentID        int64               `json:"agent_id"         env:"SPIDERWEB_CHANNELS_WECOM_APP_AGENT_ID"`
	Token          string              `json:"token"            env:"SPIDERWEB_CHANNELS_WECOM_APP_TOKEN"`
	EncodingAESKey string              `json:"encoding_aes_key" env:"SPIDERWEB_CHANNELS_WECOM_APP_ENCODING_AES_KEY"`
	WebhookHost    string              `json:"webhook_host"     env:"SPIDERWEB_CHANNELS_WECOM_APP_WEBHOOK_HOST"`
	WebhookPort    int                 `json:"webhook_port"     env:"SPIDERWEB_CHANNELS_WECOM_APP_WEBHOOK_PORT"`
	WebhookPath    string              `json:"webhook_path"     env:"SPIDERWEB_CHANNELS_WECOM_APP_WEBHOOK_PATH"`
	AllowFrom      FlexibleStringSlice `json:"allow_from"       env:"SPIDERWEB_CHANNELS_WECOM_APP_ALLOW_FROM"`
	ReplyTimeout   int                 `json:"reply_timeout"    env:"SPIDERWEB_CHANNELS_WECOM_APP_REPLY_TIMEOUT"`
}

type OpenClawConfig struct {
	Enabled       bool                `json:"enabled"          env:"SPIDERWEB_CHANNELS_OPENCLAW_ENABLED"`
	SharedSecret  string              `json:"shared_secret"    env:"SPIDERWEB_CHANNELS_OPENCLAW_SHARED_SECRET"`
	AllowFrom     FlexibleStringSlice `json:"allow_from"       env:"SPIDERWEB_CHANNELS_OPENCLAW_ALLOW_FROM"`
	AutoHandshake bool                `json:"auto_handshake"   env:"SPIDERWEB_CHANNELS_OPENCLAW_AUTO_HANDSHAKE"`
	IntakeEnabled bool                `json:"intake_enabled"   env:"SPIDERWEB_CHANNELS_OPENCLAW_INTAKE_ENABLED"`
	WebhookPath   string              `json:"webhook_path"     env:"SPIDERWEB_CHANNELS_OPENCLAW_WEBHOOK_PATH"`
}

type HeartbeatConfig struct {
	Enabled  bool `json:"enabled"  env:"SPIDERWEB_HEARTBEAT_ENABLED"`
	Interval int  `json:"interval" env:"SPIDERWEB_HEARTBEAT_INTERVAL"` // minutes, min 5
}

type TriggerConfig struct {
	Enabled   bool   `json:"enabled"    env:"SPIDERWEB_TRIGGER_ENABLED"`
	AutoStart bool   `json:"auto_start" env:"SPIDERWEB_TRIGGER_AUTO_START"`
	Workdir   string `json:"workdir"    env:"SPIDERWEB_TRIGGER_WORKDIR"`
	PIDFile   string `json:"pid_file"   env:"SPIDERWEB_TRIGGER_PID_FILE"`
	LogFile   string `json:"log_file"   env:"SPIDERWEB_TRIGGER_LOG_FILE"`
	Host      string `json:"host"       env:"SPIDERWEB_TRIGGER_HOST"`
	Port      int    `json:"port"       env:"SPIDERWEB_TRIGGER_PORT"`
}

type MaintenanceConfig struct {
	Enabled               bool   `json:"enabled"                  env:"SPIDERWEB_MAINTENANCE_ENABLED"`
	IntervalHours         int    `json:"interval_hours"           env:"SPIDERWEB_MAINTENANCE_INTERVAL_HOURS"`
	AutoRemediate         bool   `json:"auto_remediate"           env:"SPIDERWEB_MAINTENANCE_AUTO_REMEDIATE"`
	HealthFile            string `json:"health_file"              env:"SPIDERWEB_MAINTENANCE_HEALTH_FILE"`
	BudgetPercent         int    `json:"budget_percent"           env:"SPIDERWEB_MAINTENANCE_BUDGET_PERCENT"`
	BusyWindowMinutes     int    `json:"busy_window_minutes"      env:"SPIDERWEB_MAINTENANCE_BUSY_WINDOW_MINUTES"`
	RestartBackoffMinutes int    `json:"restart_backoff_minutes"  env:"SPIDERWEB_MAINTENANCE_RESTART_BACKOFF_MINUTES"`
	MaxLogMB              int    `json:"max_log_mb"               env:"SPIDERWEB_MAINTENANCE_MAX_LOG_MB"`
	HighLatencyMs         int    `json:"high_latency_ms"          env:"SPIDERWEB_MAINTENANCE_HIGH_LATENCY_MS"`
	MaxCheapFailures      int    `json:"max_cheap_failures"       env:"SPIDERWEB_MAINTENANCE_MAX_CHEAP_FAILURES"`
	MaxForwardSkips       int    `json:"max_forward_skips"        env:"SPIDERWEB_MAINTENANCE_MAX_FORWARD_SKIPS"`
	RestartOnProcessDeath bool   `json:"restart_on_process_death" env:"SPIDERWEB_MAINTENANCE_RESTART_ON_PROCESS_DEATH"`
}

type CheapCognitionConfig struct {
	Enabled        bool   `json:"enabled"         env:"SPIDERWEB_INTAKE_CHEAP_COGNITION_ENABLED"`
	Runtime        string `json:"runtime"         env:"SPIDERWEB_INTAKE_CHEAP_COGNITION_RUNTIME"`
	BaseURL        string `json:"base_url"        env:"SPIDERWEB_INTAKE_CHEAP_COGNITION_BASE_URL"`
	APIKey         string `json:"api_key"         env:"SPIDERWEB_INTAKE_CHEAP_COGNITION_API_KEY"`
	Model          string `json:"model"           env:"SPIDERWEB_INTAKE_CHEAP_COGNITION_MODEL"`
	TimeoutSeconds int    `json:"timeout_seconds" env:"SPIDERWEB_INTAKE_CHEAP_COGNITION_TIMEOUT_SECONDS"`
}

type IntakeConfig struct {
	Enabled              bool                 `json:"enabled"               env:"SPIDERWEB_INTAKE_ENABLED"`
	CoalesceWindowMs     int                  `json:"coalesce_window_ms"    env:"SPIDERWEB_INTAKE_COALESCE_WINDOW_MS"`
	MaxBatchMessages     int                  `json:"max_batch_messages"    env:"SPIDERWEB_INTAKE_MAX_BATCH_MESSAGES"`
	MaxBatchChars        int                  `json:"max_batch_chars"       env:"SPIDERWEB_INTAKE_MAX_BATCH_CHARS"`
	DedupeWindowSeconds  int                  `json:"dedupe_window_seconds" env:"SPIDERWEB_INTAKE_DEDUPE_WINDOW_SECONDS"`
	FollowUpEnabled      bool                 `json:"follow_up_enabled"     env:"SPIDERWEB_INTAKE_FOLLOW_UP_ENABLED"`
	FollowUpWindowSec    int                  `json:"follow_up_window_sec"  env:"SPIDERWEB_INTAKE_FOLLOW_UP_WINDOW_SEC"`
	FollowUpMaxItems     int                  `json:"follow_up_max_items"   env:"SPIDERWEB_INTAKE_FOLLOW_UP_MAX_ITEMS"`
	ForwardURL           string               `json:"forward_url"           env:"SPIDERWEB_INTAKE_FORWARD_URL"`
	ForwardTimeout       int                  `json:"forward_timeout"       env:"SPIDERWEB_INTAKE_FORWARD_TIMEOUT"`
	ForwardAllowChannels FlexibleStringSlice  `json:"forward_allow_channels" env:"SPIDERWEB_INTAKE_FORWARD_ALLOW_CHANNELS"`
	ForwardDenyChannels  FlexibleStringSlice  `json:"forward_deny_channels"  env:"SPIDERWEB_INTAKE_FORWARD_DENY_CHANNELS"`
	ForwardAllowServices FlexibleStringSlice  `json:"forward_allow_services" env:"SPIDERWEB_INTAKE_FORWARD_ALLOW_SERVICES"`
	ForwardDenyServices  FlexibleStringSlice  `json:"forward_deny_services"  env:"SPIDERWEB_INTAKE_FORWARD_DENY_SERVICES"`
	CheapCognition       CheapCognitionConfig `json:"cheap_cognition"`
}

type DevicesConfig struct {
	Enabled    bool `json:"enabled"     env:"SPIDERWEB_DEVICES_ENABLED"`
	MonitorUSB bool `json:"monitor_usb" env:"SPIDERWEB_DEVICES_MONITOR_USB"`
}

type ObserverConfig struct {
	Journal ObserverJournalConfig `json:"journal"`
}

type ObserverJournalConfig struct {
	Enabled        bool   `json:"enabled"`
	RolloverHour   int    `json:"rollover_hour"`
	RolloverMinute int    `json:"rollover_minute"`
	StyleMode      string `json:"style_mode"`
	MaxLengthCap   int    `json:"max_length_cap"`
	Timezone       string `json:"timezone"`
}

func (o ObserverConfig) withDefaults() ObserverConfig {
	if !o.Journal.Enabled && o.Journal.RolloverHour == 0 && o.Journal.RolloverMinute == 0 {
		o.Journal.Enabled = true
	}
	if o.Journal.RolloverHour < 0 || o.Journal.RolloverHour > 23 {
		o.Journal.RolloverHour = 23
	}
	if o.Journal.RolloverMinute < 0 || o.Journal.RolloverMinute > 59 {
		o.Journal.RolloverMinute = 50
	}
	if o.Journal.StyleMode == "" {
		o.Journal.StyleMode = "dark_humor"
	}
	if o.Journal.MaxLengthCap < 0 {
		o.Journal.MaxLengthCap = 2000
	}
	if o.Journal.Timezone == "" {
		o.Journal.Timezone = "UTC"
	}
	return o
}

type ProvidersConfig struct {
	Anthropic     ProviderConfig       `json:"anthropic"`
	OpenAI        OpenAIProviderConfig `json:"openai"`
	OpenRouter    ProviderConfig       `json:"openrouter"`
	Groq          ProviderConfig       `json:"groq"`
	Zhipu         ProviderConfig       `json:"zhipu"`
	VLLM          ProviderConfig       `json:"vllm"`
	Gemini        ProviderConfig       `json:"gemini"`
	Nvidia        ProviderConfig       `json:"nvidia"`
	Ollama        ProviderConfig       `json:"ollama"`
	Moonshot      ProviderConfig       `json:"moonshot"`
	ShengSuanYun  ProviderConfig       `json:"shengsuanyun"`
	DeepSeek      ProviderConfig       `json:"deepseek"`
	Cerebras      ProviderConfig       `json:"cerebras"`
	VolcEngine    ProviderConfig       `json:"volcengine"`
	GitHubCopilot ProviderConfig       `json:"github_copilot"`
	Antigravity   ProviderConfig       `json:"antigravity"`
	Qwen          ProviderConfig       `json:"qwen"`
	Mistral       ProviderConfig       `json:"mistral"`
}

// IsEmpty checks if all provider configs are empty (no API keys or API bases set)
// Note: WebSearch is an optimization option and doesn't count as "non-empty"
func (p ProvidersConfig) IsEmpty() bool {
	return p.Anthropic.APIKey == "" && p.Anthropic.APIBase == "" &&
		p.OpenAI.APIKey == "" && p.OpenAI.APIBase == "" &&
		p.OpenRouter.APIKey == "" && p.OpenRouter.APIBase == "" &&
		p.Groq.APIKey == "" && p.Groq.APIBase == "" &&
		p.Zhipu.APIKey == "" && p.Zhipu.APIBase == "" &&
		p.VLLM.APIKey == "" && p.VLLM.APIBase == "" &&
		p.Gemini.APIKey == "" && p.Gemini.APIBase == "" &&
		p.Nvidia.APIKey == "" && p.Nvidia.APIBase == "" &&
		p.Ollama.APIKey == "" && p.Ollama.APIBase == "" &&
		p.Moonshot.APIKey == "" && p.Moonshot.APIBase == "" &&
		p.ShengSuanYun.APIKey == "" && p.ShengSuanYun.APIBase == "" &&
		p.DeepSeek.APIKey == "" && p.DeepSeek.APIBase == "" &&
		p.Cerebras.APIKey == "" && p.Cerebras.APIBase == "" &&
		p.VolcEngine.APIKey == "" && p.VolcEngine.APIBase == "" &&
		p.GitHubCopilot.APIKey == "" && p.GitHubCopilot.APIBase == "" &&
		p.Antigravity.APIKey == "" && p.Antigravity.APIBase == "" &&
		p.Qwen.APIKey == "" && p.Qwen.APIBase == "" &&
		p.Mistral.APIKey == "" && p.Mistral.APIBase == ""
}

// MarshalJSON implements custom JSON marshaling for ProvidersConfig
// to omit the entire section when empty
func (p ProvidersConfig) MarshalJSON() ([]byte, error) {
	if p.IsEmpty() {
		return []byte("null"), nil
	}
	type Alias ProvidersConfig
	return json.Marshal((*Alias)(&p))
}

type ProviderConfig struct {
	APIKey      string `json:"api_key"                env:"SPIDERWEB_PROVIDERS_{{.Name}}_API_KEY"`
	APIBase     string `json:"api_base"               env:"SPIDERWEB_PROVIDERS_{{.Name}}_API_BASE"`
	Proxy       string `json:"proxy,omitempty"        env:"SPIDERWEB_PROVIDERS_{{.Name}}_PROXY"`
	AuthMethod  string `json:"auth_method,omitempty"  env:"SPIDERWEB_PROVIDERS_{{.Name}}_AUTH_METHOD"`
	ConnectMode string `json:"connect_mode,omitempty" env:"SPIDERWEB_PROVIDERS_{{.Name}}_CONNECT_MODE"` // only for Github Copilot, `stdio` or `grpc`
}

type OpenAIProviderConfig struct {
	ProviderConfig
	WebSearch bool `json:"web_search" env:"SPIDERWEB_PROVIDERS_OPENAI_WEB_SEARCH"`
}

// ModelConfig represents a model-centric provider configuration.
// It allows adding new providers (especially OpenAI-compatible ones) via configuration only.
// The model field uses protocol prefix format: [protocol/]model-identifier
// Supported protocols: openai, anthropic, antigravity, claude-cli, codex-cli, github-copilot
// Default protocol is "openai" if no prefix is specified.
type ModelConfig struct {
	// Required fields
	ModelName string `json:"model_name"` // User-facing alias for the model
	Model     string `json:"model"`      // Protocol/model-identifier (e.g., "openai/gpt-4o", "anthropic/claude-sonnet-4.6")

	// HTTP-based providers
	APIBase string `json:"api_base,omitempty"` // API endpoint URL
	APIKey  string `json:"api_key"`            // API authentication key
	Proxy   string `json:"proxy,omitempty"`    // HTTP proxy URL

	// Special providers (CLI-based, OAuth, etc.)
	AuthMethod  string `json:"auth_method,omitempty"`  // Authentication method: oauth, token
	ConnectMode string `json:"connect_mode,omitempty"` // Connection mode: stdio, grpc
	Workspace   string `json:"workspace,omitempty"`    // Workspace path for CLI-based providers

	// Optional optimizations
	RPM            int    `json:"rpm,omitempty"`              // Requests per minute limit
	MaxTokensField string `json:"max_tokens_field,omitempty"` // Field name for max tokens (e.g., "max_completion_tokens")
}

// Validate checks if the ModelConfig has all required fields.
func (c *ModelConfig) Validate() error {
	if c.ModelName == "" {
		return fmt.Errorf("model_name is required")
	}
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}

type GatewayConfig struct {
	Host string `json:"host" env:"SPIDERWEB_GATEWAY_HOST"`
	Port int    `json:"port" env:"SPIDERWEB_GATEWAY_PORT"`
}

type BraveConfig struct {
	Enabled    bool   `json:"enabled"     env:"SPIDERWEB_TOOLS_WEB_BRAVE_ENABLED"`
	APIKey     string `json:"api_key"     env:"SPIDERWEB_TOOLS_WEB_BRAVE_API_KEY"`
	MaxResults int    `json:"max_results" env:"SPIDERWEB_TOOLS_WEB_BRAVE_MAX_RESULTS"`
}

type TavilyConfig struct {
	Enabled    bool   `json:"enabled"     env:"SPIDERWEB_TOOLS_WEB_TAVILY_ENABLED"`
	APIKey     string `json:"api_key"     env:"SPIDERWEB_TOOLS_WEB_TAVILY_API_KEY"`
	BaseURL    string `json:"base_url"    env:"SPIDERWEB_TOOLS_WEB_TAVILY_BASE_URL"`
	MaxResults int    `json:"max_results" env:"SPIDERWEB_TOOLS_WEB_TAVILY_MAX_RESULTS"`
}

type DuckDuckGoConfig struct {
	Enabled    bool `json:"enabled"     env:"SPIDERWEB_TOOLS_WEB_DUCKDUCKGO_ENABLED"`
	MaxResults int  `json:"max_results" env:"SPIDERWEB_TOOLS_WEB_DUCKDUCKGO_MAX_RESULTS"`
}

type PerplexityConfig struct {
	Enabled    bool   `json:"enabled"     env:"SPIDERWEB_TOOLS_WEB_PERPLEXITY_ENABLED"`
	APIKey     string `json:"api_key"     env:"SPIDERWEB_TOOLS_WEB_PERPLEXITY_API_KEY"`
	MaxResults int    `json:"max_results" env:"SPIDERWEB_TOOLS_WEB_PERPLEXITY_MAX_RESULTS"`
}

type WebToolsConfig struct {
	Brave      BraveConfig      `json:"brave"`
	Tavily     TavilyConfig     `json:"tavily"`
	DuckDuckGo DuckDuckGoConfig `json:"duckduckgo"`
	Perplexity PerplexityConfig `json:"perplexity"`
	// Proxy is an optional proxy URL for web tools (http/https/socks5/socks5h).
	// For authenticated proxies, prefer HTTP_PROXY/HTTPS_PROXY env vars instead of embedding credentials in config.
	Proxy string `json:"proxy,omitempty" env:"SPIDERWEB_TOOLS_WEB_PROXY"`
}

type CronToolsConfig struct {
	ExecTimeoutMinutes int `json:"exec_timeout_minutes" env:"SPIDERWEB_TOOLS_CRON_EXEC_TIMEOUT_MINUTES"` // 0 means no timeout
}

type ExecConfig struct {
	EnableDenyPatterns bool     `json:"enable_deny_patterns" env:"SPIDERWEB_TOOLS_EXEC_ENABLE_DENY_PATTERNS"`
	CustomDenyPatterns []string `json:"custom_deny_patterns" env:"SPIDERWEB_TOOLS_EXEC_CUSTOM_DENY_PATTERNS"`
}

type ToolsConfig struct {
	Web    WebToolsConfig    `json:"web"`
	Cron   CronToolsConfig   `json:"cron"`
	Exec   ExecConfig        `json:"exec"`
	Skills SkillsToolsConfig `json:"skills"`
}

type SkillsToolsConfig struct {
	Registries            SkillsRegistriesConfig `json:"registries"`
	MaxConcurrentSearches int                    `json:"max_concurrent_searches" env:"SPIDERWEB_SKILLS_MAX_CONCURRENT_SEARCHES"`
	SearchCache           SearchCacheConfig      `json:"search_cache"`
}

type SearchCacheConfig struct {
	MaxSize    int `json:"max_size"    env:"SPIDERWEB_SKILLS_SEARCH_CACHE_MAX_SIZE"`
	TTLSeconds int `json:"ttl_seconds" env:"SPIDERWEB_SKILLS_SEARCH_CACHE_TTL_SECONDS"`
}

type SkillsRegistriesConfig struct {
	ClawHub ClawHubRegistryConfig `json:"clawhub"`
}

type ClawHubRegistryConfig struct {
	Enabled         bool   `json:"enabled"           env:"SPIDERWEB_SKILLS_REGISTRIES_CLAWHUB_ENABLED"`
	BaseURL         string `json:"base_url"          env:"SPIDERWEB_SKILLS_REGISTRIES_CLAWHUB_BASE_URL"`
	AuthToken       string `json:"auth_token"        env:"SPIDERWEB_SKILLS_REGISTRIES_CLAWHUB_AUTH_TOKEN"`
	SearchPath      string `json:"search_path"       env:"SPIDERWEB_SKILLS_REGISTRIES_CLAWHUB_SEARCH_PATH"`
	SkillsPath      string `json:"skills_path"       env:"SPIDERWEB_SKILLS_REGISTRIES_CLAWHUB_SKILLS_PATH"`
	DownloadPath    string `json:"download_path"     env:"SPIDERWEB_SKILLS_REGISTRIES_CLAWHUB_DOWNLOAD_PATH"`
	Timeout         int    `json:"timeout"           env:"SPIDERWEB_SKILLS_REGISTRIES_CLAWHUB_TIMEOUT"`
	MaxZipSize      int    `json:"max_zip_size"      env:"SPIDERWEB_SKILLS_REGISTRIES_CLAWHUB_MAX_ZIP_SIZE"`
	MaxResponseSize int    `json:"max_response_size" env:"SPIDERWEB_SKILLS_REGISTRIES_CLAWHUB_MAX_RESPONSE_SIZE"`
}

func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	} else {
		// Pre-scan the JSON to check how many model_list entries the user provided.
		// Go's JSON decoder reuses existing slice backing-array elements rather than
		// zero-initializing them, so fields absent from the user's JSON (e.g. api_base)
		// would silently inherit values from the DefaultConfig template at the same
		// index position. We only reset cfg.ModelList when the user actually provides
		// entries; when count is 0 we keep DefaultConfig's built-in list as fallback.
		var tmp Config
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
		if len(tmp.ModelList) > 0 {
			cfg.ModelList = nil
		}

		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	if err := loadRuntimeEnv(path); err != nil {
		return nil, err
	}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	// Auto-migrate: if only legacy providers config exists, convert to model_list
	if len(cfg.ModelList) == 0 && cfg.HasProvidersConfig() {
		cfg.ModelList = ConvertProvidersToModelList(cfg)
	}

	// Validate model_list for uniqueness and required fields
	if err := cfg.ValidateModelList(); err != nil {
		return nil, err
	}

	// Apply observer config defaults
	cfg.Observer = cfg.Observer.withDefaults()

	return cfg, nil
}

func loadRuntimeEnv(configPath string) error {
	for _, candidate := range runtimeEnvCandidates(configPath) {
		if candidate == "" {
			continue
		}
		if err := loadEnvFile(candidate); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("load runtime env %q: %w", candidate, err)
		}
	}
	return nil
}

func runtimeEnvCandidates(configPath string) []string {
	candidates := []string{}
	seen := map[string]struct{}{}

	add := func(path string) {
		if path == "" {
			return
		}
		path = expandHome(path)
		path = filepath.Clean(path)
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		candidates = append(candidates, path)
	}

	add(os.Getenv("SPIDERWEB_RUNTIME_ENV"))

	if configPath != "" {
		add(filepath.Join(filepath.Dir(configPath), "runtime.env"))
	}

	if home, err := os.UserHomeDir(); err == nil {
		add(filepath.Join(home, ".spiderweb", "runtime.env"))
	}

	return candidates
}

func loadEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}

		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}

	return scanner.Err()
}

func SaveConfig(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

func (c *Config) WorkspacePath() string {
	return expandHome(c.Agents.Defaults.Workspace)
}

func (c *Config) GetAPIKey() string {
	if c.Providers.OpenRouter.APIKey != "" {
		return c.Providers.OpenRouter.APIKey
	}
	if c.Providers.Anthropic.APIKey != "" {
		return c.Providers.Anthropic.APIKey
	}
	if c.Providers.OpenAI.APIKey != "" {
		return c.Providers.OpenAI.APIKey
	}
	if c.Providers.Gemini.APIKey != "" {
		return c.Providers.Gemini.APIKey
	}
	if c.Providers.Zhipu.APIKey != "" {
		return c.Providers.Zhipu.APIKey
	}
	if c.Providers.Groq.APIKey != "" {
		return c.Providers.Groq.APIKey
	}
	if c.Providers.VLLM.APIKey != "" {
		return c.Providers.VLLM.APIKey
	}
	if c.Providers.ShengSuanYun.APIKey != "" {
		return c.Providers.ShengSuanYun.APIKey
	}
	if c.Providers.Cerebras.APIKey != "" {
		return c.Providers.Cerebras.APIKey
	}
	return ""
}

func (c *Config) GetAPIBase() string {
	if c.Providers.OpenRouter.APIKey != "" {
		if c.Providers.OpenRouter.APIBase != "" {
			return c.Providers.OpenRouter.APIBase
		}
		return "https://openrouter.ai/api/v1"
	}
	if c.Providers.Zhipu.APIKey != "" {
		return c.Providers.Zhipu.APIBase
	}
	if c.Providers.VLLM.APIKey != "" && c.Providers.VLLM.APIBase != "" {
		return c.Providers.VLLM.APIBase
	}
	return ""
}

func expandHome(path string) string {
	if path == "" {
		return path
	}
	if path[0] == '~' {
		home, _ := os.UserHomeDir()
		if len(path) > 1 && path[1] == '/' {
			return home + path[1:]
		}
		return home
	}
	return path
}

// GetModelConfig returns the ModelConfig for the given model name.
// If multiple configs exist with the same model_name, it uses round-robin
// selection for load balancing. Returns an error if the model is not found.
func (c *Config) GetModelConfig(modelName string) (*ModelConfig, error) {
	matches := c.findMatches(modelName)
	if len(matches) == 0 {
		return nil, fmt.Errorf("model %q not found in model_list or providers", modelName)
	}
	if len(matches) == 1 {
		return &matches[0], nil
	}

	// Multiple configs - use round-robin for load balancing
	idx := rrCounter.Add(1) % uint64(len(matches))
	return &matches[idx], nil
}

// findMatches finds all ModelConfig entries with the given model_name.
func (c *Config) findMatches(modelName string) []ModelConfig {
	var matches []ModelConfig
	for i := range c.ModelList {
		if c.ModelList[i].ModelName == modelName {
			matches = append(matches, c.ModelList[i])
		}
	}
	return matches
}

// HasProvidersConfig checks if any provider in the old providers config has configuration.
func (c *Config) HasProvidersConfig() bool {
	v := c.Providers
	return v.Anthropic.APIKey != "" || v.Anthropic.APIBase != "" ||
		v.OpenAI.APIKey != "" || v.OpenAI.APIBase != "" ||
		v.OpenRouter.APIKey != "" || v.OpenRouter.APIBase != "" ||
		v.Groq.APIKey != "" || v.Groq.APIBase != "" ||
		v.Zhipu.APIKey != "" || v.Zhipu.APIBase != "" ||
		v.VLLM.APIKey != "" || v.VLLM.APIBase != "" ||
		v.Gemini.APIKey != "" || v.Gemini.APIBase != "" ||
		v.Nvidia.APIKey != "" || v.Nvidia.APIBase != "" ||
		v.Ollama.APIKey != "" || v.Ollama.APIBase != "" ||
		v.Moonshot.APIKey != "" || v.Moonshot.APIBase != "" ||
		v.ShengSuanYun.APIKey != "" || v.ShengSuanYun.APIBase != "" ||
		v.DeepSeek.APIKey != "" || v.DeepSeek.APIBase != "" ||
		v.Cerebras.APIKey != "" || v.Cerebras.APIBase != "" ||
		v.VolcEngine.APIKey != "" || v.VolcEngine.APIBase != "" ||
		v.GitHubCopilot.APIKey != "" || v.GitHubCopilot.APIBase != "" ||
		v.Antigravity.APIKey != "" || v.Antigravity.APIBase != "" ||
		v.Qwen.APIKey != "" || v.Qwen.APIBase != "" ||
		v.Mistral.APIKey != "" || v.Mistral.APIBase != ""
}

// ValidateModelList validates all ModelConfig entries in the model_list.
// It checks that each model config is valid.
// Note: Multiple entries with the same model_name are allowed for load balancing.
func (c *Config) ValidateModelList() error {
	for i := range c.ModelList {
		if err := c.ModelList[i].Validate(); err != nil {
			return fmt.Errorf("model_list[%d]: %w", i, err)
		}
	}
	return nil
}
