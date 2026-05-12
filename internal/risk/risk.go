package risk

import (
	"strings"

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
