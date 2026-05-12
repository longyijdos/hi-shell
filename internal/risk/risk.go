package risk

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/longyijdos/hi-shell/internal/config"
	"mvdan.cc/sh/v3/syntax"
)

type Level string

const (
	Safe    Level = "safe"
	Warn    Level = "warn"
	Blocked Level = "blocked"
)

type Assessment struct {
	Level   Level
	Warning string
	Blocked bool
	RuleID  string
}

func Score(command string, safety config.SafetyConfig) Assessment {
	command = strings.TrimSpace(command)
	if command == "" {
		return Assessment{Level: Safe}
	}

	if isForkBomb(command) {
		return critical(safety, "fork-bomb", "fork bomb pattern")
	}

	file, err := syntax.NewParser().Parse(strings.NewReader(command), "")
	if err != nil {
		return warn("parse-error", "Risky command: contains shell syntax hi-shell could not analyze.")
	}

	return scoreStmts(file.Stmts, safety)
}

func scoreStmts(stmts []*syntax.Stmt, safety config.SafetyConfig) Assessment {
	assessment := Assessment{Level: Safe}
	for _, stmt := range stmts {
		assessment = combine(assessment, scoreStmt(stmt, safety))
	}
	return assessment
}

func scoreStmt(stmt *syntax.Stmt, safety config.SafetyConfig) Assessment {
	if stmt == nil {
		return Assessment{Level: Safe}
	}

	assessment := Assessment{Level: Safe}
	for _, redir := range stmt.Redirs {
		assessment = combine(assessment, scoreRedirect(redir, safety))
	}
	return combine(assessment, scoreCommand(stmt.Cmd, safety))
}

func scoreCommand(cmd syntax.Command, safety config.SafetyConfig) Assessment {
	switch cmd := cmd.(type) {
	case nil:
		return Assessment{Level: Safe}
	case *syntax.CallExpr:
		return scoreCall(cmd, safety)
	case *syntax.BinaryCmd:
		assessment := scorePipelineRemoteScript(cmd)
		assessment = combine(assessment, scoreStmt(cmd.X, safety))
		assessment = combine(assessment, scoreStmt(cmd.Y, safety))
		return assessment
	case *syntax.Subshell:
		return scoreStmts(cmd.Stmts, safety)
	case *syntax.Block:
		return scoreStmts(cmd.Stmts, safety)
	case *syntax.IfClause:
		return scoreIfClause(cmd, safety)
	case *syntax.WhileClause:
		return combine(scoreStmts(cmd.Cond, safety), scoreStmts(cmd.Do, safety))
	case *syntax.ForClause:
		return scoreStmts(cmd.Do, safety)
	case *syntax.FuncDecl:
		return warn("function-declaration", "Risky command: defines shell code.")
	case *syntax.TimeClause:
		return scoreStmt(cmd.Stmt, safety)
	case *syntax.DeclClause, *syntax.LetClause, *syntax.ArithmCmd, *syntax.TestClause, *syntax.TestDecl:
		return Assessment{Level: Safe}
	default:
		return warn("complex-shell", "Risky command: uses complex shell syntax.")
	}
}

func scoreIfClause(clause *syntax.IfClause, safety config.SafetyConfig) Assessment {
	if clause == nil {
		return Assessment{Level: Safe}
	}

	assessment := combine(scoreStmts(clause.Cond, safety), scoreStmts(clause.Then, safety))
	return combine(assessment, scoreIfClause(clause.Else, safety))
}

func scoreCall(call *syntax.CallExpr, safety config.SafetyConfig) Assessment {
	if len(call.Args) == 0 {
		return Assessment{Level: Safe}
	}

	assessment := Assessment{Level: Safe}
	argv := make([]string, 0, len(call.Args))
	for _, word := range call.Args {
		assessment = combine(assessment, scoreWordExecutions(word, safety))

		value, ok := wordStaticValue(word)
		if !ok {
			if len(argv) == 0 {
				assessment = combine(assessment, warn("dynamic-command", "Risky command: uses dynamic shell execution."))
			}
			value = ""
		}
		argv = append(argv, value)
	}

	return combine(assessment, scoreArgv(argv, safety))
}

