package cli

import (
	"flag"
	"fmt"
	"io"
)

const (
	generateUsage     = `usage: hi-shell generate --prompt <text> [--format text|json]`
	reviseUsage       = `usage: hi-shell revise --session-json <json|-|@file> [--format text|json]`
	askUsage          = `usage: hi-shell ask --session-json <json|-|@file> [--format text|json]`
	configUsage       = `usage: hi-shell config get [key] | hi-shell config set <key> <value> | hi-shell config path`
	configSetUsage    = `usage: hi-shell config set <key> <value>`
	riskUsage         = `usage: hi-shell risk --command <command> [--format text|json]`
	installUsage      = `usage: hi-shell install zsh`
	uninstallUsage    = `usage: hi-shell uninstall [--purge]`
	doctorUsage       = `usage: hi-shell doctor`
	versionUsage      = `usage: hi-shell version`
	parseFieldUsage   = `usage: hi-shell parse-field <field> <json>`
	parseCommandUsage = `usage: hi-shell parse-command <json>`
)

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer, version string) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}

	switch args[0] {
	case "generate":
		return commandGenerate(args[1:], stdout, stderr)
	case "revise":
		return commandRevise(args[1:], stdin, stdout, stderr)
	case "ask":
		return commandAsk(args[1:], stdin, stdout, stderr)
	case "config":
		return commandConfig(args[1:], stdout, stderr)
	case "risk":
		return commandRisk(args[1:], stdout, stderr)
	case "install":
		return commandInstall(args[1:], stdout, stderr)
	case "uninstall":
		return commandUninstall(args[1:], stdout, stderr)
	case "doctor":
		return commandDoctor(args[1:], stdout, stderr, version)
	case "version":
		if wantsHelp(args[1:]) {
			return printHelp(stdout, versionUsage)
		}
		if rejectUnexpectedArgs(stderr, "version", args[1:]) {
			return 2
		}
		fmt.Fprintln(stdout, version)
		return 0
	case "parse-command":
		return commandParseCommand(args[1:], stdout, stderr)
	case "parse-field":
		return commandParseField(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		if rejectUnexpectedArgs(stderr, args[0], args[1:]) {
			return 2
		}
		usage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		usage(stderr)
		return 2
	}
}

func wantsHelp(args []string) bool {
	return len(args) == 1 && isHelpArg(args[0])
}

func isHelpArg(arg string) bool {
	return arg == "help" || arg == "-h" || arg == "--help"
}

func printHelp(stdout io.Writer, usage string) int {
	fmt.Fprintln(stdout, usage)
	return 0
}

func newFlagSet(name, usage string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintln(stderr, usage)
	}
	return fs
}

func rejectUnexpectedArgs(stderr io.Writer, command string, args []string) bool {
	if len(args) == 0 {
		return false
	}
	unexpectedArg(stderr, command, args[0])
	return true
}

func unexpectedArg(stderr io.Writer, command, arg string) int {
	fmt.Fprintf(stderr, "hi-shell %s: unexpected argument %q\n", command, arg)
	return 2
}

func usage(w io.Writer) {
	fmt.Fprintln(w, `hi-shell: tiny AI command composer for zsh

Usage:
  hi-shell generate --prompt "list all files" --format json
  hi-shell revise --session-json - --format json
  hi-shell ask --session-json - --format json
  hi-shell config get [key]
  hi-shell config set <key> <value>
  hi-shell risk --command 'rm -rf /' --format json
  hi-shell install zsh
  hi-shell uninstall [--purge]
  hi-shell doctor
  hi-shell version`)
}
