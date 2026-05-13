package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeepSeekProviderSendsFastCommandOptions(t *testing.T) {
	var payload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %q, want /chat/completions", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"find . -name '*.go'"}}]}`))
	}))
	defer server.Close()

	provider := DeepSeekProvider{
		BaseURL:   server.URL,
		Thinking:  "disabled",
		MaxTokens: 64,
	}
	completion, err := provider.Generate(context.Background(), Request{
		Model: "deepseek-v4-flash",
		Messages: []Message{
			{Role: "system", Content: "Return one command."},
			{Role: "user", Content: "list go files"},
		},
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if completion.Command != "find . -name '*.go'" {
		t.Fatalf("Command = %q", completion.Command)
	}

	thinking, ok := payload["thinking"].(map[string]any)
	if !ok {
		t.Fatalf("thinking payload = %#v", payload["thinking"])
	}
	if thinking["type"] != "disabled" {
		t.Fatalf("thinking.type = %q, want disabled", thinking["type"])
	}
	if payload["max_tokens"] != float64(64) {
		t.Fatalf("max_tokens = %#v, want 64", payload["max_tokens"])
	}
	if payload["temperature"] != 0.1 {
		t.Fatalf("temperature = %#v, want 0.1", payload["temperature"])
	}
	if _, ok := payload["stop"]; ok {
		t.Fatalf("stop = %#v, want omitted", payload["stop"])
	}

	messages, ok := payload["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("messages = %#v", payload["messages"])
	}
	firstMessage, ok := messages[0].(map[string]any)
	if !ok {
		t.Fatalf("first message = %#v", messages[0])
	}
	if _, ok := firstMessage["role"]; !ok {
		t.Fatalf("first message missing lowercase role: %#v", firstMessage)
	}
	if _, ok := firstMessage["content"]; !ok {
		t.Fatalf("first message missing lowercase content: %#v", firstMessage)
	}
}

func TestDeepSeekProviderPreservesMultiLineResponse(t *testing.T) {
	var payload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"Line one.\nLine two."}}]}`))
	}))
	defer server.Close()

	provider := DeepSeekProvider{
		BaseURL:  server.URL,
		Thinking: "disabled",
	}
	completion, err := provider.Generate(context.Background(), Request{
		Model: "deepseek-v4-flash",
		Messages: []Message{
			{Role: "system", Content: "Answer questions."},
			{Role: "user", Content: "explain this command"},
		},
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if completion.Command != "Line one.\nLine two." {
		t.Fatalf("Command = %q", completion.Command)
	}
	if _, ok := payload["stop"]; ok {
		t.Fatalf("stop = %#v, want omitted", payload["stop"])
	}
}
