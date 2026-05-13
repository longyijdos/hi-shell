package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRejectsInvalidArguments(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantStderr string
	}{
		{
			name:       "config get extra args",
			args:       []string{"config", "get", "provider", "extra"},
			wantStderr: `unexpected argument "extra"`,
		},
		{
			name:       "config set extra args",
			args:       []string{"config", "set", "provider", "openai", "extra"},
			wantStderr: `unexpected argument "extra"`,
		},
		{
			name:       "config set blank value",
			args:       []string{"config", "set", "provider", "   "},
			wantStderr: "requires a non-empty value",
		},
		{
			name:       "config path extra args",
			args:       []string{"config", "path", "extra"},
			wantStderr: `unexpected argument "extra"`,
		},
		{
			name:       "install extra args",
			args:       []string{"install", "zsh", "extra"},
			wantStderr: "usage: hi-shell install zsh",
		},
		{
			name:       "uninstall extra args",
			args:       []string{"uninstall", "extra"},
			wantStderr: `unexpected argument "extra"`,
		},
		{
			name:       "doctor extra args",
			args:       []string{"doctor", "extra"},
			wantStderr: `unexpected argument "extra"`,
		},
		{
			name:       "risk missing command",
			args:       []string{"risk"},
			wantStderr: "requires --command",
		},
		{
			name:       "ask missing session json",
			args:       []string{"ask"},
			wantStderr: "requires --session-json",
		},
		{
			name:       "version extra args",
			args:       []string{"version", "extra"},
			wantStderr: `unexpected argument "extra"`,
		},
		{
			name:       "parse-field extra args",
			args:       []string{"parse-field", "command", `{"command":"ls"}`, "extra"},
			wantStderr: "usage: hi-shell parse-field",
		},
		{
			name:       "parse-command extra args",
			args:       []string{"parse-command", `{"command":"ls"}`, "extra"},
			wantStderr: "usage: hi-shell parse-command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("HI_SHELL_HOME", "")

			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(tt.args, strings.NewReader(""), &stdout, &stderr, "test")
			if code != 2 {
				t.Fatalf("exit code = %d, want 2; stderr = %q", code, stderr.String())
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			if !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tt.wantStderr)
			}
			_, err := os.Stat(filepath.Join(home, ".hi-shell"))
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("invalid command created config directory: %v", err)
			}
		})
	}
}

func TestRunSubcommandHelpReturnsZero(t *testing.T) {
	subcommands := map[string]string{
		"generate":      generateUsage,
		"revise":        reviseUsage,
		"ask":           askUsage,
		"config":        configUsage,
		"risk":          riskUsage,
		"install":       installUsage,
		"uninstall":     uninstallUsage,
		"doctor":        doctorUsage,
		"version":       versionUsage,
		"parse-field":   parseFieldUsage,
		"parse-command": parseCommandUsage,
	}
	helpArgs := []string{"help", "-h", "--help"}

	for subcommand, wantStdout := range subcommands {
		for _, helpArg := range helpArgs {
			t.Run(subcommand+" "+helpArg, func(t *testing.T) {
				home := t.TempDir()
				t.Setenv("HOME", home)
				t.Setenv("HI_SHELL_HOME", "")

				var stdout bytes.Buffer
				var stderr bytes.Buffer

				code := Run([]string{subcommand, helpArg}, strings.NewReader(""), &stdout, &stderr, "test")
				if code != 0 {
					t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
				}
				if stderr.Len() != 0 {
					t.Fatalf("stderr = %q, want empty", stderr.String())
				}
				if !strings.Contains(stdout.String(), wantStdout) {
					t.Fatalf("stdout = %q, want %q", stdout.String(), wantStdout)
				}
				_, err := os.Stat(filepath.Join(home, ".hi-shell"))
				if !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("help command created config directory: %v", err)
				}
			})
		}
	}
}
