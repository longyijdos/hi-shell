package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/longyijdos/hi-shell/internal/config"
	"github.com/longyijdos/hi-shell/internal/llm"
	promptpkg "github.com/longyijdos/hi-shell/internal/prompt"
)

func TestReadAskSessionSupportsInlineStdinAndFile(t *testing.T) {
	raw := `{
		"initial_prompt": " list large files ",
		"turns": [
			{
				"command": " find . -type f -size +100M ",
				"risk": " safe ",
				"warning": " ",
				"question": " will this modify files? ",
				"answer": " no "
			}
		]
	}`

	inline, err := readAskSession(raw, nil)
	if err != nil {
		t.Fatalf("readAskSession(inline) error = %v", err)
	}
	if inline.InitialPrompt != "list large files" {
		t.Fatalf("InitialPrompt = %q", inline.InitialPrompt)
	}
	if inline.Turns[0].Command != "find . -type f -size +100M" {
		t.Fatalf("Command = %q", inline.Turns[0].Command)
	}
	if inline.Turns[0].Warning != "" {
		t.Fatalf("Warning = %q, want empty after trim", inline.Turns[0].Warning)
	}
	fromStdin, err := readAskSession("-", strings.NewReader(raw))
	if err != nil {
		t.Fatalf("readAskSession(stdin) error = %v", err)
	}
	if fromStdin.InitialPrompt != inline.InitialPrompt {
		t.Fatalf("stdin InitialPrompt = %q, want %q", fromStdin.InitialPrompt, inline.InitialPrompt)
	}

	path := filepath.Join(t.TempDir(), "session.json")
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("write session fixture: %v", err)
	}
	fromFile, err := readAskSession("@"+path, nil)
	if err != nil {
		t.Fatalf("readAskSession(file) error = %v", err)
	}
	if fromFile.Turns[0].Question != inline.Turns[0].Question {
		t.Fatalf("file question = %q, want %q", fromFile.Turns[0].Question, inline.Turns[0].Question)
	}
}

func TestParseAskSessionValidation(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr string
	}{
		{
			name:    "missing initial prompt",
			raw:     `{}`,
			wantErr: "initial_prompt is required",
		},
		{
			name:    "unknown field",
			raw:     `{"initial_prompt":"list files","unknown":true}`,
			wantErr: "unknown field",
		},
		{
			name:    "no turns",
			raw:     `{"initial_prompt":"list files"}`,
			wantErr: "turns must contain at least one command with a question",
		},
		{
			name:    "latest turn needs question",
			raw:     `{"initial_prompt":"list files","turns":[{"command":"ls"}]}`,
			wantErr: "turns[0].question is required",
		},
		{
			name:    "empty command",
			raw:     `{"initial_prompt":"list files","turns":[{"command":"  ","question":"what does this do"}]}`,
			wantErr: "turns[0].command is required",
		},
		{
			name:    "trailing data",
			raw:     `{"initial_prompt":"list files"} {}`,
			wantErr: "trailing data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseAskSession([]byte(tt.raw))
			if err == nil {
				t.Fatalf("parseAskSession() error = nil, want %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestParseAskSessionKeepsRecentTurns(t *testing.T) {
	session := promptpkg.AskSession{
		InitialPrompt: "list files",
	}
	for i := 0; i < maxSessionTurns+2; i++ {
		session.Turns = append(session.Turns, promptpkg.AskTurn{
			Command:  "ls",
			Question: "what does this do?",
		})
	}

	data, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	parsed, err := parseAskSession(data)
	if err != nil {
		t.Fatalf("parseAskSession() error = %v", err)
	}
	if len(parsed.Turns) != maxSessionTurns {
		t.Fatalf("len(Turns) = %d, want %d", len(parsed.Turns), maxSessionTurns)
	}
}

func TestAskSessionJSONStdinValidationDoesNotCreateConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("HI_SHELL_HOME", "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"ask", "--session-json", "-"}, strings.NewReader(`{}`), &stdout, &stderr, "test")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2; stderr = %q", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "initial_prompt is required") {
		t.Fatalf("stderr = %q, want initial_prompt error", stderr.String())
	}
	_, err := os.Stat(filepath.Join(home, ".hi-shell"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid session created config directory: %v", err)
	}
}

func TestAskRejectsInvalidArgumentsWithoutCreatingConfig(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantStderr string
	}{
		{
			name:       "invalid session json",
			args:       []string{"ask", "--session-json", `{}`},
			wantStderr: "initial_prompt is required",
		},
		{
			name:       "missing session json",
			args:       []string{"ask"},
			wantStderr: "requires --session-json",
		},
		{
			name:       "unsupported format",
			args:       []string{"ask", "--session-json", `{"initial_prompt":"list files"}`, "--format", "xml"},
			wantStderr: `unsupported format "xml"`,
		},
		{
			name:       "extra args",
			args:       []string{"ask", "--session-json", `{"initial_prompt":"list files"}`, "extra"},
			wantStderr: `unexpected argument "extra"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("HI_SHELL_HOME", "")

			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(tt.args, strings.NewReader(""), &stdout, &stderr, "test")
			if code != 2 {
				t.Fatalf("exit code = %d, want 2; stderr = %q", code, stderr.String())
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			if !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tt.wantStderr)
			}
			_, err := os.Stat(filepath.Join(home, ".hi-shell"))
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("invalid ask command created config directory: %v", err)
			}
		})
	}
}

func TestAskSessionJSONUsesAskPrompt(t *testing.T) {
	var payload struct {
		Messages []llm.Message `json:"messages"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %q, want /chat/completions", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"It only reads matching file metadata and prints paths."}}]}`))
	}))
	defer server.Close()

	hiHome := t.TempDir()
	t.Setenv(config.HomeEnv, hiHome)

	cfg := config.Default()
	cfg.OpenAI.Model = "test-model"
	cfg.OpenAI.BaseURL = server.URL
	cfg.OpenAI.APIKeyEnv = ""
	cfg.Context = config.ContextConfig{}
	if err := config.SaveFile(filepath.Join(hiHome, config.ConfigFileName), cfg); err != nil {
		t.Fatalf("SaveFile() error = %v", err)
	}

	rawSession := `{
		"initial_prompt": "list large files",
		"turns": [
			{
				"command": "find . -type f -size +100M",
				"risk": "safe",
				"question": "will this modify files?"
			}
		]
	}`

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"ask", "--session-json", "-", "--format", "json"}, strings.NewReader(rawSession), &stdout, &stderr, "test")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var response askResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Answer != "It only reads matching file metadata and prints paths." {
		t.Fatalf("Answer = %q", response.Answer)
	}
	if len(payload.Messages) != 2 {
		t.Fatalf("messages = %#v, want system and user messages", payload.Messages)
	}
	userPrompt := payload.Messages[1].Content
	for _, fragment := range []string{
		"Command question session:",
		"Initial user request:\nlist large files",
		"Command: find . -type f -size +100M",
		"User question: will this modify files?",
	} {
		if !strings.Contains(userPrompt, fragment) {
			t.Fatalf("user prompt missing %q:\n%s", fragment, userPrompt)
		}
	}
}
