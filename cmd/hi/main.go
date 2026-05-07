package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	hishell "github.com/longyijdos/hi-shell"
	"github.com/longyijdos/hi-shell/internal/config"
	shellcontext "github.com/longyijdos/hi-shell/internal/context"
	"github.com/longyijdos/hi-shell/internal/llm"
	promptpkg "github.com/longyijdos/hi-shell/internal/prompt"
	"github.com/longyijdos/hi-shell/internal/risk"
)

var version = "dev"

const (
	beginMarker = "# >>> hi-shell initialize >>>"
	endMarker   = "# <<< hi-shell initialize <<<"
)

type generateResponse struct {
	Command     string `json:"command"`
	Risk        string `json:"risk"`
	Warning     string `json:"warning"`
	Explanation string `json:"explanation"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}

	switch args[0] {
	case "generate":
		return commandGenerate(args[1:], stdout, stderr)
	case "config":
		return commandConfig(args[1:], stdout, stderr)
	case "install":
		return commandInstall(args[1:], stdout, stderr)
	case "uninstall":
		return commandUninstall(args[1:], stdout, stderr)
	case "doctor":
		return commandDoctor(args[1:], stdout, stderr)
	case "version":
		fmt.Fprintln(stdout, version)
		return 0
	case "parse-command":
		return commandParseField(append([]string{"command"}, args[1:]...), stdout, stderr)
	case "parse-field":
		return commandParseField(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		usage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		usage(stderr)
		return 2
	}
}

func commandGenerate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var promptText string
	var outputFormat string
	fs.StringVar(&promptText, "prompt", "", "natural language command request")
	fs.StringVar(&outputFormat, "format", "text", "output format: text or json")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if promptText == "" {
		promptText = strings.TrimSpace(strings.Join(fs.Args(), " "))
	}
	if promptText == "" {
		fmt.Fprintln(stderr, "hi generate requires --prompt or prompt text")
		return 2
	}

	cfg, _, err := config.Ensure()
	if err != nil {
		fmt.Fprintf(stderr, "load config: %v\n", err)
		return 1
	}

	snapshot := shellcontext.Collect(cfg.Context)
	builtPrompt := promptpkg.Build(promptText, snapshot)
	provider, model, err := providerFor(cfg)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	timeout := time.Duration(cfg.TimeoutMS) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	completion, err := provider.Generate(ctx, llm.Request{
		Model: model,
		Messages: []llm.Message{
			{Role: "system", Content: builtPrompt.System},
			{Role: "user", Content: builtPrompt.User},
		},
	})
	if err != nil {
		fmt.Fprintf(stderr, "generate: %v\n", err)
		return 1
	}

	command := llm.NormalizeCommand(completion.Command)
	if command == "" {
		fmt.Fprintln(stderr, "generate: provider returned an empty command")
		return 1
	}

	assessment := risk.Score(command, cfg.Safety)
	response := generateResponse{
		Command:     command,
		Risk:        string(assessment.Level),
		Warning:     assessment.Warning,
		Explanation: completion.Explanation,
	}
	if assessment.Blocked {
		response.Command = ""
		response.Warning = assessment.Warning
		response.Explanation = "The generated command was blocked by safety settings."
	}

	switch outputFormat {
	case "json":
		encoder := json.NewEncoder(stdout)
		if err := encoder.Encode(response); err != nil {
			fmt.Fprintf(stderr, "encode json: %v\n", err)
			return 1
		}
	case "text":
		if response.Command == "" {
			fmt.Fprintln(stderr, response.Warning)
			return 1
		}
		fmt.Fprintln(stdout, response.Command)
		if response.Warning != "" {
			fmt.Fprintln(stderr, response.Warning)
		}
	default:
		fmt.Fprintf(stderr, "unsupported format %q\n", outputFormat)
		return 2
	}

	return 0
}

func providerFor(cfg config.Config) (llm.Provider, string, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "openai", "openai-compatible":
		return llm.OpenAIProvider{
			BaseURL:   cfg.OpenAI.BaseURL,
			APIKeyEnv: cfg.OpenAI.APIKeyEnv,
		}, cfg.Model, nil
	case "deepseek":
		model := cfg.DeepSeek.Model
		if cfg.Model != "" && cfg.Model != config.Default().Model {
			model = cfg.Model
		}
		return llm.DeepSeekProvider{
			BaseURL:   cfg.DeepSeek.BaseURL,
			APIKeyEnv: cfg.DeepSeek.APIKeyEnv,
			Thinking:  cfg.DeepSeek.Thinking,
			MaxTokens: cfg.DeepSeek.MaxTokens,
		}, model, nil
	default:
		return nil, "", fmt.Errorf("unsupported provider %q; use openai, openai-compatible, or deepseek", cfg.Provider)
	}
}

func commandConfig(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: hi config get [key] | hi config set <key> <value> | hi config path")
		return 2
	}

	cfg, path, err := config.Ensure()
	if err != nil {
		fmt.Fprintf(stderr, "config: %v\n", err)
		return 1
	}

	switch args[0] {
	case "get":
		if len(args) == 1 {
			data, err := config.Marshal(cfg)
			if err != nil {
				fmt.Fprintf(stderr, "config: %v\n", err)
				return 1
			}
			fmt.Fprint(stdout, string(data))
			return 0
		}
		value, err := config.Get(cfg, args[1])
		if err != nil {
			fmt.Fprintf(stderr, "config: %v\n", err)
			return 2
		}
		fmt.Fprintln(stdout, value)
		return 0
	case "set":
		if len(args) < 3 {
			fmt.Fprintln(stderr, "usage: hi config set <key> <value>")
			return 2
		}
		value := strings.Join(args[2:], " ")
		if err := config.Set(&cfg, args[1], value); err != nil {
			if errors.Is(err, config.ErrSecretNotStored) {
				fmt.Fprintln(stderr, err)
				return 2
			}
			fmt.Fprintf(stderr, "config: %v\n", err)
			return 2
		}
		if err := config.SaveFile(path, cfg); err != nil {
			fmt.Fprintf(stderr, "config: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "set %s\n", args[1])
		return 0
	case "path":
		fmt.Fprintln(stdout, path)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown config command %q\n", args[0])
		return 2
	}
}

func commandInstall(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "zsh" {
		fmt.Fprintln(stderr, "usage: hi install zsh")
		return 2
	}

	_, cfgPath, err := config.Ensure()
	if err != nil {
		fmt.Fprintf(stderr, "install: %v\n", err)
		return 1
	}

	hiHome, err := config.HomeDir()
	if err != nil {
		fmt.Fprintf(stderr, "install: %v\n", err)
		return 1
	}

	shellDir := filepath.Join(hiHome, "shell")
	if err := os.MkdirAll(shellDir, 0o755); err != nil {
		fmt.Fprintf(stderr, "install: %v\n", err)
		return 1
	}

	pluginPath := filepath.Join(shellDir, "hi.zsh")
	if err := os.WriteFile(pluginPath, []byte(hishell.ShellPlugin), 0o644); err != nil {
		fmt.Fprintf(stderr, "install: %v\n", err)
		return 1
	}

	zshrcPath, err := zshrcPath()
	if err != nil {
		fmt.Fprintf(stderr, "install: %v\n", err)
		return 1
	}
	if err := ensureManagedBlock(zshrcPath, managedBlock()); err != nil {
		fmt.Fprintf(stderr, "install: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Installed zsh plugin: %s\n", pluginPath)
	fmt.Fprintf(stdout, "Config: %s\n", cfgPath)
	fmt.Fprintln(stdout, "Restart zsh or run: exec zsh")
	return 0
}

func commandUninstall(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var purge bool
	fs.BoolVar(&purge, "purge", false, "remove ~/.hi, including config")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	zshrc, err := zshrcPath()
	if err == nil {
		if err := removeManagedBlock(zshrc); err != nil {
			fmt.Fprintf(stderr, "uninstall: %v\n", err)
			return 1
		}
	}

	home, err := os.UserHomeDir()
	if err == nil {
		_ = os.Remove(filepath.Join(home, ".local", "bin", "hi"))
	}

	hiHome, err := config.HomeDir()
	if err == nil {
		if purge {
			_ = os.RemoveAll(hiHome)
		} else {
			_ = os.Remove(filepath.Join(hiHome, "shell", "hi.zsh"))
		}
	}

	if purge {
		fmt.Fprintln(stdout, "Uninstalled hi-shell and removed ~/.hi.")
	} else {
		fmt.Fprintln(stdout, "Uninstalled hi-shell. Config was preserved.")
	}
	return 0
}

func commandDoctor(args []string, stdout, stderr io.Writer) int {
	_ = args
	_ = stderr

	ok := true
	check := func(name string, pass bool, detail string) {
		status := "ok"
		if !pass {
			status = "fail"
			ok = false
		}
		if detail == "" {
			fmt.Fprintf(stdout, "%-18s %s\n", name, status)
		} else {
			fmt.Fprintf(stdout, "%-18s %s - %s\n", name, status, detail)
		}
	}

	fmt.Fprintf(stdout, "hi-shell %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)

	cfg, cfgPath, err := config.Load()
	_, cfgStatErr := os.Stat(cfgPath)
	check("config", err == nil && cfgStatErr == nil, cfgPath)
	if err != nil {
		return 1
	}

	check("provider", cfg.Provider == "openai" || cfg.Provider == "deepseek" || cfg.Provider == "openai-compatible", cfg.Provider)
	switch strings.ToLower(cfg.Provider) {
	case "openai", "openai-compatible":
		apiKeyReady := cfg.OpenAI.APIKeyEnv == "" || os.Getenv(cfg.OpenAI.APIKeyEnv) != "" || !strings.Contains(cfg.OpenAI.BaseURL, "api.openai.com")
		detail := cfg.OpenAI.BaseURL
		if cfg.OpenAI.APIKeyEnv != "" {
			detail += " via $" + cfg.OpenAI.APIKeyEnv
		}
		check("openai", apiKeyReady, detail)
	case "deepseek":
		apiKeyReady := cfg.DeepSeek.APIKeyEnv == "" || os.Getenv(cfg.DeepSeek.APIKeyEnv) != "" || !strings.Contains(cfg.DeepSeek.BaseURL, "deepseek.com")
		detail := cfg.DeepSeek.BaseURL + ", thinking=" + cfg.DeepSeek.Thinking
		if cfg.DeepSeek.APIKeyEnv != "" {
			detail += " via $" + cfg.DeepSeek.APIKeyEnv
		}
		check("deepseek", apiKeyReady, detail)
	}

	hiHome, err := config.HomeDir()
	if err == nil {
		_, statErr := os.Stat(filepath.Join(hiHome, "shell", "hi.zsh"))
		check("zsh plugin", statErr == nil, filepath.Join(hiHome, "shell", "hi.zsh"))
	}

	zshrc, err := zshrcPath()
	if err == nil {
		hasBlock := fileContains(zshrc, beginMarker) && fileContains(zshrc, endMarker)
		check("zshrc block", hasBlock, zshrc)
	}

	hiPath, err := exec.LookPath("hi")
	check("binary in PATH", err == nil, hiPath)

	if !ok {
		return 1
	}
	return 0
}

func commandParseField(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		fmt.Fprintln(stderr, "usage: hi parse-field <field> <json>")
		return 2
	}

	field := args[0]
	raw := strings.Join(args[1:], " ")

	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		fmt.Fprintf(stderr, "parse json: %v\n", err)
		return 1
	}

	value, ok := parsed[field]
	if !ok || value == nil {
		return 0
	}

	switch typed := value.(type) {
	case string:
		fmt.Fprintln(stdout, typed)
	case bool:
		fmt.Fprintln(stdout, typed)
	case float64:
		fmt.Fprintln(stdout, typed)
	default:
		encoded, _ := json.Marshal(typed)
		fmt.Fprintln(stdout, string(encoded))
	}
	return 0
}

func managedBlock() string {
	return beginMarker + "\n" +
		`source "$HOME/.hi/shell/hi.zsh"` + "\n" +
		endMarker + "\n"
}

func zshrcPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".zshrc"), nil
}

func ensureManagedBlock(path, block string) error {
	existing, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	text, _ := removeBlock(string(existing))
	text = strings.TrimRight(text, "\n")
	if text != "" {
		text += "\n\n"
	}
	text += block

	return os.WriteFile(path, []byte(text), 0o644)
}

func removeManagedBlock(path string) error {
	existing, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	text, changed := removeBlock(string(existing))
	if !changed {
		return nil
	}
	return os.WriteFile(path, []byte(text), 0o644)
}

func removeBlock(text string) (string, bool) {
	start := strings.Index(text, beginMarker)
	if start < 0 {
		return text, false
	}
	endRel := strings.Index(text[start:], endMarker)
	if endRel < 0 {
		return text, false
	}
	end := start + endRel + len(endMarker)
	if end < len(text) && text[end] == '\n' {
		end++
	}

	before := strings.TrimRight(text[:start], "\n")
	after := strings.TrimLeft(text[end:], "\n")
	switch {
	case before == "":
		return after, true
	case after == "":
		return before + "\n", true
	default:
		return before + "\n\n" + after, true
	}
}

func fileContains(path, needle string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), needle)
}

func usage(w io.Writer) {
	fmt.Fprintln(w, `hi-shell: tiny AI command composer for zsh

Usage:
  hi generate --prompt "list all files" --format json
  hi config get [key]
  hi config set <key> <value>
  hi install zsh
  hi uninstall [--purge]
  hi doctor
  hi version`)
}
