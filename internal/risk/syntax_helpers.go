package risk

import (
	"strings"

	"github.com/longyijdos/hi-shell/internal/config"
	"mvdan.cc/sh/v3/syntax"
)

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
