package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/longyijdos/hi-shell/internal/config"
	"github.com/longyijdos/hi-shell/internal/risk"
)

type riskResponse struct {
	Risk    string `json:"risk"`
	Warning string `json:"warning"`
	Blocked bool   `json:"blocked"`
}

func commandRisk(args []string, stdout, stderr io.Writer) int {
	if wantsHelp(args) {
		return printHelp(stdout, riskUsage)
	}

	fs := newFlagSet("risk", riskUsage, stderr)

	var command string
	var outputFormat string
	fs.StringVar(&command, "command", "", "shell command to score")
	fs.StringVar(&outputFormat, "format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if rejectUnexpectedArgs(stderr, "risk", fs.Args()) {
		return 2
	}

	format, err := parseOutputFormat(outputFormat)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	command = strings.TrimSpace(command)
	if command == "" {
		fmt.Fprintln(stderr, `hi-shell risk requires --command with non-empty text, for example: hi-shell risk --command 'rm -rf /'`)
		return 2
	}

	cfg, _, err := config.Load()
	if err != nil {
		fmt.Fprintf(stderr, "risk: %v\n", err)
		return 1
	}

	assessment := risk.Score(command, cfg.Safety)
	response := riskResponse{
		Risk:    string(assessment.Level),
		Warning: assessment.Warning,
		Blocked: assessment.Blocked,
	}

	switch format {
	case "json":
		encoder := json.NewEncoder(stdout)
		if err := encoder.Encode(response); err != nil {
			fmt.Fprintf(stderr, "encode json: %v\n", err)
			return 1
		}
	case "text":
		fmt.Fprintln(stdout, response.Risk)
		if response.Warning != "" {
			fmt.Fprintln(stdout, response.Warning)
		}
	}

	return 0
}
