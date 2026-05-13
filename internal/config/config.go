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
	HomeEnv        = "HI_SHELL_HOME"
	DefaultHomeDir = ".hi-shell"
	ConfigFileName = "config.toml"
)

var ErrSecretNotStored = errors.New("hi-shell does not store API keys in config; set the provider API key environment variable instead")

type Config struct {
	Provider    string            `toml:"provider"`
	TimeoutMS   int               `toml:"timeout_ms"`
	Keybindings KeybindingsConfig `toml:"keybindings"`
	OpenAI      OpenAIConfig      `toml:"openai"`
	DeepSeek    DeepSeekConfig    `toml:"deepseek"`
	Context     ContextConfig     `toml:"context"`
	History     HistoryConfig     `toml:"history"`
	Session     SessionConfig     `toml:"session"`
	Safety      SafetyConfig      `toml:"safety"`
}

type OpenAIConfig struct {
	BaseURL   string `toml:"base_url"`
	APIKeyEnv string `toml:"api_key_env"`
	Model     string `toml:"model"`
}

type DeepSeekConfig struct {
	BaseURL   string `toml:"base_url"`
	APIKeyEnv string `toml:"api_key_env"`
	Model     string `toml:"model"`
	Thinking  string `toml:"thinking"`
	MaxTokens int    `toml:"max_tokens"`
}

type KeybindingsConfig struct {
	Prefix string `toml:"prefix"`
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

type HistoryConfig struct {
	FetchLimit      int `toml:"fetch_limit"`
	MaxEntries      int `toml:"max_entries"`
	MaxCommandChars int `toml:"max_command_chars"`
	MaxBytes        int `toml:"max_bytes"`
}

type SessionConfig struct {
	ReviseTurns   int `toml:"revise_turns"`
	AskTurns      int `toml:"ask_turns"`
	MaxFieldChars int `toml:"max_field_chars"`
	MaxJSONBytes  int `toml:"max_json_bytes"`
}

type SafetyConfig struct {
	BlockCritical   bool `toml:"block_critical"`
	WarnSudo        bool `toml:"warn_sudo"`
	WarnDestructive bool `toml:"warn_destructive"`
}

func Default() Config {
	return Config{
		Provider:  "openai",
		TimeoutMS: 5000,
		Keybindings: KeybindingsConfig{
			Prefix: "^]",
		},
		OpenAI: OpenAIConfig{
			BaseURL:   "https://api.openai.com/v1",
			APIKeyEnv: "OPENAI_API_KEY",
			Model:     "gpt-4.1-mini",
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
		History: HistoryConfig{
			FetchLimit:      20,
			MaxEntries:      12,
			MaxCommandChars: 240,
			MaxBytes:        2000,
		},
		Session: SessionConfig{
			ReviseTurns:   8,
			AskTurns:      8,
			MaxFieldChars: 4000,
			MaxJSONBytes:  64 * 1024,
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
	case "timeout_ms":
		timeout, err := strconv.Atoi(value)
		if err != nil || timeout <= 0 {
			return fmt.Errorf("timeout_ms must be a positive integer")
		}
		cfg.TimeoutMS = timeout
	case "openai.model":
		cfg.OpenAI.Model = value
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
	case "keybindings.prefix":
		if value == "" {
			return fmt.Errorf("keybindings.prefix must be non-empty")
		}
		cfg.Keybindings.Prefix = value
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
	case "history.fetch_limit":
		return setPositiveInt(value, &cfg.History.FetchLimit, "history.fetch_limit")
	case "history.max_entries":
		return setPositiveInt(value, &cfg.History.MaxEntries, "history.max_entries")
	case "history.max_command_chars":
		return setPositiveInt(value, &cfg.History.MaxCommandChars, "history.max_command_chars")
	case "history.max_bytes":
		return setPositiveInt(value, &cfg.History.MaxBytes, "history.max_bytes")
	case "session.revise_turns":
		return setPositiveInt(value, &cfg.Session.ReviseTurns, "session.revise_turns")
	case "session.ask_turns":
		return setPositiveInt(value, &cfg.Session.AskTurns, "session.ask_turns")
	case "session.max_field_chars":
		return setPositiveInt(value, &cfg.Session.MaxFieldChars, "session.max_field_chars")
	case "session.max_json_bytes":
		return setPositiveInt(value, &cfg.Session.MaxJSONBytes, "session.max_json_bytes")
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
	case "timeout_ms":
		return strconv.Itoa(cfg.TimeoutMS), nil
	case "openai.model":
		return cfg.OpenAI.Model, nil
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
	case "keybindings.prefix":
		return cfg.Keybindings.Prefix, nil
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
	case "history.fetch_limit":
		return strconv.Itoa(cfg.History.FetchLimit), nil
	case "history.max_entries":
		return strconv.Itoa(cfg.History.MaxEntries), nil
	case "history.max_command_chars":
		return strconv.Itoa(cfg.History.MaxCommandChars), nil
	case "history.max_bytes":
		return strconv.Itoa(cfg.History.MaxBytes), nil
	case "session.revise_turns":
		return strconv.Itoa(cfg.Session.ReviseTurns), nil
	case "session.ask_turns":
		return strconv.Itoa(cfg.Session.AskTurns), nil
	case "session.max_field_chars":
		return strconv.Itoa(cfg.Session.MaxFieldChars), nil
	case "session.max_json_bytes":
		return strconv.Itoa(cfg.Session.MaxJSONBytes), nil
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

func setPositiveInt(value string, target *int, name string) error {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fmt.Errorf("%s must be a positive integer", name)
	}
	*target = parsed
	return nil
}

func applyDefaults(cfg *Config) {
	defaults := Default()

	if cfg.Provider == "" {
		cfg.Provider = defaults.Provider
	}
	if cfg.TimeoutMS <= 0 {
		cfg.TimeoutMS = defaults.TimeoutMS
	}
	cfg.Keybindings.Prefix = strings.TrimSpace(cfg.Keybindings.Prefix)
	if cfg.Keybindings.Prefix == "" {
		cfg.Keybindings.Prefix = defaults.Keybindings.Prefix
	}
	if cfg.History.FetchLimit <= 0 {
		cfg.History.FetchLimit = defaults.History.FetchLimit
	}
	if cfg.History.MaxEntries <= 0 {
		cfg.History.MaxEntries = defaults.History.MaxEntries
	}
	if cfg.History.MaxCommandChars <= 0 {
		cfg.History.MaxCommandChars = defaults.History.MaxCommandChars
	}
	if cfg.History.MaxBytes <= 0 {
		cfg.History.MaxBytes = defaults.History.MaxBytes
	}
	if cfg.Session.ReviseTurns <= 0 {
		cfg.Session.ReviseTurns = defaults.Session.ReviseTurns
	}
	if cfg.Session.AskTurns <= 0 {
		cfg.Session.AskTurns = defaults.Session.AskTurns
	}
	if cfg.Session.MaxFieldChars <= 0 {
		cfg.Session.MaxFieldChars = defaults.Session.MaxFieldChars
	}
	if cfg.Session.MaxJSONBytes <= 0 {
		cfg.Session.MaxJSONBytes = defaults.Session.MaxJSONBytes
	}
	if cfg.OpenAI.BaseURL == "" {
		cfg.OpenAI.BaseURL = defaults.OpenAI.BaseURL
	}
	if cfg.OpenAI.APIKeyEnv == "" {
		cfg.OpenAI.APIKeyEnv = defaults.OpenAI.APIKeyEnv
	}
	if cfg.OpenAI.Model == "" {
		cfg.OpenAI.Model = defaults.OpenAI.Model
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
