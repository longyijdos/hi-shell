package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func commandParseField(args []string, stdout, stderr io.Writer) int {
	if wantsHelp(args) {
		return printHelp(stdout, parseFieldUsage)
	}
	if len(args) != 2 {
		fmt.Fprintln(stderr, parseFieldUsage)
		return 2
	}

	field := strings.TrimSpace(args[0])
	raw := strings.TrimSpace(args[1])
	if field == "" {
		fmt.Fprintln(stderr, "hi-shell parse-field requires a non-empty field")
		return 2
	}
	if raw == "" {
		fmt.Fprintln(stderr, "hi-shell parse-field requires non-empty json")
		return 2
	}

	return printParsedField(field, raw, stdout, stderr)
}

func commandParseCommand(args []string, stdout, stderr io.Writer) int {
	if wantsHelp(args) {
		return printHelp(stdout, parseCommandUsage)
	}
	if len(args) != 1 {
		fmt.Fprintln(stderr, parseCommandUsage)
		return 2
	}

	raw := strings.TrimSpace(args[0])
	if raw == "" {
		fmt.Fprintln(stderr, "hi-shell parse-command requires non-empty json")
		return 2
	}

	return printParsedField("command", raw, stdout, stderr)
}

func printParsedField(field, raw string, stdout, stderr io.Writer) int {
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
