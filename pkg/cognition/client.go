package cognition

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/JustSebNL/Spiderweb/pkg/config"
)

const (
	defaultBaseURL = "http://127.0.0.1:8000/v1"
	defaultModel   = "tencent/Youtu-LLM-2B"
)

type Client struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Temperature float64       `json:"temperature,omitempty"`
	Messages    []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func NewClient(baseURL, apiKey, model string, timeout time.Duration) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	if strings.TrimSpace(model) == "" {
		model = defaultModel
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func NewClientFromConfig(cfg *config.Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	cc := cfg.Intake.CheapCognition
	if !cc.Enabled {
		return nil, fmt.Errorf("cheap cognition is disabled")
	}
	return NewClient(cc.BaseURL, cc.APIKey, cc.Model, time.Duration(cc.TimeoutSeconds)*time.Second), nil
}

func (c *Client) ClassifyEvent(ctx context.Context, event Event) (*ClassificationResult, error) {
	payload, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal event: %w", err)
	}

	content, err := c.chat(ctx, []chatMessage{
		{
			Role: "system",
			Content: "You are Spiderweb's cheap intake classifier. Return strict JSON with fields: priority, category, escalation_needed, one_line_summary.",
		},
		{
			Role:    "user",
			Content: string(payload),
		},
	})
	if err != nil {
		return nil, err
	}

	result, err := parseClassificationResult(content)
	if err != nil {
		return nil, err
	}
	result.RawContent = content
	return result, nil
}

func (c *Client) SummarizeText(ctx context.Context, input string) (string, error) {
	return c.chat(ctx, []chatMessage{
		{
			Role: "system",
			Content: "Summarize the input into a short operational summary for Spiderweb intake. Keep it concise and factual.",
		},
		{
			Role:    "user",
			Content: input,
		},
	})
}

func (c *Client) chat(ctx context.Context, messages []chatMessage) (string, error) {
	requestBody := chatCompletionRequest{
		Model:       c.model,
		Temperature: 0.2,
		Messages:    messages,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cheap cognition request failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var decoded chatCompletionResponse
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return "", fmt.Errorf("cheap cognition response contained no choices")
	}

	return strings.TrimSpace(decoded.Choices[0].Message.Content), nil
}

func parseClassificationResult(content string) (*ClassificationResult, error) {
	jsonBody, err := extractJSONObject(content)
	if err != nil {
		return nil, err
	}

	var result ClassificationResult
	if err := json.Unmarshal([]byte(jsonBody), &result); err != nil {
		return nil, fmt.Errorf("decode classification result: %w", err)
	}
	return &result, nil
}

func extractJSONObject(content string) (string, error) {
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "```") {
		trimmed = strings.TrimPrefix(trimmed, "```json")
		trimmed = strings.TrimPrefix(trimmed, "```")
		trimmed = strings.TrimSuffix(trimmed, "```")
		trimmed = strings.TrimSpace(trimmed)
	}

	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start == -1 || end == -1 || end < start {
		return "", fmt.Errorf("classification result did not contain a JSON object")
	}
	return trimmed[start : end+1], nil
}
