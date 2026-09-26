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
		BaseURL:   server.URL,
		APIKeyEnv: "ANTHROPIC_API_KEY",
		Thinking:  "adaptive",
		Effort:    "medium",
		MaxTokens: 64,
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
	if !ok || thinking["type"] != "adaptive" {
		t.Fatalf("thinking = %#v", payload["thinking"])
	}
	if _, exists := thinking["budget_tokens"]; exists {
		t.Fatalf("budget_tokens = %#v, want omitted", thinking["budget_tokens"])
	}
	outputConfig, ok := payload["output_config"].(map[string]any)
	if !ok || outputConfig["effort"] != "medium" {
		t.Fatalf("output_config = %#v, want medium effort", payload["output_config"])
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

func TestClaudeProviderSendsDisabledThinking(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"pwd"}]}`))
	}))
	defer server.Close()

	provider := ClaudeProvider{BaseURL: server.URL}
	if _, err := provider.Generate(context.Background(), Request{
		Model:    "claude-test",
		Messages: []Message{{Role: "user", Content: "where am I"}},
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	thinking, ok := payload["thinking"].(map[string]any)
	if !ok || thinking["type"] != "disabled" {
		t.Fatalf("thinking = %#v, want disabled", payload["thinking"])
	}
	if _, exists := payload["output_config"]; exists {
		t.Fatalf("output_config = %#v, want omitted", payload["output_config"])
	}
}

func TestClaudeProviderRejectsInvalidThinking(t *testing.T) {
	provider := ClaudeProvider{BaseURL: "https://example.test", Thinking: "enabled"}
	_, err := provider.Generate(context.Background(), Request{
		Model:    "claude-test",
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	if err == nil || err.Error() != "claude thinking must be disabled or adaptive" {
		t.Fatalf("Generate() error = %v", err)
	}
}
