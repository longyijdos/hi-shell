package cli

import (
	"fmt"
	"io"
	"strings"

	shellcontext "github.com/longyijdos/hi-shell/internal/context"
	promptpkg "github.com/longyijdos/hi-shell/internal/prompt"
)

func commandGenerate(args []string, stdout, stderr io.Writer) int {
	if wantsHelp(args) {
		return printHelp(stdout, generateUsage)
	}

	fs := newFlagSet("generate", generateUsage, stderr)

	var promptText string
	var outputFormat string
	fs.StringVar(&promptText, "prompt", "", "natural language command request")
	fs.StringVar(&outputFormat, "format", "text", "output format: text or json")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	promptText = strings.TrimSpace(promptText)
	if promptText == "" {
		fmt.Fprintln(stderr, `hi-shell generate requires --prompt with non-empty text, for example: hi-shell generate --prompt "list go files"`)
		return 2
	}
	if rejectUnexpectedArgs(stderr, "generate", fs.Args()) {
		return 2
	}
	format, err := parseOutputFormat(outputFormat)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	return runCommandGeneration(format, func(snapshot shellcontext.Snapshot) promptpkg.Prompt {
		return promptpkg.BuildGenerate(promptText, snapshot)
	}, stdout, stderr)
}
