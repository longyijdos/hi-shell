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
)

func TestGenerateRequiresPromptFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"generate", "list", "go", "files"}, strings.NewReader(""), &stdout, &stderr, "test")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "requires --prompt") {
		t.Fatalf("stderr = %q, want prompt error", stderr.String())
	}
	if !strings.Contains(stderr.String(), "hi-shell generate") {
		t.Fatalf("stderr = %q, want hi-shell generate example", stderr.String())
	}
}

func TestGenerateRejectsInvalidArgumentsWithoutCreatingConfig(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantStderr string
	}{
		{
			name:       "extra args",
			args:       []string{"generate", "--prompt", "list go files", "extra"},
			wantStderr: `unexpected argument "extra"`,
		},
		{
			name:       "blank prompt",
			args:       []string{"generate", "--prompt", "   "},
			wantStderr: "requires --prompt",
		},
		{
			name:       "unsupported format",
			args:       []string{"generate", "--prompt", "list go files", "--format", "xml"},
			wantStderr: `unsupported format "xml"`,
		},
		{
			name:       "session flag",
			args:       []string{"generate", "--prompt", "list go files", "--session-json", `{"initial_prompt":"list files"}`},
			wantStderr: "flag provided but not defined: -session-json",
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
				t.Fatalf("invalid generate command created config directory: %v", err)
			}
		})
	}
}

func TestGenerateJSONOmitsExplanation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %q, want /chat/completions", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"find . -name '*.go'"}}]}`))
	}))
	defer server.Close()

	hiHome := t.TempDir()
	t.Setenv(config.HomeEnv, hiHome)

	cfg := config.Default()
	cfg.OpenAI.BaseURL = server.URL
	cfg.OpenAI.APIKeyEnv = ""
	cfg.Context = config.ContextConfig{}
	if err := config.SaveFile(filepath.Join(hiHome, config.ConfigFileName), cfg); err != nil {
		t.Fatalf("SaveFile() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"generate", "--prompt", "list go files", "--format", "json"}, strings.NewReader(""), &stdout, &stderr, "test")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := response["explanation"]; ok {
		t.Fatalf("response includes explanation: %s", stdout.String())
	}
	if response["command"] != "find . -name '*.go'" {
		t.Fatalf("command = %#v", response["command"])
	}
	if response["risk"] != "low" {
		t.Fatalf("risk = %#v", response["risk"])
	}
	if response["warning"] != "" {
		t.Fatalf("warning = %#v", response["warning"])
	}
}