func scoreArgv(argv []string, safety config.SafetyConfig) Assessment {
	argv = trimEmpty(argv)
	if len(argv) == 0 {
		return Assessment{Level: Safe}
	}

	name := executableName(argv[0])
	switch name {
	case "sudo":
		return scorePrivilegeWrapper(argv, safety, "sudo", "Risky command: uses sudo.")
	case "doas":
		return scorePrivilegeWrapper(argv, safety, "doas", "Risky command: uses privilege escalation.")
	case "su":
		return scoreSu(argv, safety)
	case "command":
		return scoreArgv(argv[1:], safety)
	case "env":
		return scoreEnv(argv, safety)
	case "sh", "bash", "zsh":
		return scoreShell(argv, safety)
	case "eval":
		return scoreEval(argv, safety)
	case "source", ".":
		return warn("source-script", "Risky command: sources shell code.")
	case "rm":
		return scoreRM(argv, safety)
	case "chmod":
		return scoreChmod(argv, safety)
	case "chown", "chgrp":
		return scoreChown(argv, safety)
	case "dd":
		return scoreDD(argv, safety)
	case "mkfs", "newfs":
		return scoreMkfs(argv, safety)
	case "find":
		return scoreFind(argv, safety)
	case "rg":
		return scoreRipgrep(argv)
	case "sed":
		return scoreSed(argv)
	case "base64":
		return scoreBase64(argv)
	case "git":
		return scoreGit(argv)
	case "kill", "killall", "pkill", "shutdown", "reboot", "halt", "poweroff":
		return warn("process-service", "Risky command: may stop processes or the system.")
	case "systemctl":
		return scoreSystemctl(argv)
	case "launchctl":
		return scoreLaunchctl(argv)
	case "docker":
		return scoreDocker(argv)
	case "podman":
		return scorePodman(argv)
	case "kubectl":
		return scoreKubectl(argv)
	case "helm":
		return scoreHelm(argv)
	case "terraform":
		return scoreTerraform(argv)
	case "pulumi":
		return scorePulumi(argv)
	case "apt", "apt-get", "dnf", "yum":
		return scoreSystemPackageManager(argv)
	case "brew":
		return scoreBrew(argv)
	case "npm", "pnpm", "yarn":
		return scoreNodePackageManager(argv)
	case "pip", "pip3":
		return scorePip(argv)
	case "go":
		return scoreGo(argv)
	case "cargo":
		return scoreCargo(argv)
	case "bundle":
		return scoreBundle(argv)
	case "curl":
		return scoreCurl(argv)
	case "wget":
		return scoreWget(argv)
	case "tee", "touch", "mkdir", "cp", "mv", "ln":
		return warn("file-write", "This command writes to files or changes project state.")
	default:
		return Assessment{Level: Safe}
	}
}

func scorePrivilegeWrapper(argv []string, safety config.SafetyConfig, ruleID, warning string) Assessment {
	idx := wrapperCommandIndex(argv, map[string]bool{
		"-C": true, "-T": true, "-U": true, "-g": true, "-h": true, "-p": true, "-t": true, "-u": true,
		"--askpass": true, "--close-from": true, "--group": true, "--host": true, "--prompt": true, "--user": true,
	})
	inner := Assessment{Level: Safe}
	if idx > 0 && idx < len(argv) {
		inner = scoreArgv(argv[idx:], safety)
	}
	if !safety.WarnSudo {
		return inner
	}
	return combine(warn(ruleID, warning), inner)
}

func scoreSu(argv []string, safety config.SafetyConfig) Assessment {
	assessment := Assessment{Level: Safe}
	if safety.WarnSudo {
		assessment = warn("su", "Risky command: uses privilege escalation.")
	}
	if script, ok := optionValue(argv[1:], "-c", "--command"); ok {
		assessment = combine(assessment, Score(script, safety))
	}
	return assessment
}

