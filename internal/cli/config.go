package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/longyijdos/hi-shell/internal/config"
)

func commandConfig(args []string, stdout, stderr io.Writer) int {
	if wantsHelp(args) {
		return printHelp(stdout, configUsage)
	}
	if len(args) == 0 {
		fmt.Fprintln(stderr, configUsage)
		return 2
	}

	subcommand := args[0]
	switch subcommand {
	case "get":
		if len(args) > 2 {
			return unexpectedArg(stderr, "config get", args[2])
		}
		if len(args) == 2 && strings.TrimSpace(args[1]) == "" {
			fmt.Fprintln(stderr, "hi-shell config get requires a non-empty key")
			return 2
		}
	case "set":
		if len(args) < 3 {
			fmt.Fprintln(stderr, configSetUsage)
			return 2
		}
		if len(args) > 3 {
			return unexpectedArg(stderr, "config set", args[3])
		}
		if strings.TrimSpace(args[1]) == "" {
			fmt.Fprintln(stderr, "hi-shell config set requires a non-empty key")
			return 2
		}
		if strings.TrimSpace(args[2]) == "" {
			fmt.Fprintln(stderr, "hi-shell config set requires a non-empty value")
			return 2
		}
	case "path":
		if len(args) > 1 {
			return unexpectedArg(stderr, "config path", args[1])
		}
	default:
		fmt.Fprintf(stderr, "unknown config command %q\n", subcommand)
		return 2
	}

	cfg, path, err := config.Ensure()
	if err != nil {
		fmt.Fprintf(stderr, "config: %v\n", err)
		return 1
	}

	switch subcommand {
	case "get":
		if len(args) == 1 {
			data, err := config.Marshal(cfg)
			if err != nil {
				fmt.Fprintf(stderr, "config: %v\n", err)
				return 1
			}
			fmt.Fprint(stdout, string(data))
			return 0
		}
		value, err := config.Get(cfg, strings.TrimSpace(args[1]))
		if err != nil {
			fmt.Fprintf(stderr, "config: %v\n", err)
			return 2
		}
		fmt.Fprintln(stdout, value)
		return 0
	case "set":
		key := strings.TrimSpace(args[1])
		value := strings.TrimSpace(args[2])
		if err := config.Set(&cfg, key, value); err != nil {
			if errors.Is(err, config.ErrSecretNotStored) {
				fmt.Fprintln(stderr, err)
				return 2
			}
			fmt.Fprintf(stderr, "config: %v\n", err)
			return 2
		}
		if err := config.SaveFile(path, cfg); err != nil {
			fmt.Fprintf(stderr, "config: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "set %s\n", key)
		return 0
	case "path":
		fmt.Fprintln(stdout, path)
		return 0
	}

	return 0
}
