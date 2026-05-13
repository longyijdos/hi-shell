package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	shellcontext "github.com/longyijdos/hi-shell/internal/context"
	promptpkg "github.com/longyijdos/hi-shell/internal/prompt"
)

const (
	maxSessionJSONBytes  = 64 * 1024
	maxSessionTurns      = 8
	maxSessionFieldRunes = 4000
)

func commandRevise(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if wantsHelp(args) {
		return printHelp(stdout, reviseUsage)
	}

	fs := newFlagSet("revise", reviseUsage, stderr)

	var sessionJSON string
	var outputFormat string
	fs.StringVar(&sessionJSON, "session-json", "", "revision session JSON: inline, @file, or - for stdin")
	fs.StringVar(&outputFormat, "format", "text", "output format: text or json")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	sessionJSON = strings.TrimSpace(sessionJSON)
	if sessionJSON == "" {
		fmt.Fprintln(stderr, `hi-shell revise requires --session-json with non-empty JSON, for example: hi-shell revise --session-json -`)
		return 2
	}
	if rejectUnexpectedArgs(stderr, "revise", fs.Args()) {
		return 2
	}
	format, err := parseOutputFormat(outputFormat)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	session, err := readRevisionSession(sessionJSON, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "session-json: %v\n", err)
		return 2
	}

	return runCommandGeneration(format, func(snapshot shellcontext.Snapshot) promptpkg.Prompt {
		return promptpkg.BuildReviseSession(session, snapshot)
	}, stdout, stderr)
}

func readRevisionSession(source string, stdin io.Reader) (promptpkg.ReviseSession, error) {
	data, err := readSessionJSON(source, stdin)
	if err != nil {
		return promptpkg.ReviseSession{}, err
	}

	return parseRevisionSession(data)
}

func readLimited(reader io.Reader, maxBytes int64) ([]byte, error) {
	if reader == nil {
		return nil, fmt.Errorf("input is unavailable")
	}
	data, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("JSON exceeds %d bytes", maxBytes)
	}
	return data, nil
}

func parseRevisionSession(data []byte) (promptpkg.ReviseSession, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return promptpkg.ReviseSession{}, fmt.Errorf("requires non-empty JSON")
	}

	var session promptpkg.ReviseSession
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&session); err != nil {
		return promptpkg.ReviseSession{}, fmt.Errorf("parse JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return promptpkg.ReviseSession{}, fmt.Errorf("parse JSON: trailing data")
	}

	var err error
	session.InitialPrompt, err = cleanSessionField("initial_prompt", session.InitialPrompt, true)
	if err != nil {
		return promptpkg.ReviseSession{}, err
	}

	if len(session.Turns) > maxSessionTurns {
		session.Turns = session.Turns[len(session.Turns)-maxSessionTurns:]
	}
	if len(session.Turns) == 0 {
		return promptpkg.ReviseSession{}, fmt.Errorf("turns must contain at least one command with feedback")
	}
	for i := range session.Turns {
		turn := &session.Turns[i]
		turn.Command, err = cleanSessionField(fmt.Sprintf("turns[%d].command", i), turn.Command, true)
		if err != nil {
			return promptpkg.ReviseSession{}, err
		}
		turn.Risk, err = cleanSessionField(fmt.Sprintf("turns[%d].risk", i), turn.Risk, false)
		if err != nil {
			return promptpkg.ReviseSession{}, err
		}
		turn.Warning, err = cleanSessionField(fmt.Sprintf("turns[%d].warning", i), turn.Warning, false)
		if err != nil {
			return promptpkg.ReviseSession{}, err
		}
		turn.Feedback, err = cleanSessionField(fmt.Sprintf("turns[%d].feedback", i), turn.Feedback, false)
		if err != nil {
			return promptpkg.ReviseSession{}, err
		}
	}

	if session.Turns[len(session.Turns)-1].Feedback == "" {
		return promptpkg.ReviseSession{}, fmt.Errorf("turns[%d].feedback is required", len(session.Turns)-1)
	}

	return session, nil
}

func cleanSessionField(name, value string, required bool) (string, error) {
	value = strings.TrimSpace(value)
	if required && value == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	if utf8.RuneCountInString(value) > maxSessionFieldRunes {
		return "", fmt.Errorf("%s exceeds %d characters", name, maxSessionFieldRunes)
	}
	return value, nil
}
