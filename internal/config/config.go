package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const (
	HomeEnv        = "HI_HOME"
	DefaultHomeDir = ".hi"
	ConfigFileName = "config.toml"
)

var ErrSecretNotStored = errors.New("hi-shell does not store API keys in config; set the provider API key environment variable instead")

type Config struct {
	Provider  string         `toml:"provider"`
	Model     string         `toml:"model"`
	TimeoutMS int            `toml:"timeout_ms"`
	OpenAI    OpenAIConfig   `toml:"openai"`
	DeepSeek  DeepSeekConfig `toml:"deepseek"`
	Context   ContextConfig  `toml:"context"`
	Safety    SafetyConfig   `toml:"safety"`
}

type OpenAIConfig struct {
	BaseURL   string `toml:"base_url"`
	APIKeyEnv string `toml:"api_key_env"`
}

type DeepSeekConfig struct {
	BaseURL   string `toml:"base_url"`
	APIKeyEnv string `toml:"api_key_env"`
	Model     string `toml:"model"`
	Thinking  string `toml:"thinking"`
	MaxTokens int    `toml:"max_tokens"`
}

type ContextConfig struct {
	PWD            bool `toml:"pwd"`
	OS             bool `toml:"os"`
	Shell          bool `toml:"shell"`
	Git            bool `toml:"git"`
	ProjectFiles   bool `toml:"project_files"`
	PackageScripts bool `toml:"package_scripts"`
	History        bool `toml:"history"`
}

type SafetyConfig struct {
	BlockCritical   bool `toml:"block_critical"`
	WarnSudo        bool `toml:"warn_sudo"`
	WarnDestructive bool `toml:"warn_destructive"`
}

func Default() Config {
	return Config{
		Provider:  "openai",
		Model:     "gpt-4.1-mini",
		TimeoutMS: 5000,
		OpenAI: OpenAIConfig{
			BaseURL:   "https://api.openai.com/v1",
			APIKeyEnv: "OPENAI_API_KEY",
		},
		DeepSeek: DeepSeekConfig{
			BaseURL:   "https://api.deepseek.com/v1",
			APIKeyEnv: "DEEPSEEK_API_KEY",
			Model:     "deepseek-v4-flash",
			Thinking:  "disabled",
			MaxTokens: 256,
		},
		Context: ContextConfig{
			PWD:            true,
			OS:             true,
			Shell:          true,
			Git:            true,
			ProjectFiles:   true,
			PackageScripts: true,
			History:        false,
		},
		Safety: SafetyConfig{
			BlockCritical:   true,
			WarnSudo:        true,
			WarnDestructive: true,
		},
	}
}

func HomeDir() (string, error) {
	if home := strings.TrimSpace(os.Getenv(HomeEnv)); home != "" {
		return home, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, DefaultHomeDir), nil
}

func Path() (string, error) {
	home, err := HomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ConfigFileName), nil
}

func Load() (Config, string, error) {
	path, err := Path()
	if err != nil {
		return Config{}, "", err
	}

	cfg, err := LoadFile(path)
	if err != nil {
		return Config{}, path, err
	}
	return cfg, path, nil
}

func LoadFile(path string) (Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return Config{}, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return cfg, nil
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}

	applyDefaults(&cfg)
	return cfg, nil
}

func Ensure() (Config, string, error) {
	path, err := Path()
	if err != nil {
		return Config{}, "", err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Config{}, path, err
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		cfg := Default()
		if err := SaveFile(path, cfg); err != nil {
			return Config{}, path, err
		}
		return cfg, path, nil
	} else if err != nil {
		return Config{}, path, err
	}

	cfg, err := LoadFile(path)
	return cfg, path, err
}

