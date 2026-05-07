package risk

import (
	"regexp"
	"strings"

	"github.com/longyijdos/hi-shell/internal/config"
)

type Level string

const (
	Low      Level = "low"
	Medium   Level = "medium"
	High     Level = "high"
	Critical Level = "critical"
)

type Assessment struct {
	Level   Level
	Warning string
	Blocked bool
}

var (
	spacePattern         = regexp.MustCompile(`\s+`)
	redirectPattern      = regexp.MustCompile(`(^|\s)(>>?|2>)\s*\S+`)
	criticalRMPattern    = regexp.MustCompile(`\brm\s+(-[a-z]*r[a-z]*f[a-z]*|-[a-z]*f[a-z]*r[a-z]*)\s+(--\s+)?(/|/\*|~|~/\*|\$home|\$home/\*)($|\s)`)
	criticalChmodPattern = regexp.MustCompile(`\bchmod\s+-r\s+777\s+(--\s+)?(/|/\*|~|~/\*|\$home|\$home/\*)($|\s)`)
	criticalChownPattern = regexp.MustCompile(`\bchown\s+-r\s+[^ ]+\s+(--\s+)?(/|/\*|~|~/\*|\$home|\$home/\*)($|\s)`)
)

func Score(command string, safety config.SafetyConfig) Assessment {
	normalized := normalize(command)
	if normalized == "" {
		return Assessment{Level: Low}
	}

	if isCritical(normalized) {
		warning := "Blocked critical command."
		if !safety.BlockCritical {
			warning = "Critical command."
		}
		return Assessment{
			Level:   Critical,
			Warning: warning,
			Blocked: safety.BlockCritical,
		}
	}

	if isHigh(normalized, safety) {
		return Assessment{
			Level:   High,
			Warning: "High-risk command. Review carefully before running.",
		}
	}

	if isMedium(normalized) {
		return Assessment{
			Level:   Medium,
			Warning: "This command may change files or local project state.",
		}
	}

	return Assessment{Level: Low}
}

func normalize(command string) string {
	command = strings.TrimSpace(strings.ToLower(command))
	command = strings.ReplaceAll(command, "\t", " ")
	return spacePattern.ReplaceAllString(command, " ")
}

func isCritical(command string) bool {
	if criticalRMPattern.MatchString(command) ||
		criticalChmodPattern.MatchString(command) ||
		criticalChownPattern.MatchString(command) {
		return true
	}

	criticalFragments := []string{
		"mkfs.",
		"dd if=",
		":(){ :|:& };:",
		"> /dev/sda",
		">/dev/sda",
	}
	for _, fragment := range criticalFragments {
		if strings.Contains(command, fragment) {
			return true
		}
	}

	return false
}

func isHigh(command string, safety config.SafetyConfig) bool {
	highFragments := []string{
		"rm -rf ",
		"rm -fr ",
		"chmod -r ",
		"chown -r ",
		"killall ",
		"pkill ",
		"shutdown ",
		"reboot",
		"systemctl stop ",
		"systemctl disable ",
		"systemctl restart ",
	}

	if safety.WarnSudo && hasCommand(command, "sudo") {
		return true
	}

	if safety.WarnDestructive {
		for _, fragment := range highFragments {
			if strings.Contains(command, fragment) {
				return true
			}
		}
	}

	return false
}

func isMedium(command string) bool {
	mediumPrefixes := []string{
		"touch ",
		"mkdir ",
		"mv ",
		"cp ",
		"git add ",
		"git commit ",
		"git checkout ",
		"git switch ",
		"git reset ",
		"git clean ",
		"npm install",
		"npm i ",
		"pnpm add ",
		"yarn add ",
		"pip install ",
		"go get ",
		"go install ",
		"cargo add ",
		"docker compose up",
	}
	for _, prefix := range mediumPrefixes {
		if strings.HasPrefix(command, prefix) || strings.Contains(command, " && "+prefix) || strings.Contains(command, "; "+prefix) {
			return true
		}
	}

	return redirectPattern.MatchString(command) || strings.Contains(command, " tee ")
}

func hasCommand(command, name string) bool {
	return command == name ||
		strings.HasPrefix(command, name+" ") ||
		strings.Contains(command, " && "+name+" ") ||
		strings.Contains(command, "; "+name+" ")
}
