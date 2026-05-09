package prompt

import (
	"fmt"
	"strings"

	shellcontext "github.com/longyijdos/hi-shell/internal/context"
)

type Prompt struct {
	System string
	User   string
}

type Session struct {
	InitialPrompt string        `json:"initial_prompt"`
	Turns         []SessionTurn `json:"turns"`
}

type SessionTurn struct {
	Command  string `json:"command"`
	Risk     string `json:"risk"`
	Warning  string `json:"warning"`
	Feedback string `json:"feedback"`
}

func Build(userRequest string, snapshot shellcontext.Snapshot) Prompt {
	return Prompt{
		System: systemPrompt,
		User: "User request:\n" + userRequest + "\n\n" +
			"Local context:\n" + contextString(snapshot),
	}
}

func BuildSession(session Session, snapshot shellcontext.Snapshot) Prompt {
	if len(session.Turns) == 0 {
		return Build(session.InitialPrompt, snapshot)
	}

	var b strings.Builder
	b.WriteString("Command revision session:\n\n")
	b.WriteString("Initial user request:\n")
	b.WriteString(session.InitialPrompt)
	b.WriteString("\n\n")

	if len(session.Turns) > 0 {
		b.WriteString("Generated commands and user feedback so far:\n")
		for i, turn := range session.Turns {
			fmt.Fprintf(&b, "%d. Command: %s\n", i+1, turn.Command)
			if turn.Risk != "" {
				fmt.Fprintf(&b, "   Risk: %s\n", turn.Risk)
			}
			if turn.Warning != "" {
				fmt.Fprintf(&b, "   Warning: %s\n", turn.Warning)
			}
			if turn.Feedback != "" {
				fmt.Fprintf(&b, "   User feedback after this command: %s\n", turn.Feedback)
			}
		}
	} else {
		b.WriteString("Generated commands and user feedback so far:\n")
		b.WriteString("None.\n")
	}

	b.WriteString("\nLocal context:\n")
	b.WriteString(contextString(snapshot))
	b.WriteString("\n\n")
	b.WriteString("Generate a revised command that satisfies the initial request and the user's feedback. Avoid repeating issues the user has already corrected.")

	return Prompt{
		System: systemPrompt,
		User:   b.String(),
	}
}

func contextString(snapshot shellcontext.Snapshot) string {
	contextText := snapshot.String()
	if contextText == "" {
		return "No local context was collected."
	}
	return contextText
}

const systemPrompt = `You are hi-shell, a tiny zsh command composer.

Return exactly one single-line zsh-compatible shell command and nothing else.

Rules:
- Output must not contain newline characters.
- Do not include markdown, code fences, commentary, explanations, or shell prompts.
- Generate a command for the user's current OS and shell.
- Prefer safe, read-only commands when intent is ambiguous.
- If multiple shell operations are needed, combine them on one line using &&, ||, ;, pipes, subshells, or command substitution.
- Prefer && over ; when later operations depend on earlier operations succeeding.
- Do not chain multiple independent operations unless the user explicitly asks for multiple steps.
- Do not use destructive commands unless the user explicitly requested a destructive action.
- If the request is unsafe or unclear, return the safest useful alternative command.`