func scoreEnv(argv []string, safety config.SafetyConfig) Assessment {
	for i := 1; i < len(argv); i++ {
		arg := argv[i]
		if arg == "" {
			continue
		}
		if arg == "--" {
			if i+1 < len(argv) {
				return scoreArgv(argv[i+1:], safety)
			}
			return Assessment{Level: Safe}
		}
		if strings.Contains(arg, "=") && !strings.HasPrefix(arg, "-") {
			continue
		}
		if strings.HasPrefix(arg, "-") {
			if arg == "-u" || arg == "--unset" || arg == "-C" || arg == "--chdir" {
				i++
			}
			continue
		}
		return scoreArgv(argv[i:], safety)
	}
	return Assessment{Level: Safe}
}

func scoreShell(argv []string, safety config.SafetyConfig) Assessment {
	if script, ok := shellCommandString(argv); ok {
		return Score(script, safety)
	}
	if len(argv) > 1 {
		return warn("shell-script", "Risky command: runs a shell script.")
	}
	return Assessment{Level: Safe}
}

func scoreEval(argv []string, safety config.SafetyConfig) Assessment {
	if len(argv) <= 1 {
		return warn("eval", "Risky command: uses eval.")
	}

	script := strings.Join(argv[1:], " ")
	assessment := Score(script, safety)
	if assessment.Level == Safe {
		return Assessment{Level: Safe}
	}
	return combine(warn("eval", "Risky command: uses eval."), assessment)
}

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

func scoreSystemctl(argv []string) Assessment {
	if len(argv) > 1 && containsAny([]string{argv[1]}, "stop", "restart", "disable", "kill", "poweroff", "reboot") {
		return warn("systemctl", "Risky command: controls system services.")
	}
	return Assessment{Level: Safe}
}

func scoreLaunchctl(argv []string) Assessment {
	if len(argv) > 1 && containsAny([]string{argv[1]}, "unload", "bootout", "remove", "disable", "kickstart") {
		return warn("launchctl", "Risky command: controls system services.")
	}
	return Assessment{Level: Safe}
}

func scoreDocker(argv []string) Assessment {
	if len(argv) <= 1 {
		return Assessment{Level: Safe}
	}
	switch argv[1] {
	case "rm", "rmi":
		return warn("docker-remove", "Risky command: changes containers or infrastructure.")
	case "system":
		if len(argv) > 2 && argv[2] == "prune" {
			return warn("docker-prune", "Risky command: changes containers or infrastructure.")
		}
	case "compose":
		return scoreDockerCompose(argv[2:])
	}
	return Assessment{Level: Safe}
}

func scoreDockerCompose(args []string) Assessment {
	if len(args) == 0 {
		return Assessment{Level: Safe}
	}
	switch args[0] {
	case "down":
		if containsAny(args[1:], "-v", "--volumes") {
			return warn("docker-compose-down-volumes", "Risky command: changes containers or infrastructure.")
		}
		return warn("docker-compose-down", "Risky command: changes containers or infrastructure.")
	case "up":
		return warn("docker-compose-up", "This command writes to files or changes project state.")
	}
	return Assessment{Level: Safe}
}

func scorePodman(argv []string) Assessment {
	if len(argv) <= 1 {
		return Assessment{Level: Safe}
	}
	if containsAny([]string{argv[1]}, "rm", "rmi") || (argv[1] == "system" && len(argv) > 2 && argv[2] == "prune") {
		return warn("podman-change", "Risky command: changes containers or infrastructure.")
	}
	return Assessment{Level: Safe}
}

func scoreKubectl(argv []string) Assessment {
	if len(argv) > 1 && containsAny([]string{argv[1]}, "delete", "apply", "replace", "scale", "patch", "rollout") {
		return warn("kubectl-change", "Risky command: changes containers or infrastructure.")
	}
	return Assessment{Level: Safe}
}

