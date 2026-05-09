package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateRequiresPromptFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"generate", "list", "go", "files"}, strings.NewReader(""), &stdout, &stderr, "test")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "requires --prompt") {
		t.Fatalf("stderr = %q, want prompt error", stderr.String())
	}
	if !strings.Contains(stderr.String(), "hi-shell generate") {
		t.Fatalf("stderr = %q, want hi-shell generate example", stderr.String())
	}
}

func TestGenerateRejectsInvalidArgumentsWithoutCreatingConfig(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantStderr string
	}{
		{
			name:       "extra args",
			args:       []string{"generate", "--prompt", "list go files", "extra"},
			wantStderr: `unexpected argument "extra"`,
		},
		{
			name:       "blank prompt",
			args:       []string{"generate", "--prompt", "   "},
			wantStderr: "requires --prompt",
		},
		{
			name:       "unsupported format",
			args:       []string{"generate", "--prompt", "list go files", "--format", "xml"},
			wantStderr: `unsupported format "xml"`,
		},
		{
			name:       "session flag",
			args:       []string{"generate", "--prompt", "list go files", "--session-json", `{"initial_prompt":"list files"}`},
			wantStderr: "flag provided but not defined: -session-json",
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
				t.Fatalf("invalid generate command created config directory: %v", err)
			}
		})
	}
}
