package risk

import (
	"path/filepath"
	"strings"
	"unicode"

	"github.com/longyijdos/hi-shell/internal/config"
)

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