func scoreHelm(argv []string) Assessment {
	if len(argv) > 1 && containsAny([]string{argv[1]}, "uninstall", "upgrade", "install", "rollback") {
		return warn("helm-change", "Risky command: changes containers or infrastructure.")
	}
	return Assessment{Level: Safe}
}

func scoreTerraform(argv []string) Assessment {
	if len(argv) > 1 && containsAny([]string{argv[1]}, "apply", "destroy") {
		return warn("terraform-change", "Risky command: changes containers or infrastructure.")
	}
	return Assessment{Level: Safe}
}

func scorePulumi(argv []string) Assessment {
	if len(argv) > 1 && containsAny([]string{argv[1]}, "up", "destroy") {
		return warn("pulumi-change", "Risky command: changes containers or infrastructure.")
	}
	return Assessment{Level: Safe}
}

func scoreSystemPackageManager(argv []string) Assessment {
	if len(argv) <= 1 {
		return Assessment{Level: Safe}
	}
	switch argv[1] {
	case "remove", "purge", "autoremove":
		return warn("package-remove", "Risky command: removes installed packages.")
	case "install", "update", "upgrade":
		return warn("package-install", "This command changes dependencies.")
	}
	return Assessment{Level: Safe}
}

func scoreBrew(argv []string) Assessment {
	if len(argv) > 1 && containsAny([]string{argv[1]}, "install", "uninstall", "upgrade", "update") {
		return warn("brew-change", "This command changes dependencies.")
	}
	return Assessment{Level: Safe}
}

func scoreNodePackageManager(argv []string) Assessment {
	if len(argv) <= 1 {
		return Assessment{Level: Safe}
	}
	if argv[0] == "npm" && argv[1] == "run" {
		if len(argv) > 2 && containsAny([]string{argv[2]}, "dev", "start") {
			return warn("npm-run", "This command starts or changes local project state.")
		}
		return Assessment{Level: Safe}
	}
	if containsAny([]string{argv[1]}, "install", "i", "add", "update", "uninstall", "remove") {
		return warn("node-package-change", "This command changes dependencies.")
	}
	return Assessment{Level: Safe}
}

func scorePip(argv []string) Assessment {
	if len(argv) > 1 && containsAny([]string{argv[1]}, "install", "uninstall") {
		return warn("pip-change", "This command changes dependencies.")
	}
	return Assessment{Level: Safe}
}

func scoreGo(argv []string) Assessment {
	if len(argv) > 1 && containsAny([]string{argv[1]}, "get", "install", "run") {
		return warn("go-change", "This command writes to files or changes project state.")
	}
	return Assessment{Level: Safe}
}

func scoreCargo(argv []string) Assessment {
	if len(argv) > 1 && containsAny([]string{argv[1]}, "add", "install", "update") {
		return warn("cargo-change", "This command changes dependencies.")
	}
	return Assessment{Level: Safe}
}

