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

type ReviseSession struct {
	InitialPrompt string       `json:"initial_prompt"`
	Turns         []ReviseTurn `json:"turns"`
}

type ReviseTurn struct {
	Command  string `json:"command"`
	Risk     string `json:"risk"`
	Warning  string `json:"warning"`
	Feedback string `json:"feedback"`
}

type AskSession struct {
	InitialPrompt string    `json:"initial_prompt"`
	Turns         []AskTurn `json:"turns"`
}

type AskTurn struct {
	Command  string `json:"command"`
	Risk     string `json:"risk"`
	Warning  string `json:"warning"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

func BuildGenerate(userRequest string, snapshot shellcontext.Snapshot) Prompt {
	return Prompt{
		System: commandSystemPrompt,
		User: "User request:\n" + userRequest + "\n\n" +
			"Local context:\n" + contextString(snapshot),
	}
}

func BuildReviseSession(session ReviseSession, snapshot shellcontext.Snapshot) Prompt {
	if len(session.Turns) == 0 {
		return BuildGenerate(session.InitialPrompt, snapshot)
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
		System: commandSystemPrompt,
		User:   b.String(),
	}
}

func BuildAskSession(session AskSession, snapshot shellcontext.Snapshot) Prompt {
	var b strings.Builder
	b.WriteString("Command question session:\n\n")
	b.WriteString("Initial user request:\n")
	b.WriteString(session.InitialPrompt)
	b.WriteString("\n\n")

	b.WriteString("Commands and questions so far:\n")
	for i, turn := range session.Turns {
		fmt.Fprintf(&b, "%d. Command: %s\n", i+1, turn.Command)
		if turn.Risk != "" {
			fmt.Fprintf(&b, "   Risk: %s\n", turn.Risk)
		}
		if turn.Warning != "" {
			fmt.Fprintf(&b, "   Warning: %s\n", turn.Warning)
		}
		if turn.Question != "" {
			fmt.Fprintf(&b, "   User question: %s\n", turn.Question)
		}
		if turn.Answer != "" {
			fmt.Fprintf(&b, "   Previous answer: %s\n", turn.Answer)
		}
	}

	b.WriteString("\nLocal context:\n")
	b.WriteString(contextString(snapshot))
	b.WriteString("\n\n")
	b.WriteString("Answer the user's latest question about the command. Be concise and practical. Do not generate a replacement command unless the user explicitly asks what a possible command would look like.")

	return Prompt{
		System: askSystemPrompt,
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

const commandSystemPrompt = `You are hi-shell, a tiny zsh command composer.

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

const askSystemPrompt = `You are hi-shell, a concise shell command explainer.

Answer questions about the provided zsh command and its risk/warning context.

Rules:
- Return only the answer text.
- Do not include markdown code fences.
- Be direct and practical.
- If the question asks whether the command changes files, installs packages, deletes data, or touches credentials, answer that explicitly.
- Do not invent facts about files or command output that are not present in the context.`
