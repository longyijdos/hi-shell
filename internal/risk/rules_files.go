package risk

import (
	"fmt"
	"strings"

	"github.com/longyijdos/hi-shell/internal/config"
	"mvdan.cc/sh/v3/syntax"
)

func scoreRM(argv []string, safety config.SafetyConfig) Assessment {
	recursive, force, targets := rmOptionsAndTargets(argv[1:])
	if len(targets) == 0 {
		return Assessment{Level: Safe}
	}

	if safety.WarnDestructive {
		for _, target := range targets {
			if recursive && isProtectedTarget(target) {
				return critical(safety, "rm-protected-target", fmt.Sprintf("recursive delete targets %s", displayTarget(target)))
			}
		}
		if recursive {
			return warn("rm-recursive", "Risky command: recursively deletes files.")
		}
		if force {
			return warn("rm-force", "Risky command: force-deletes files.")
		}
		return warn("rm-delete", "Risky command: deletes files.")
	}

	for _, target := range targets {
		if recursive && isProtectedTarget(target) {
			return critical(safety, "rm-protected-target", fmt.Sprintf("recursive delete targets %s", displayTarget(target)))
		}
	}
	return Assessment{Level: Safe}
}

func scoreChmod(argv []string, safety config.SafetyConfig) Assessment {
	recursive, args := splitOptions(argv[1:], map[string]bool{})
	if len(args) == 0 {
		return Assessment{Level: Safe}
	}

	mode := args[0]
	targets := args[1:]
	for _, target := range targets {
		if recursive && mode == "777" && isProtectedTarget(target) {
			return critical(safety, "chmod-protected-target", fmt.Sprintf("recursively changes permissions under %s", displayTarget(target)))
		}
	}
	if safety.WarnDestructive {
		return warn("chmod", "Risky command: changes file permissions.")
	}
	return Assessment{Level: Safe}
}

func scoreChown(argv []string, safety config.SafetyConfig) Assessment {
	recursive, args := splitOptions(argv[1:], map[string]bool{})
	if len(args) == 0 {
		return Assessment{Level: Safe}
	}

	targets := args[1:]
	for _, target := range targets {
		if recursive && isProtectedTarget(target) {
			return critical(safety, "chown-protected-target", fmt.Sprintf("recursively changes ownership under %s", displayTarget(target)))
		}
	}
	if safety.WarnDestructive {
		return warn("chown", "Risky command: changes file ownership.")
	}
	return Assessment{Level: Safe}
}

func scoreDD(argv []string, safety config.SafetyConfig) Assessment {
	for _, arg := range argv[1:] {
		if value, ok := strings.CutPrefix(arg, "of="); ok {
			if isDiskDevice(value) {
				return critical(safety, "dd-disk-device", "writes directly to a disk device")
			}
			return warn("dd-write", "This command writes to files or changes project state.")
		}
	}
	return Assessment{Level: Safe}
}

func scoreMkfs(argv []string, safety config.SafetyConfig) Assessment {
	for _, arg := range argv[1:] {
		if isDiskDevice(arg) {
			return critical(safety, "mkfs-disk-device", "formats a disk device")
		}
	}
	if safety.WarnDestructive {
		return warn("mkfs", "Risky command: creates a filesystem.")
	}
	return Assessment{Level: Safe}
}

func scoreFind(argv []string, safety config.SafetyConfig) Assessment {
	if !safety.WarnDestructive {
		return Assessment{Level: Safe}
	}
	for i, arg := range argv[1:] {
		switch arg {
		case "-delete":
			return warn("find-delete", "Risky command: deletes files found by find.")
		case "-exec", "-execdir", "-ok", "-okdir":
			if i+2 < len(argv) && executableName(argv[i+2]) == "rm" {
				return warn("find-exec-rm", "Risky command: deletes files found by find.")
			}
			return warn("find-exec", "Risky command: executes another command from find.")
		case "-fls", "-fprint", "-fprint0", "-fprintf":
			return warn("find-write", "This command writes to files or changes project state.")
		}
	}
	return Assessment{Level: Safe}
}

