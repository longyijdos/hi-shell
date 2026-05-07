package llm

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMessageJSONUsesProviderFieldNames(t *testing.T) {
	data, err := json.Marshal(Message{Role: "system", Content: "compose commands"})
	if err != nil {
		t.Fatalf("json.Marshal(Message) error = %v", err)
	}

	encoded := string(data)
	if !strings.Contains(encoded, `"role":"system"`) {
		t.Fatalf("encoded message = %s, want lowercase role field", encoded)
	}
	if !strings.Contains(encoded, `"content":"compose commands"`) {
		t.Fatalf("encoded message = %s, want lowercase content field", encoded)
	}
	if strings.Contains(encoded, `"Role"`) || strings.Contains(encoded, `"Content"`) {
		t.Fatalf("encoded message = %s, contains Go field names", encoded)
	}
}

func TestNormalizeCommand(t *testing.T) {
	tests := map[string]string{
		"find . -type f":                   "find . -type f",
		"$ ls -la":                         "ls -la",
		"```zsh\nfind . -maxdepth 1\n```":  "find . -maxdepth 1",
		`{"command":"git status --short"}`: "git status --short",
	}

	for input, want := range tests {
		if got := NormalizeCommand(input); got != want {
			t.Fatalf("NormalizeCommand(%q) = %q, want %q", input, got, want)
		}
	}
}
