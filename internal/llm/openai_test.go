package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIProviderSendsChatRequest(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"pwd"}}]}`))
	}))
	defer server.Close()

	provider := OpenAIProvider{BaseURL: server.URL}
	completion, err := provider.Generate(context.Background(), Request{
		Model:    "test-model",
		Messages: []Message{{Role: "user", Content: "where am I"}},
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if completion.Command != "pwd" || payload["model"] != "test-model" {
		t.Fatalf("completion = %#v, payload = %#v", completion, payload)
	}
	if _, exists := payload["temperature"]; exists {
		t.Fatalf("temperature = %#v, want omitted", payload["temperature"])
	}
}
