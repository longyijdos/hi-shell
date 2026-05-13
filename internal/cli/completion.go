package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/longyijdos/hi-shell/internal/config"
	shellcontext "github.com/longyijdos/hi-shell/internal/context"
	"github.com/longyijdos/hi-shell/internal/llm"
	promptpkg "github.com/longyijdos/hi-shell/internal/prompt"
	"github.com/longyijdos/hi-shell/internal/risk"
)

type generateResponse struct {
	Command string `json:"command"`
	Risk    string `json:"risk"`
	Warning string `json:"warning"`
}

func parseOutputFormat(outputFormat string) (string, error) {
	outputFormat = strings.TrimSpace(outputFormat)
	if outputFormat != "text" && outputFormat != "json" {
		return "", fmt.Errorf("unsupported format %q", outputFormat)
	}
	return outputFormat, nil
}

func runCommandGeneration(outputFormat string, buildPrompt func(shellcontext.Snapshot) promptpkg.Prompt, stdout, stderr io.Writer) int {
	cfg, _, err := config.Ensure()
	if err != nil {
		fmt.Fprintf(stderr, "load config: %v\n", err)
		return 1
	}

	snapshot := shellcontext.Collect(cfg.Context, cfg.History)
	builtPrompt := buildPrompt(snapshot)
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
		Command: command,
		Risk:    string(assessment.Level),
		Warning: assessment.Warning,
	}
	if assessment.Blocked {
		response.Command = ""
		response.Warning = assessment.Warning
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
	}

	return 0
}

func providerFor(cfg config.Config) (llm.Provider, string, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "openai", "openai-compatible":
		return llm.OpenAIProvider{
			BaseURL:   cfg.OpenAI.BaseURL,
			APIKeyEnv: cfg.OpenAI.APIKeyEnv,
		}, cfg.OpenAI.Model, nil
	case "deepseek":
		return llm.DeepSeekProvider{
			BaseURL:   cfg.DeepSeek.BaseURL,
			APIKeyEnv: cfg.DeepSeek.APIKeyEnv,
			Thinking:  cfg.DeepSeek.Thinking,
			MaxTokens: cfg.DeepSeek.MaxTokens,
		}, cfg.DeepSeek.Model, nil
	default:
		return nil, "", fmt.Errorf("unsupported provider %q; use openai, openai-compatible, or deepseek", cfg.Provider)
	}
}