func scoreBundle(argv []string) Assessment {
	if len(argv) > 1 && argv[1] == "install" {
		return warn("bundle-install", "This command changes dependencies.")
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

func scorePipelineRemoteScript(binary *syntax.BinaryCmd) Assessment {
	if binary == nil || (binary.Op != syntax.Pipe && binary.Op != syntax.PipeAll) {
		return Assessment{Level: Safe}
	}
	if stmtStartsShell(binary.Y) {
		if stmtStartsDownloader(binary.X) {
			return warn("remote-script", "Risky command: executes a remote script.")
		}
		return warn("shell-stdin", "Risky command: executes shell input.")
	}
	return Assessment{Level: Safe}
}

func scoreWordExecutions(word *syntax.Word, safety config.SafetyConfig) Assessment {
	if word == nil {
		return Assessment{Level: Safe}
	}
	assessment := Assessment{Level: Safe}
	for _, part := range word.Parts {
		assessment = combine(assessment, scoreWordPartExecutions(part, safety))
	}
	return assessment
}

func scoreWordPartExecutions(part syntax.WordPart, safety config.SafetyConfig) Assessment {
	switch part := part.(type) {
	case *syntax.CmdSubst:
		return scoreStmts(part.Stmts, safety)
	case *syntax.ProcSubst:
		return scoreStmts(part.Stmts, safety)
	case *syntax.DblQuoted:
		assessment := Assessment{Level: Safe}
		for _, nested := range part.Parts {
			assessment = combine(assessment, scoreWordPartExecutions(nested, safety))
		}
		return assessment
	default:
		return Assessment{Level: Safe}
	}
}

func wordStaticValue(word *syntax.Word) (string, bool) {
	if word == nil {
		return "", false
	}
	var b strings.Builder
	for _, part := range word.Parts {
		value, ok := wordPartStaticValue(part)
		if !ok {
			return "", false
		}
		b.WriteString(value)
	}
	return b.String(), true
}

func wordPartStaticValue(part syntax.WordPart) (string, bool) {
	switch part := part.(type) {
	case *syntax.Lit:
		return part.Value, true
	case *syntax.SglQuoted:
		return part.Value, true
	case *syntax.DblQuoted:
		var b strings.Builder
		for _, nested := range part.Parts {
			value, ok := wordPartStaticValue(nested)
			if !ok {
				return "", false
			}
			b.WriteString(value)
		}
		return b.String(), true
	case *syntax.ParamExp:
		if part.Param != nil && strings.EqualFold(part.Param.Value, "HOME") && part.Exp == nil && part.Index == nil {
			return "$HOME", true
		}
		return "", false
	default:
		return "", false
	}
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

func wrapperCommandIndex(argv []string, optionsWithValues map[string]bool) int {
	for i := 1; i < len(argv); i++ {
		arg := argv[i]
		if arg == "--" {
			return i + 1
		}
		if strings.HasPrefix(arg, "--") {
			if strings.Contains(arg, "=") {
				continue
			}
			if optionsWithValues[arg] {
				i++
			}
			continue
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			if len(arg) == 2 && optionsWithValues[arg] {
				i++
			}
			continue
		}
		return i
	}
	return len(argv)
}

func shellCommandString(argv []string) (string, bool) {
	for i := 1; i < len(argv); i++ {
		arg := argv[i]
		if !strings.HasPrefix(arg, "-") {
			continue
		}
		if strings.Contains(arg, "c") {
			if i+1 < len(argv) {
				return argv[i+1], true
			}
			return "", false
		}
	}
	return "", false
}

func optionValue(args []string, short, long string) (string, bool) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == short || arg == long {
			if i+1 < len(args) {
				return args[i+1], true
			}
			return "", false
		}
		if strings.HasPrefix(arg, long+"=") {
			return strings.TrimPrefix(arg, long+"="), true
		}
	}
	return "", false
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

func stmtStartsDownloader(stmt *syntax.Stmt) bool {
	argv := stmtArgv(stmt)
	if len(argv) == 0 {
		return false
	}
	name := executableName(argv[0])
	return name == "curl" || name == "wget"
}

func stmtStartsShell(stmt *syntax.Stmt) bool {
	argv := stmtArgv(stmt)
	argv = unwrapSimpleWrappers(argv)
	if len(argv) == 0 {
		return false
	}
	switch executableName(argv[0]) {
	case "sh", "bash", "zsh":
		return true
	default:
		return false
	}
}

func stmtArgv(stmt *syntax.Stmt) []string {
	call, ok := stmt.Cmd.(*syntax.CallExpr)
	if !ok {
		return nil
	}
	var argv []string
	for _, word := range call.Args {
		value, ok := wordStaticValue(word)
		if !ok {
			return nil
		}
		argv = append(argv, value)
	}
	return trimEmpty(argv)
}

func unwrapSimpleWrappers(argv []string) []string {
	for len(argv) > 0 {
		switch executableName(argv[0]) {
		case "sudo", "doas", "command":
			idx := wrapperCommandIndex(argv, nil)
			if idx <= 0 || idx >= len(argv) {
				return nil
			}
			argv = argv[idx:]
		case "env":
			for i := 1; i < len(argv); i++ {
				if strings.Contains(argv[i], "=") && !strings.HasPrefix(argv[i], "-") {
					continue
				}
				argv = argv[i:]
				break
			}
		default:
			return argv
		}
	}
	return argv
}

func redirectWrites(op syntax.RedirOperator) bool {
	switch op {
	case syntax.RdrOut, syntax.AppOut, syntax.RdrInOut, syntax.DplOut, syntax.ClbOut, syntax.RdrAll, syntax.AppAll:
		return true
	default:
		return false
	}
}

func executableName(raw string) string {
	name := strings.ToLower(filepath.Base(raw))
	for _, suffix := range []string{".exe", ".cmd", ".bat", ".com"} {
		name = strings.TrimSuffix(name, suffix)
	}
	if strings.HasPrefix(name, "mkfs.") || strings.HasPrefix(name, "newfs_") {
		return "mkfs"
	}
	return name
}

func isForkBomb(command string) bool {
	compact := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, command)
	return strings.Contains(compact, ":(){:|:&};:")
}

func isProtectedTarget(target string) bool {
	target = normalizeTarget(target)
	switch target {
	case "/", "/*", "~", "~/", "~/*", "$home", "$home/", "$home/*":
		return true
	default:
		return false
	}
}

func displayTarget(target string) string {
	target = normalizeTarget(target)
	switch target {
	case "/", "/*":
		return "/"
	case "~", "~/", "~/*", "$home", "$home/", "$home/*":
		return "home"
	default:
		return target
	}
}

func normalizeTarget(target string) string {
	target = strings.TrimSpace(strings.ToLower(target))
	target = strings.ReplaceAll(target, "${home}", "$home")
	return target
}

func isDiskDevice(path string) bool {
	path = strings.ToLower(strings.TrimSpace(path))
	if isNullDevice(path) || !strings.HasPrefix(path, "/dev/") {
		return false
	}
	base := strings.TrimPrefix(path, "/dev/")
	return strings.HasPrefix(base, "sd") ||
		strings.HasPrefix(base, "hd") ||
		strings.HasPrefix(base, "xvd") ||
		strings.HasPrefix(base, "vd") ||
		strings.HasPrefix(base, "nvme") ||
		strings.HasPrefix(base, "mmcblk") ||
		strings.HasPrefix(base, "disk") ||
		strings.HasPrefix(base, "rdisk")
}

func isNullDevice(path string) bool {
	switch strings.ToLower(strings.TrimSpace(path)) {
	case "/dev/null", "/dev/stdout", "/dev/stderr", "/dev/stdin", "/dev/fd/1", "/dev/fd/2":
		return true
	default:
		return false
	}
}

func trimEmpty(items []string) []string {
	out := items[:0]
	for _, item := range items {
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func containsAny(items []string, candidates ...string) bool {
	for _, item := range items {
		for _, candidate := range candidates {
			if item == candidate {
				return true
			}
		}
	}
	return false
}

func hasPrefix(items []string, prefix string) bool {
	for _, item := range items {
		if strings.HasPrefix(item, prefix) {
			return true
		}
	}
	return false
}

func warn(ruleID, warning string) Assessment {
	return Assessment{
		Level:   Warn,
		Warning: warning,
		RuleID:  ruleID,
	}
}

func critical(safety config.SafetyConfig, ruleID, reason string) Assessment {
	if safety.BlockCritical {
		return Assessment{
			Level:   Blocked,
			Warning: "Blocked critical command: " + reason + ".",
			Blocked: true,
			RuleID:  ruleID,
		}
	}
	return Assessment{
		Level:   Warn,
		Warning: "Critical command: " + reason + ".",
		RuleID:  ruleID,
	}
}

func combine(left, right Assessment) Assessment {
	if rank(right.Level) > rank(left.Level) {
		return right
	}
	if rank(right.Level) == rank(left.Level) && left.Warning == "" && right.Warning != "" {
		return right
	}
	return left
}

func rank(level Level) int {
	switch level {
	case Blocked:
		return 2
	case Warn:
		return 1
	default:
		return 0
	}
}
