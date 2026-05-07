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
