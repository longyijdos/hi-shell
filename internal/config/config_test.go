package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	cfg := Default()
	cfg.Provider = "deepseek"
	cfg.DeepSeek.Model = "deepseek-v4-pro"
	cfg.History.FetchLimit = 30
	cfg.History.MaxEntries = 16
	cfg.Session.ReviseTurns = 6
	cfg.Session.AskTurns = 5

	if err := SaveFile(path, cfg); err != nil {
		t.Fatalf("SaveFile() error = %v", err)
	}

	loaded, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	if loaded.Provider != "deepseek" {
		t.Fatalf("Provider = %q, want deepseek", loaded.Provider)
	}
	if loaded.DeepSeek.Model != "deepseek-v4-pro" {
		t.Fatalf("DeepSeek.Model = %q", loaded.DeepSeek.Model)
	}
	if loaded.History.FetchLimit != 30 {
		t.Fatalf("History.FetchLimit = %d", loaded.History.FetchLimit)
	}
	if loaded.History.MaxEntries != 16 {
		t.Fatalf("History.MaxEntries = %d", loaded.History.MaxEntries)
	}
	if loaded.Session.ReviseTurns != 6 {
		t.Fatalf("Session.ReviseTurns = %d", loaded.Session.ReviseTurns)
	}
	if loaded.Session.AskTurns != 5 {
		t.Fatalf("Session.AskTurns = %d", loaded.Session.AskTurns)
	}
}

func TestLoadFileMissingUsesDefaults(t *testing.T) {
	loaded, err := LoadFile(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	if loaded.Provider != Default().Provider {
		t.Fatalf("Provider = %q, want %q", loaded.Provider, Default().Provider)
	}
	if loaded.DeepSeek.Thinking != "disabled" {
		t.Fatalf("DeepSeek.Thinking = %q, want disabled", loaded.DeepSeek.Thinking)
	}
	if loaded.DeepSeek.ReasoningEffort != "" || loaded.Claude.Effort != "" {
		t.Fatalf("optional efforts = %q, %q, want unset", loaded.DeepSeek.ReasoningEffort, loaded.Claude.Effort)
	}
	if loaded.History.FetchLimit != 20 {
		t.Fatalf("History.FetchLimit = %d, want 20", loaded.History.FetchLimit)
	}
	if loaded.History.MaxEntries != 12 {
		t.Fatalf("History.MaxEntries = %d, want 12", loaded.History.MaxEntries)
	}
	if loaded.History.MaxCommandChars != 240 {
		t.Fatalf("History.MaxCommandChars = %d, want 240", loaded.History.MaxCommandChars)
	}
	if loaded.History.MaxBytes != 2000 {
		t.Fatalf("History.MaxBytes = %d, want 2000", loaded.History.MaxBytes)
	}
	if loaded.Session.ReviseTurns != 8 {
		t.Fatalf("Session.ReviseTurns = %d, want 8", loaded.Session.ReviseTurns)
	}
	if loaded.Session.AskTurns != 8 {
		t.Fatalf("Session.AskTurns = %d, want 8", loaded.Session.AskTurns)
	}
	if loaded.Session.MaxFieldChars != 4000 {
		t.Fatalf("Session.MaxFieldChars = %d, want 4000", loaded.Session.MaxFieldChars)
	}
	if loaded.Session.MaxJSONBytes != 64*1024 {
		t.Fatalf("Session.MaxJSONBytes = %d, want %d", loaded.Session.MaxJSONBytes, 64*1024)
	}
}

func TestLoadFileWithLegacyKeybinding(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("provider = \"deepseek\"\n[keybindings]\nprefix = \"^]\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	if loaded.Provider != "deepseek" {
		t.Fatalf("Provider = %q, want deepseek", loaded.Provider)
	}
}

func TestHomeDirDefaultsToHiShellDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv(HomeEnv, "")
	t.Setenv("HOME", home)

	got, err := HomeDir()
	if err != nil {
		t.Fatalf("HomeDir() error = %v", err)
	}

	want := filepath.Join(home, ".hi-shell")
	if got != want {
		t.Fatalf("HomeDir() = %q, want %q", got, want)
	}
}

func TestSetRejectsAPIKey(t *testing.T) {
	cfg := Default()
	err := Set(&cfg, "api_key", "sk-test")
	if !errors.Is(err, ErrSecretNotStored) {
		t.Fatalf("Set(api_key) error = %v, want ErrSecretNotStored", err)
	}
}

func TestSetDeepSeekConfig(t *testing.T) {
	cfg := Default()

	settings := map[string]string{
		"deepseek.base_url":         "https://api.deepseek.com/v1",
		"deepseek.api_key_env":      "OPENAI_API_KEY",
		"deepseek.model":            "deepseek-flash",
		"deepseek.thinking":         "enabled",
		"deepseek.reasoning_effort": "max",
		"deepseek.max_tokens":       "128",
	}
	for key, value := range settings {
		if err := Set(&cfg, key, value); err != nil {
			t.Fatalf("Set(%s) error = %v", key, err)
		}
	}

	if cfg.DeepSeek.APIKeyEnv != "OPENAI_API_KEY" {
		t.Fatalf("DeepSeek.APIKeyEnv = %q", cfg.DeepSeek.APIKeyEnv)
	}
	if cfg.DeepSeek.Thinking != "enabled" {
		t.Fatalf("DeepSeek.Thinking = %q", cfg.DeepSeek.Thinking)
	}
	if cfg.DeepSeek.ReasoningEffort != "max" {
		t.Fatalf("DeepSeek.ReasoningEffort = %q", cfg.DeepSeek.ReasoningEffort)
	}
	if cfg.DeepSeek.MaxTokens != 128 {
		t.Fatalf("DeepSeek.MaxTokens = %d", cfg.DeepSeek.MaxTokens)
	}
	if err := Set(&cfg, "deepseek.reasoning_effort", "unset"); err != nil || cfg.DeepSeek.ReasoningEffort != "" {
		t.Fatalf("unset reasoning_effort: value = %q, error = %v", cfg.DeepSeek.ReasoningEffort, err)
	}
}

