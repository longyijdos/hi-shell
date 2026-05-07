package risk

import (
	"testing"

	"github.com/longyijdos/hi-shell/internal/config"
)

func TestScore(t *testing.T) {
	safety := config.Default().Safety

	tests := []struct {
		command string
		level   Level
		blocked bool
	}{
		{command: "find . -type f", level: Low},
		{command: "git checkout -b feature-x", level: Medium},
		{command: "sudo apt install postgresql", level: High},
		{command: "rm -rf node_modules", level: High},
		{command: "rm -rf /", level: Critical, blocked: true},
		{command: "rm -fr /", level: Critical, blocked: true},
		{command: "chmod -R 777 /", level: Critical, blocked: true},
		{command: "echo hi >out.txt", level: Medium},
	}

	for _, tt := range tests {
		got := Score(tt.command, safety)
		if got.Level != tt.level {
			t.Fatalf("Score(%q).Level = %q, want %q", tt.command, got.Level, tt.level)
		}
		if got.Blocked != tt.blocked {
			t.Fatalf("Score(%q).Blocked = %v, want %v", tt.command, got.Blocked, tt.blocked)
		}
	}
}