func SaveFile(path string, cfg Config) error {
	applyDefaults(&cfg)

	data, err := Marshal(cfg)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func Marshal(cfg Config) ([]byte, error) {
	applyDefaults(&cfg)

	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(cfg); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Set(cfg *Config, key, value string) error {
	key = strings.ToLower(strings.TrimSpace(key))
	value = strings.TrimSpace(value)

	switch key {
	case "provider":
		cfg.Provider = value
	case "model":
		cfg.Model = value
	case "timeout_ms":
		timeout, err := strconv.Atoi(value)
		if err != nil || timeout <= 0 {
			return fmt.Errorf("timeout_ms must be a positive integer")
		}
		cfg.TimeoutMS = timeout
	case "openai.base_url", "base_url":
		cfg.OpenAI.BaseURL = value
	case "openai.api_key_env", "api_key_env":
		cfg.OpenAI.APIKeyEnv = value
	case "api_key":
		return ErrSecretNotStored
	case "deepseek.base_url":
		cfg.DeepSeek.BaseURL = value
	case "deepseek.api_key_env":
		cfg.DeepSeek.APIKeyEnv = value
	case "deepseek.model":
		cfg.DeepSeek.Model = value
	case "deepseek.thinking":
		if value != "enabled" && value != "disabled" {
			return fmt.Errorf("deepseek.thinking must be enabled or disabled")
		}
		cfg.DeepSeek.Thinking = value
	case "deepseek.max_tokens":
		maxTokens, err := strconv.Atoi(value)
		if err != nil || maxTokens <= 0 {
			return fmt.Errorf("deepseek.max_tokens must be a positive integer")
		}
		cfg.DeepSeek.MaxTokens = maxTokens
	case "context.pwd":
		return setBool(value, &cfg.Context.PWD)
	case "context.os":
		return setBool(value, &cfg.Context.OS)
	case "context.shell":
		return setBool(value, &cfg.Context.Shell)
	case "context.git":
		return setBool(value, &cfg.Context.Git)
	case "context.project_files":
		return setBool(value, &cfg.Context.ProjectFiles)
	case "context.package_scripts":
		return setBool(value, &cfg.Context.PackageScripts)
	case "context.history":
		return setBool(value, &cfg.Context.History)
	case "safety.block_critical":
		return setBool(value, &cfg.Safety.BlockCritical)
	case "safety.warn_sudo":
		return setBool(value, &cfg.Safety.WarnSudo)
	case "safety.warn_destructive":
		return setBool(value, &cfg.Safety.WarnDestructive)
	default:
		return fmt.Errorf("unknown config key %q", key)
	}

	applyDefaults(cfg)
	return nil
}

func Get(cfg Config, key string) (string, error) {
	key = strings.ToLower(strings.TrimSpace(key))

	switch key {
	case "provider":
		return cfg.Provider, nil
	case "model":
		return cfg.Model, nil
	case "timeout_ms":
		return strconv.Itoa(cfg.TimeoutMS), nil
	case "openai.base_url", "base_url":
		return cfg.OpenAI.BaseURL, nil
	case "openai.api_key_env", "api_key_env":
		return cfg.OpenAI.APIKeyEnv, nil
	case "deepseek.base_url":
		return cfg.DeepSeek.BaseURL, nil
	case "deepseek.api_key_env":
		return cfg.DeepSeek.APIKeyEnv, nil
	case "deepseek.model":
		return cfg.DeepSeek.Model, nil
	case "deepseek.thinking":
		return cfg.DeepSeek.Thinking, nil
	case "deepseek.max_tokens":
		return strconv.Itoa(cfg.DeepSeek.MaxTokens), nil
	case "context.pwd":
		return strconv.FormatBool(cfg.Context.PWD), nil
	case "context.os":
		return strconv.FormatBool(cfg.Context.OS), nil
	case "context.shell":
		return strconv.FormatBool(cfg.Context.Shell), nil
	case "context.git":
		return strconv.FormatBool(cfg.Context.Git), nil
	case "context.project_files":
		return strconv.FormatBool(cfg.Context.ProjectFiles), nil
	case "context.package_scripts":
		return strconv.FormatBool(cfg.Context.PackageScripts), nil
	case "context.history":
		return strconv.FormatBool(cfg.Context.History), nil
	case "safety.block_critical":
		return strconv.FormatBool(cfg.Safety.BlockCritical), nil
	case "safety.warn_sudo":
		return strconv.FormatBool(cfg.Safety.WarnSudo), nil
	case "safety.warn_destructive":
		return strconv.FormatBool(cfg.Safety.WarnDestructive), nil
	default:
		return "", fmt.Errorf("unknown config key %q", key)
	}
}

func setBool(value string, target *bool) error {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("value must be true or false")
	}
	*target = parsed
	return nil
}

func applyDefaults(cfg *Config) {
	defaults := Default()

	if cfg.Provider == "" {
		cfg.Provider = defaults.Provider
	}
	if cfg.Model == "" {
		cfg.Model = defaults.Model
	}
	if cfg.TimeoutMS <= 0 {
		cfg.TimeoutMS = defaults.TimeoutMS
	}
	if cfg.OpenAI.BaseURL == "" {
		cfg.OpenAI.BaseURL = defaults.OpenAI.BaseURL
	}
	if cfg.OpenAI.APIKeyEnv == "" {
		cfg.OpenAI.APIKeyEnv = defaults.OpenAI.APIKeyEnv
	}
	if cfg.DeepSeek.BaseURL == "" {
		cfg.DeepSeek.BaseURL = defaults.DeepSeek.BaseURL
	}
	if cfg.DeepSeek.APIKeyEnv == "" {
		cfg.DeepSeek.APIKeyEnv = defaults.DeepSeek.APIKeyEnv
	}
	if cfg.DeepSeek.Model == "" {
		cfg.DeepSeek.Model = defaults.DeepSeek.Model
	}
	if cfg.DeepSeek.Thinking == "" {
		cfg.DeepSeek.Thinking = defaults.DeepSeek.Thinking
	}
	if cfg.DeepSeek.MaxTokens <= 0 {
		cfg.DeepSeek.MaxTokens = defaults.DeepSeek.MaxTokens
	}
}
