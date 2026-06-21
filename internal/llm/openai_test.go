package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIProviderOmitsUnsetTemperature(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"pwd"}}]}`))
	}))
	defer server.Close()

	provider := OpenAIProvider{BaseURL: server.URL}
	if _, err := provider.Generate(context.Background(), Request{
		Model:    "test-model",
		Messages: []Message{{Role: "user", Content: "where am I"}},
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if _, exists := payload["temperature"]; exists {
		t.Fatalf("temperature = %#v, want omitted", payload["temperature"])
	}
}

func TestOpenAIProviderSendsConfiguredTemperature(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"pwd"}}]}`))
	}))
	defer server.Close()

	provider := OpenAIProvider{BaseURL: server.URL, Temperature: float64Pointer(0.3)}
	if _, err := provider.Generate(context.Background(), Request{
		Model:    "test-model",
		Messages: []Message{{Role: "user", Content: "where am I"}},
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if payload["temperature"] != 0.3 {
		t.Fatalf("temperature = %#v, want 0.3", payload["temperature"])
	}
}
