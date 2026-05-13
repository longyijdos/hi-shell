package config

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	cfg := Default()
	cfg.Provider = "deepseek"
	cfg.DeepSeek.Model = "deepseek-v4-pro"
	cfg.Keybindings.Prefix = "^O"
	cfg.History.FetchLimit = 30
	cfg.History.MaxEntries = 16

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
	if loaded.Keybindings.Prefix != "^O" {
		t.Fatalf("Keybindings.Prefix = %q", loaded.Keybindings.Prefix)
	}
	if loaded.History.FetchLimit != 30 {
		t.Fatalf("History.FetchLimit = %d", loaded.History.FetchLimit)
	}
	if loaded.History.MaxEntries != 16 {
		t.Fatalf("History.MaxEntries = %d", loaded.History.MaxEntries)
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
	if loaded.Keybindings.Prefix != "^]" {
		t.Fatalf("Keybindings.Prefix = %q, want ^]", loaded.Keybindings.Prefix)
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
		"deepseek.base_url":    "https://api.deepseek.com/v1",
		"deepseek.api_key_env": "OPENAI_API_KEY",
		"deepseek.model":       "deepseek-v4-flash",
		"deepseek.thinking":    "enabled",
		"deepseek.max_tokens":  "128",
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
	if cfg.DeepSeek.MaxTokens != 128 {
		t.Fatalf("DeepSeek.MaxTokens = %d", cfg.DeepSeek.MaxTokens)
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

func TestSetKeybindingsPrefix(t *testing.T) {
	cfg := Default()

	if err := Set(&cfg, "keybindings.prefix", "^[;"); err != nil {
		t.Fatalf("Set(keybindings.prefix) error = %v", err)
	}
	if cfg.Keybindings.Prefix != "^[;" {
		t.Fatalf("Keybindings.Prefix = %q", cfg.Keybindings.Prefix)
	}

	got, err := Get(cfg, "keybindings.prefix")
	if err != nil {
		t.Fatalf("Get(keybindings.prefix) error = %v", err)
	}
	if got != "^[;" {
		t.Fatalf("Get(keybindings.prefix) = %q", got)
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
