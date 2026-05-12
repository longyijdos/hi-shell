package risk

import (
	"strings"

	"github.com/longyijdos/hi-shell/internal/config"
)

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
