package cli

import (
	"bytes"
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
)

type askResponse struct {
	Answer string `json:"answer"`
}

func commandAsk(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if wantsHelp(args) {
		return printHelp(stdout, askUsage)
	}

	fs := newFlagSet("ask", askUsage, stderr)

	var sessionJSON string
	var outputFormat string
	fs.StringVar(&sessionJSON, "session-json", "", "ask session JSON: inline, @file, or - for stdin")
	fs.StringVar(&outputFormat, "format", "text", "output format: text or json")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	sessionJSON = strings.TrimSpace(sessionJSON)
	if sessionJSON == "" {
		fmt.Fprintln(stderr, `hi-shell ask requires --session-json with non-empty JSON, for example: hi-shell ask --session-json -`)
		return 2
	}
	if rejectUnexpectedArgs(stderr, "ask", fs.Args()) {
		return 2
	}
	format, err := parseOutputFormat(outputFormat)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	session, err := readAskSession(sessionJSON, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "session-json: %v\n", err)
		return 2
	}

	return runAskCompletion(format, func(snapshot shellcontext.Snapshot) promptpkg.Prompt {
		return promptpkg.BuildAskSession(session, snapshot)
	}, stdout, stderr)
}

func readAskSession(source string, stdin io.Reader) (promptpkg.AskSession, error) {
	data, err := readSessionJSON(source, stdin)
	if err != nil {
		return promptpkg.AskSession{}, err
	}
	return parseAskSession(data)
}

func parseAskSession(data []byte) (promptpkg.AskSession, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return promptpkg.AskSession{}, fmt.Errorf("requires non-empty JSON")
	}

	var session promptpkg.AskSession
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&session); err != nil {
		return promptpkg.AskSession{}, fmt.Errorf("parse JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return promptpkg.AskSession{}, fmt.Errorf("parse JSON: trailing data")
	}

	var err error
	session.InitialPrompt, err = cleanSessionField("initial_prompt", session.InitialPrompt, true)
	if err != nil {
		return promptpkg.AskSession{}, err
	}

	if len(session.Turns) > maxSessionTurns {
		session.Turns = session.Turns[len(session.Turns)-maxSessionTurns:]
	}
	if len(session.Turns) == 0 {
		return promptpkg.AskSession{}, fmt.Errorf("turns must contain at least one command with a question")
	}
	for i := range session.Turns {
		turn := &session.Turns[i]
		turn.Command, err = cleanSessionField(fmt.Sprintf("turns[%d].command", i), turn.Command, true)
		if err != nil {
			return promptpkg.AskSession{}, err
		}
		turn.Risk, err = cleanSessionField(fmt.Sprintf("turns[%d].risk", i), turn.Risk, false)
		if err != nil {
			return promptpkg.AskSession{}, err
		}
		turn.Warning, err = cleanSessionField(fmt.Sprintf("turns[%d].warning", i), turn.Warning, false)
		if err != nil {
			return promptpkg.AskSession{}, err
		}
		turn.Question, err = cleanSessionField(fmt.Sprintf("turns[%d].question", i), turn.Question, false)
		if err != nil {
			return promptpkg.AskSession{}, err
		}
		turn.Answer, err = cleanSessionField(fmt.Sprintf("turns[%d].answer", i), turn.Answer, false)
		if err != nil {
			return promptpkg.AskSession{}, err
		}
	}

	if session.Turns[len(session.Turns)-1].Question == "" {
		return promptpkg.AskSession{}, fmt.Errorf("turns[%d].question is required", len(session.Turns)-1)
	}

	return session, nil
}

func runAskCompletion(outputFormat string, buildPrompt func(shellcontext.Snapshot) promptpkg.Prompt, stdout, stderr io.Writer) int {
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
		fmt.Fprintf(stderr, "ask: %v\n", err)
		return 1
	}

	response := askResponse{Answer: llm.NormalizeAnswer(completion.Command)}
	if response.Answer == "" {
		fmt.Fprintln(stderr, "ask: provider returned an empty answer")
		return 1
	}

	switch outputFormat {
	case "json":
		encoder := json.NewEncoder(stdout)
		if err := encoder.Encode(response); err != nil {
			fmt.Fprintf(stderr, "encode json: %v\n", err)
			return 1
		}
	case "text":
		fmt.Fprintln(stdout, response.Answer)
	}

	return 0
}
