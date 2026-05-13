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

type DeepSeekProvider struct {
	BaseURL   string
	APIKeyEnv string
	Thinking  string
	MaxTokens int
	Client    *http.Client
}

func (p DeepSeekProvider) Generate(ctx context.Context, req Request) (Completion, error) {
	if strings.TrimSpace(req.Model) == "" {
		return Completion{}, fmt.Errorf("deepseek model is required")
	}

	baseURL := strings.TrimRight(p.BaseURL, "/")
	if baseURL == "" {
		return Completion{}, fmt.Errorf("deepseek base_url is required")
	}

	apiKey := ""
	if p.APIKeyEnv != "" {
		apiKey = os.Getenv(p.APIKeyEnv)
		if apiKey == "" && strings.Contains(baseURL, "deepseek.com") {
			return Completion{}, fmt.Errorf("%s is not set", p.APIKeyEnv)
		}
	}

	maxTokens := p.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 256
	}

	thinking := strings.ToLower(strings.TrimSpace(p.Thinking))
	if thinking == "" {
		thinking = "disabled"
	}
	if thinking != "enabled" && thinking != "disabled" {
		return Completion{}, fmt.Errorf("deepseek thinking must be enabled or disabled")
	}

	payload := deepSeekRequest{
		Model:     req.Model,
		Messages:  req.Messages,
		MaxTokens: maxTokens,
		Thinking:  &deepSeekThinking{Type: thinking},
	}
	if thinking == "disabled" {
		temperature := 0.1
		payload.Temperature = &temperature
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return Completion{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Completion{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
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
		return Completion{}, fmt.Errorf("deepseek API returned %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	var parsed deepSeekResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return Completion{}, err
	}
	if len(parsed.Choices) == 0 {
		return Completion{}, fmt.Errorf("deepseek API returned no choices")
	}

	return Completion{
		Command: parsed.Choices[0].Message.Content,
	}, nil
}

type deepSeekRequest struct {
	Model       string            `json:"model"`
	Messages    []Message         `json:"messages"`
	Temperature *float64          `json:"temperature,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Thinking    *deepSeekThinking `json:"thinking,omitempty"`
}

type deepSeekThinking struct {
	Type string `json:"type"`
}

type deepSeekResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}
