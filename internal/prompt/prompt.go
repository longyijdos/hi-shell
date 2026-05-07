package prompt

import shellcontext "github.com/longyijdos/hi-shell/internal/context"

type Prompt struct {
	System string
	User   string
}

func Build(userRequest string, snapshot shellcontext.Snapshot) Prompt {
	contextText := snapshot.String()
	if contextText == "" {
		contextText = "No local context was collected."
	}

	return Prompt{
		System: systemPrompt,
		User: "User request:\n" + userRequest + "\n\n" +
			"Local context:\n" + contextText,
	}
}

const systemPrompt = `You are hi-shell, a tiny zsh command composer.

Return exactly one zsh-compatible shell command and nothing else.

Rules:
- Generate a command for the user's current OS and shell.
- Prefer safe, read-only commands when intent is ambiguous.
- Do not include markdown, code fences, commentary, or explanations.
- Do not chain multiple commands unless the user explicitly asks for multiple steps.
- Do not use destructive commands unless the user explicitly requested a destructive action.
- If the request is unsafe or unclear, return the safest useful alternative command.`
