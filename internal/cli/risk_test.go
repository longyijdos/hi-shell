package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/longyijdos/hi-shell/internal/config"
)

func TestRiskCommandScoresCommand(t *testing.T) {
	hiHome := t.TempDir()
	t.Setenv(config.HomeEnv, hiHome)

	cfg := config.Default()
	if err := config.SaveFile(filepath.Join(hiHome, config.ConfigFileName), cfg); err != nil {
		t.Fatalf("SaveFile() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"risk", "--command", "rm -rf /", "--format", "json"}, strings.NewReader(""), &stdout, &stderr, "test")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var response riskResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Risk != "blocked" {
		t.Fatalf("Risk = %q, want blocked", response.Risk)
	}
	if !response.Blocked {
		t.Fatalf("Blocked = false, want true")
	}
	if !strings.Contains(response.Warning, "recursive delete targets /") {
		t.Fatalf("Warning = %q", response.Warning)
	}
}

func TestRiskCommandTextOutput(t *testing.T) {
	hiHome := t.TempDir()
	t.Setenv(config.HomeEnv, hiHome)

	cfg := config.Default()
	if err := config.SaveFile(filepath.Join(hiHome, config.ConfigFileName), cfg); err != nil {
		t.Fatalf("SaveFile() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"risk", "--command", "echo hi >out.txt"}, strings.NewReader(""), &stdout, &stderr, "test")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	got := stdout.String()
	if !strings.Contains(got, "warn\n") {
		t.Fatalf("stdout = %q, want warn", got)
	}
	if !strings.Contains(got, "writes to files") {
		t.Fatalf("stdout = %q, want write warning", got)
	}
}

func TestRiskRejectsInvalidArgumentsWithoutCreatingConfig(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantStderr string
	}{
		{
			name:       "missing command",
			args:       []string{"risk"},
			wantStderr: "requires --command",
		},
		{
			name:       "unsupported format",
			args:       []string{"risk", "--command", "ls", "--format", "xml"},
			wantStderr: `unsupported format "xml"`,
		},
		{
			name:       "extra args",
			args:       []string{"risk", "--command", "ls", "extra"},
			wantStderr: `unexpected argument "extra"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv(config.HomeEnv, "")

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
				t.Fatalf("invalid risk command created config directory: %v", err)
			}
		})
	}
}
