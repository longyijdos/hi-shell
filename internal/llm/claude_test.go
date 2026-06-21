package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClaudeProviderSendsMessagesRequest(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("path = %q, want /v1/messages", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Fatalf("x-api-key = %q, want test-key", got)
		}
		if got := r.Header.Get("anthropic-version"); got != claudeAPIVersion {
			t.Fatalf("anthropic-version = %q, want %q", got, claudeAPIVersion)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"thinking","thinking":"..."},{"type":"text","text":"git status"},{"type":"text","text":" --short"}]}`))
	}))
	defer server.Close()

	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	provider := ClaudeProvider{
		BaseURL:              server.URL,
		APIKeyEnv:            "ANTHROPIC_API_KEY",
		Thinking:             "enabled",
		ThinkingBudgetTokens: 32,
		MaxTokens:            64,
	}
	completion, err := provider.Generate(context.Background(), Request{
		Model: "claude-test",
		Messages: []Message{
			{Role: "system", Content: "Return a command."},
			{Role: "user", Content: "show status"},
		},
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if completion.Command != "git status --short" {
		t.Fatalf("Command = %q", completion.Command)
	}
	if payload["model"] != "claude-test" {
		t.Fatalf("model = %#v", payload["model"])
	}
	if payload["max_tokens"] != float64(64) {
		t.Fatalf("max_tokens = %#v", payload["max_tokens"])
	}
	thinking, ok := payload["thinking"].(map[string]any)
	if !ok || thinking["type"] != "enabled" || thinking["budget_tokens"] != float64(32) {
		t.Fatalf("thinking = %#v", payload["thinking"])
	}
	if payload["system"] != "Return a command." {
		t.Fatalf("system = %#v", payload["system"])
	}
	messages, ok := payload["messages"].([]any)
	if !ok || len(messages) != 1 {
		t.Fatalf("messages = %#v", payload["messages"])
	}
	message := messages[0].(map[string]any)
	if message["role"] != "user" || message["content"] != "show status" {
		t.Fatalf("message = %#v", message)
	}
}

func TestClaudeProviderOmitsThinkingWhenDisabled(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"pwd"}]}`))
	}))
	defer server.Close()

	provider := ClaudeProvider{BaseURL: server.URL, Temperature: float64Pointer(0.2)}
	if _, err := provider.Generate(context.Background(), Request{
		Model:    "claude-test",
		Messages: []Message{{Role: "user", Content: "where am I"}},
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if _, exists := payload["thinking"]; exists {
		t.Fatalf("thinking = %#v, want omitted", payload["thinking"])
	}
	if payload["temperature"] != 0.2 {
		t.Fatalf("temperature = %#v, want 0.2", payload["temperature"])
	}
}

func TestClaudeProviderOmitsTemperatureWhenThinkingEnabled(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"pwd"}]}`))
	}))
	defer server.Close()

	provider := ClaudeProvider{
		BaseURL:              server.URL,
		Thinking:             "enabled",
		ThinkingBudgetTokens: 32,
		MaxTokens:            64,
		Temperature:          float64Pointer(0.2),
	}
	if _, err := provider.Generate(context.Background(), Request{
		Model:    "claude-test",
		Messages: []Message{{Role: "user", Content: "where am I"}},
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if _, exists := payload["temperature"]; exists {
		t.Fatalf("temperature = %#v, want omitted when thinking is enabled", payload["temperature"])
	}
}

func TestClaudeProviderRejectsInvalidThinking(t *testing.T) {
	provider := ClaudeProvider{BaseURL: "https://example.test", Thinking: "always"}
	_, err := provider.Generate(context.Background(), Request{
		Model:    "claude-test",
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	if err == nil || err.Error() != "claude thinking must be disabled, enabled, or adaptive" {
		t.Fatalf("Generate() error = %v", err)
	}
}

func TestClaudeProviderRequiresBudgetBelowMaxTokensWhenEnabled(t *testing.T) {
	provider := ClaudeProvider{
		BaseURL:              "https://example.test",
		Thinking:             "enabled",
		ThinkingBudgetTokens: 256,
		MaxTokens:            256,
	}
	_, err := provider.Generate(context.Background(), Request{
		Model:    "claude-test",
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	if err == nil || err.Error() != "claude thinking_budget_tokens must be less than max_tokens when thinking is enabled" {
		t.Fatalf("Generate() error = %v", err)
	}
}