func scoreRipgrep(argv []string) Assessment {
	for _, arg := range argv[1:] {
		if arg == "--pre" || strings.HasPrefix(arg, "--pre=") || arg == "--hostname-bin" || strings.HasPrefix(arg, "--hostname-bin=") {
			return warn("rg-exec", "Risky command: executes another command from search options.")
		}
	}
	return Assessment{Level: Safe}
}

func scoreSed(argv []string) Assessment {
	for _, arg := range argv[1:] {
		if arg == "-i" || strings.HasPrefix(arg, "-i") || arg == "--in-place" || strings.HasPrefix(arg, "--in-place=") {
			return warn("sed-in-place", "This command writes to files or changes project state.")
		}
	}
	return Assessment{Level: Safe}
}

func scoreBase64(argv []string) Assessment {
	for i, arg := range argv[1:] {
		if arg == "-o" || arg == "--output" || strings.HasPrefix(arg, "--output=") || (strings.HasPrefix(arg, "-o") && arg != "-o") {
			_ = i
			return warn("base64-output", "This command writes to files or changes project state.")
		}
	}
	return Assessment{Level: Safe}
}

func scoreCurl(argv []string) Assessment {
	for i := 1; i < len(argv); i++ {
		arg := argv[i]
		if arg == "-o" || arg == "--output" || strings.HasPrefix(arg, "--output=") || arg == "-O" || arg == "--remote-name" {
			return warn("curl-output", "This command writes to files or changes project state.")
		}
	}
	return Assessment{Level: Safe}
}

func scoreWget(argv []string) Assessment {
	for i := 1; i < len(argv); i++ {
		arg := argv[i]
		if arg == "-O" || arg == "--output-document" || strings.HasPrefix(arg, "--output-document=") {
			return warn("wget-output", "This command writes to files or changes project state.")
		}
	}
	return Assessment{Level: Safe}
}

func scoreRedirect(redir *syntax.Redirect, safety config.SafetyConfig) Assessment {
	if redir == nil || !redirectWrites(redir.Op) {
		return Assessment{Level: Safe}
	}
	target, ok := wordStaticValue(redir.Word)
	if !ok {
		return warn("redirect-write", "This command writes to files or changes project state.")
	}
	if isNullDevice(target) {
		return Assessment{Level: Safe}
	}
	if isDiskDevice(target) {
		return critical(safety, "redirect-disk-device", "writes directly to a disk device")
	}
	return warn("redirect-write", "This command writes to files or changes project state.")
}

func rmOptionsAndTargets(args []string) (bool, bool, []string) {
	recursive := false
	force := false
	var targets []string

	for _, arg := range args {
		if arg == "" {
			continue
		}
		if arg == "--" {
			continue
		}
		if strings.HasPrefix(arg, "--") {
			switch arg {
			case "--recursive", "--dir":
				recursive = true
			case "--force":
				force = true
			default:
				if !strings.HasPrefix(arg, "--one-file-system") && !strings.HasPrefix(arg, "--preserve-root") && !strings.HasPrefix(arg, "--no-preserve-root") {
					targets = append(targets, arg)
				}
			}
			continue
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			for _, r := range arg[1:] {
				switch r {
				case 'r', 'R':
					recursive = true
				case 'f':
					force = true
				}
			}
			continue
		}
		targets = append(targets, arg)
	}

	return recursive, force, targets
}

func splitOptions(args []string, optionsWithValues map[string]bool) (bool, []string) {
	recursive := false
	var rest []string
	stopOptions := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if stopOptions {
			rest = append(rest, arg)
			continue
		}
		if arg == "--" {
			stopOptions = true
			continue
		}
		if strings.HasPrefix(arg, "--") {
			if arg == "--recursive" {
				recursive = true
			}
			if optionsWithValues[arg] {
				i++
			}
			continue
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			for _, r := range arg[1:] {
				if r == 'R' {
					recursive = true
				}
			}
			continue
		}
		rest = append(rest, arg)
	}

	return recursive, rest
}
