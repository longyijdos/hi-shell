package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const claudeAPIVersion = "2023-06-01"

type ClaudeProvider struct {
	BaseURL              string
	APIKeyEnv            string
	Thinking             string
	ThinkingBudgetTokens int
	MaxTokens            int
	Client               *http.Client
}

func (p ClaudeProvider) Generate(ctx context.Context, req Request) (Completion, error) {
	if strings.TrimSpace(req.Model) == "" {
		return Completion{}, fmt.Errorf("claude model is required")
	}

	baseURL := strings.TrimRight(p.BaseURL, "/")
	if baseURL == "" {
		return Completion{}, fmt.Errorf("claude base_url is required")
	}

	apiKey := ""
	if p.APIKeyEnv != "" {
		apiKey = os.Getenv(p.APIKeyEnv)
		if apiKey == "" && strings.Contains(baseURL, "api.anthropic.com") {
			return Completion{}, fmt.Errorf("%s is not set", p.APIKeyEnv)
		}
	}

	maxTokens := p.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 256
	}

	payload := claudeRequest{
		Model:     req.Model,
		MaxTokens: maxTokens,
		Messages:  make([]Message, 0, len(req.Messages)),
	}
	for _, message := range req.Messages {
		if message.Role == "system" {
			if payload.System == "" {
				payload.System = message.Content
			} else {
				payload.System += "\n\n" + message.Content
			}
			continue
		}
		payload.Messages = append(payload.Messages, message)
	}
	if len(payload.Messages) == 0 {
		return Completion{}, fmt.Errorf("claude requires at least one non-system message")
	}

	thinking := strings.ToLower(strings.TrimSpace(p.Thinking))
	if thinking == "" {
		thinking = "disabled"
	}
	switch thinking {
	case "disabled":
	case "adaptive":
		payload.Thinking = &claudeThinking{Type: thinking}
	case "enabled":
		if p.ThinkingBudgetTokens <= 0 {
			return Completion{}, fmt.Errorf("claude thinking_budget_tokens must be a positive integer when thinking is enabled")
		}
		if p.ThinkingBudgetTokens >= maxTokens {
			return Completion{}, fmt.Errorf("claude thinking_budget_tokens must be less than max_tokens when thinking is enabled")
		}
		payload.Thinking = &claudeThinking{Type: thinking, BudgetTokens: p.ThinkingBudgetTokens}
	default:
		return Completion{}, fmt.Errorf("claude thinking must be disabled, enabled, or adaptive")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return Completion{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return Completion{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("anthropic-version", claudeAPIVersion)
	if apiKey != "" {
		httpReq.Header.Set("x-api-key", apiKey)
	}

	client := p.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return Completion{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Completion{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Completion{}, fmt.Errorf("claude API returned %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	var parsed claudeResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return Completion{}, err
	}
	var text strings.Builder
	for _, block := range parsed.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}
	if text.Len() == 0 {
		return Completion{}, fmt.Errorf("claude API returned no text content")
	}

	return Completion{Command: text.String()}, nil
}

type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    string          `json:"system,omitempty"`
	Messages  []Message       `json:"messages"`
	Thinking  *claudeThinking `json:"thinking,omitempty"`
}

type claudeThinking struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens,omitempty"`
}

type claudeResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}
