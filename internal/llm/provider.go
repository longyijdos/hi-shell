package llm

import (
	"context"
	"encoding/json"
	"strings"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Request struct {
	Model    string
	Messages []Message
}

type Completion struct {
	Command string
}

type Provider interface {
	Generate(context.Context, Request) (Completion, error)
}

func NormalizeCommand(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	var jsonCommand struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(raw), &jsonCommand); err == nil && strings.TrimSpace(jsonCommand.Command) != "" {
		raw = jsonCommand.Command
	}

	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		raw = strings.TrimSpace(strings.TrimPrefix(raw, "```"))
		if idx := strings.IndexByte(raw, '\n'); idx >= 0 {
			first := strings.TrimSpace(raw[:idx])
			if first == "sh" || first == "shell" || first == "zsh" || first == "bash" {
				raw = raw[idx+1:]
			}
		}
		raw = strings.TrimSpace(strings.TrimSuffix(raw, "```"))
	}

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "```") {
			continue
		}
		line = strings.TrimPrefix(line, "$ ")
		line = strings.TrimPrefix(line, "% ")
		return strings.TrimSpace(line)
	}

	return ""
}
