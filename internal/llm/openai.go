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

type OpenAIProvider struct {
	BaseURL   string
	APIKeyEnv string
	Client    *http.Client
}

func (p OpenAIProvider) Generate(ctx context.Context, req Request) (Completion, error) {
	if strings.TrimSpace(req.Model) == "" {
		return Completion{}, fmt.Errorf("openai model is required")
	}

	baseURL := strings.TrimRight(p.BaseURL, "/")
	if baseURL == "" {
		return Completion{}, fmt.Errorf("openai base_url is required")
	}

	apiKey := ""
	if p.APIKeyEnv != "" {
		apiKey = os.Getenv(p.APIKeyEnv)
		if apiKey == "" && strings.Contains(baseURL, "api.openai.com") {
			return Completion{}, fmt.Errorf("%s is not set", p.APIKeyEnv)
		}
	}

	payload := openAIRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		Temperature: 0.1,
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
		return Completion{}, fmt.Errorf("openai-compatible API returned %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	var parsed openAIResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return Completion{}, err
	}
	if len(parsed.Choices) == 0 {
		return Completion{}, fmt.Errorf("openai-compatible API returned no choices")
	}

	return Completion{
		Command: parsed.Choices[0].Message.Content,
	}, nil
}

type openAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
}

type openAIResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}
