package risk

import "strings"

func scoreGit(argv []string) Assessment {
	idx, subcommand := gitSubcommand(argv)
	if subcommand == "" {
		return Assessment{Level: Safe}
	}
	args := argv[idx+1:]

	switch subcommand {
	case "status", "log", "diff", "show":
		if containsAny(args, "--output") || hasPrefix(args, "--output=") || containsAny(args, "--ext-diff", "--textconv") {
			return warn("git-output", "This command writes to files or changes project state.")
		}
		return Assessment{Level: Safe}
	case "branch":
		if containsAny(args, "-D", "--delete", "-d", "--move", "-m", "-M", "--copy", "-c", "-C") {
			return warn("git-branch-delete", "Risky command: changes git branches.")
		}
		if len(args) == 0 || containsAny(args, "--list", "-l", "--show-current", "-a", "--all", "-r", "--remotes", "-v", "-vv", "--verbose") || hasPrefix(args, "--format=") {
			return Assessment{Level: Safe}
		}
		return warn("git-branch-change", "This command changes git state.")
	case "reset":
		if containsAny(args, "--hard") {
			return warn("git-reset-hard", "Risky command: may discard git working tree changes.")
		}
		return warn("git-reset", "This command changes git state.")
	case "clean":
		return warn("git-clean", "Risky command: may discard git working tree changes.")
	case "checkout":
		if containsAny(args, "--") {
			return warn("git-checkout-path", "Risky command: may discard git working tree changes.")
		}
		return warn("git-checkout", "This command changes git state.")
	case "restore":
		return warn("git-restore", "Risky command: may discard git working tree changes.")
	case "push":
		if containsAny(args, "--force", "-f", "--force-with-lease") || hasPrefix(args, "--force-with-lease=") {
			return warn("git-push-force", "Risky command: force-pushes git history.")
		}
		return warn("git-push", "This command changes remote git state.")
	case "add", "commit", "switch", "merge", "stash", "pull", "rebase", "cherry-pick", "revert":
		return warn("git-state", "This command changes git state.")
	case "fetch":
		return warn("git-fetch", "This command changes git state.")
	default:
		return Assessment{Level: Safe}
	}
}

func gitSubcommand(argv []string) (int, string) {
	skipNext := false
	for i := 1; i < len(argv); i++ {
		arg := argv[i]
		if skipNext {
			skipNext = false
			continue
		}
		if gitGlobalOptionWithInlineValue(arg) {
			continue
		}
		if gitGlobalOptionWithValue(arg) {
			skipNext = true
			continue
		}
		if arg == "--" || strings.HasPrefix(arg, "-") {
			continue
		}
		return i, arg
	}
	return -1, ""
}

func gitGlobalOptionWithValue(arg string) bool {
	switch arg {
	case "-C", "-c", "--config-env", "--exec-path", "--git-dir", "--namespace", "--super-prefix", "--work-tree":
		return true
	default:
		return false
	}
}

func gitGlobalOptionWithInlineValue(arg string) bool {
	return strings.HasPrefix(arg, "--config-env=") ||
		strings.HasPrefix(arg, "--exec-path=") ||
		strings.HasPrefix(arg, "--git-dir=") ||
		strings.HasPrefix(arg, "--namespace=") ||
		strings.HasPrefix(arg, "--super-prefix=") ||
		strings.HasPrefix(arg, "--work-tree=") ||
		((strings.HasPrefix(arg, "-C") || strings.HasPrefix(arg, "-c")) && len(arg) > 2)
}
