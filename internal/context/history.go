package shellcontext

import (
	"strings"

	"github.com/longyijdos/hi-shell/internal/config"
)

const (
	historyEnv = "HI_SHELL_HISTORY"
)

var sensitiveHistoryFragments = []string{
	"api_key",
	"api-key",
	"apikey",
	"auth_token",
	"access_token",
	"refresh_token",
	"id_token",
	"token=",
	"token:",
	"secret=",
	"secret:",
	"client_secret",
	"password=",
	"password:",
	"passwd=",
	"authorization:",
	"bearer ",
	"cookie:",
	"set-cookie:",
	"aws_secret_access_key",
	"openai_api_key",
	"deepseek_api_key",
	"github_token",
	"gh_token",
	"npm_token",
	"sshpass",
	"--password",
	"--passwd",
}

func sanitizeHistory(raw string, settings config.HistoryConfig) []string {
	settings = historyConfigWithDefaults(settings)

	var commands []string
	for _, line := range strings.Split(raw, "\n") {
		command := normalizeHistoryCommand(line)
		if command == "" || shouldDropHistoryCommand(command) {
			continue
		}
		commands = append(commands, truncateHistoryCommand(command, settings.MaxCommandChars))
	}

	commands = dedupeHistoryCommands(commands)
	if len(commands) > settings.MaxEntries {
		commands = commands[len(commands)-settings.MaxEntries:]
	}
	for historyBytes(commands) > settings.MaxBytes && len(commands) > 0 {
		commands = commands[1:]
	}
	return commands
}

func historyConfigWithDefaults(settings config.HistoryConfig) config.HistoryConfig {
	defaults := config.Default().History
	if settings.FetchLimit <= 0 {
		settings.FetchLimit = defaults.FetchLimit
	}
	if settings.MaxEntries <= 0 {
		settings.MaxEntries = defaults.MaxEntries
	}
	if settings.MaxCommandChars <= 0 {
		settings.MaxCommandChars = defaults.MaxCommandChars
	}
	if settings.MaxBytes <= 0 {
		settings.MaxBytes = defaults.MaxBytes
	}
	return settings
}

func normalizeHistoryCommand(command string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		return ""
	}

	var b strings.Builder
	for _, r := range command {
		switch {
		case r == '\t':
			b.WriteByte(' ')
		case r < 0x20 || r == 0x7f:
			continue
		default:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

func shouldDropHistoryCommand(command string) bool {
	lower := strings.ToLower(command)
	if strings.HasPrefix(lower, "hi ") || strings.HasPrefix(lower, "hi-shell ") {
		return true
	}

	for _, fragment := range sensitiveHistoryFragments {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	return false
}

func truncateHistoryCommand(command string, maxCommandChars int) string {
	runes := []rune(command)
	if len(runes) <= maxCommandChars {
		return command
	}
	return strings.TrimSpace(string(runes[:maxCommandChars])) + " ..."
}

func dedupeHistoryCommands(commands []string) []string {
	seen := make(map[string]struct{}, len(commands))
	reversed := make([]string, 0, len(commands))
	for i := len(commands) - 1; i >= 0; i-- {
		command := commands[i]
		if _, ok := seen[command]; ok {
			continue
		}
		seen[command] = struct{}{}
		reversed = append(reversed, command)
	}

	deduped := make([]string, 0, len(reversed))
	for i := len(reversed) - 1; i >= 0; i-- {
		deduped = append(deduped, reversed[i])
	}
	return deduped
}

func historyBytes(commands []string) int {
	total := 0
	for _, command := range commands {
		total += len(command) + 2
	}
	return total
}