func TestSetClaudeConfig(t *testing.T) {
	cfg := Default()
	settings := map[string]string{
		"claude.base_url":    "https://api.anthropic.com",
		"claude.api_key_env": "ANTHROPIC_API_KEY",
		"claude.model":       "claude-sonnet-4-6",
		"claude.max_tokens":  "512",
		"claude.thinking":    "adaptive",
		"claude.effort":      "medium",
	}
	for key, value := range settings {
		if err := Set(&cfg, key, value); err != nil {
			t.Fatalf("Set(%s) error = %v", key, err)
		}
		got, err := Get(cfg, key)
		if err != nil {
			t.Fatalf("Get(%s) error = %v", key, err)
		}
		if got != value {
			t.Fatalf("Get(%s) = %q, want %q", key, got, value)
		}
	}
	if cfg.Claude.MaxTokens != 512 || cfg.Claude.Thinking != "adaptive" || cfg.Claude.Effort != "medium" {
		t.Fatalf("Claude = %#v", cfg.Claude)
	}
	if err := Set(&cfg, "claude.effort", "unset"); err != nil || cfg.Claude.Effort != "" {
		t.Fatalf("unset claude.effort: value = %q, error = %v", cfg.Claude.Effort, err)
	}
}

func TestSetClaudeThinkingRejectsUnknownValue(t *testing.T) {
	cfg := Default()
	if err := Set(&cfg, "claude.thinking", "enabled"); err == nil {
		t.Fatal("Set() error = nil, want validation error")
	}
}

func TestSetOpenAIModel(t *testing.T) {
	cfg := Default()

	if err := Set(&cfg, "openai.model", "gpt-4.1"); err != nil {
		t.Fatalf("Set(openai.model) error = %v", err)
	}
	if cfg.OpenAI.Model != "gpt-4.1" {
		t.Fatalf("OpenAI.Model = %q", cfg.OpenAI.Model)
	}

	got, err := Get(cfg, "openai.model")
	if err != nil {
		t.Fatalf("Get(openai.model) error = %v", err)
	}
	if got != "gpt-4.1" {
		t.Fatalf("Get(openai.model) = %q", got)
	}
}

func TestSetHistoryConfig(t *testing.T) {
	cfg := Default()

	settings := map[string]string{
		"history.fetch_limit":       "30",
		"history.max_entries":       "16",
		"history.max_command_chars": "120",
		"history.max_bytes":         "1000",
	}
	for key, value := range settings {
		if err := Set(&cfg, key, value); err != nil {
			t.Fatalf("Set(%s) error = %v", key, err)
		}
		got, err := Get(cfg, key)
		if err != nil {
			t.Fatalf("Get(%s) error = %v", key, err)
		}
		if got != value {
			t.Fatalf("Get(%s) = %q, want %q", key, got, value)
		}
	}

	if cfg.History.FetchLimit != 30 {
		t.Fatalf("History.FetchLimit = %d", cfg.History.FetchLimit)
	}
	if cfg.History.MaxEntries != 16 {
		t.Fatalf("History.MaxEntries = %d", cfg.History.MaxEntries)
	}
	if cfg.History.MaxCommandChars != 120 {
		t.Fatalf("History.MaxCommandChars = %d", cfg.History.MaxCommandChars)
	}
	if cfg.History.MaxBytes != 1000 {
		t.Fatalf("History.MaxBytes = %d", cfg.History.MaxBytes)
	}
}

func TestSetSessionConfig(t *testing.T) {
	cfg := Default()

	settings := map[string]string{
		"session.revise_turns":    "6",
		"session.ask_turns":       "5",
		"session.max_field_chars": "1000",
		"session.max_json_bytes":  "32000",
	}
	for key, value := range settings {
		if err := Set(&cfg, key, value); err != nil {
			t.Fatalf("Set(%s) error = %v", key, err)
		}
		got, err := Get(cfg, key)
		if err != nil {
			t.Fatalf("Get(%s) error = %v", key, err)
		}
		if got != value {
			t.Fatalf("Get(%s) = %q, want %q", key, got, value)
		}
	}

	if cfg.Session.ReviseTurns != 6 {
		t.Fatalf("Session.ReviseTurns = %d", cfg.Session.ReviseTurns)
	}
	if cfg.Session.AskTurns != 5 {
		t.Fatalf("Session.AskTurns = %d", cfg.Session.AskTurns)
	}
	if cfg.Session.MaxFieldChars != 1000 {
		t.Fatalf("Session.MaxFieldChars = %d", cfg.Session.MaxFieldChars)
	}
	if cfg.Session.MaxJSONBytes != 32000 {
		t.Fatalf("Session.MaxJSONBytes = %d", cfg.Session.MaxJSONBytes)
	}
}
