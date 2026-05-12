package risk

import (
	"strings"
	"testing"

	"github.com/longyijdos/hi-shell/internal/config"
)

func TestScore(t *testing.T) {
	safety := config.Default().Safety

	tests := []struct {
		name        string
		command     string
		level       Level
		blocked     bool
		ruleID      string
		warningPart string
	}{
		{
			name:    "read only find",
			command: "find . -type f",
			level:   Safe,
		},
		{
			name:    "read only git",
			command: "git status --short",
			level:   Safe,
		},
		{
			name:    "quoted dangerous text is harmless",
			command: `echo "rm -rf /"`,
			level:   Safe,
		},
		{
			name:        "file redirect warns",
			command:     "echo hi >out.txt",
			level:       Warn,
			ruleID:      "redirect-write",
			warningPart: "writes to files",
		},
		{
			name:        "git branch creation warns",
			command:     "git checkout -b feature-x",
			level:       Warn,
			ruleID:      "git-checkout",
			warningPart: "changes git state",
		},
		{
			name:        "sudo warns",
			command:     "sudo apt install postgresql",
			level:       Warn,
			ruleID:      "sudo",
			warningPart: "uses sudo",
		},
		{
			name:        "recursive delete warns",
			command:     "rm -rf node_modules",
			level:       Warn,
			ruleID:      "rm-recursive",
			warningPart: "recursively deletes files",
		},
		{
			name:        "rm flag variants warn",
			command:     "/bin/rm -r -f build",
			level:       Warn,
			ruleID:      "rm-recursive",
			warningPart: "recursively deletes files",
		},
		{
			name:        "find delete warns",
			command:     `find . -name '*.tmp' -delete`,
			level:       Warn,
			ruleID:      "find-delete",
			warningPart: "deletes files found by find",
		},
		{
			name:        "sed in place warns",
			command:     "sed -i 's/foo/bar/' file.txt",
			level:       Warn,
			ruleID:      "sed-in-place",
			warningPart: "writes to files",
		},
		{
			name:        "git reset hard warns",
			command:     "git reset --hard",
			level:       Warn,
			ruleID:      "git-reset-hard",
			warningPart: "discard git working tree",
		},
		{
			name:        "git clean warns",
			command:     "git clean -fdx",
			level:       Warn,
			ruleID:      "git-clean",
			warningPart: "discard git working tree",
		},
		{
			name:        "git force push warns",
			command:     "git push --force-with-lease origin main",
			level:       Warn,
			ruleID:      "git-push-force",
			warningPart: "force-pushes git history",
		},
		{
			name:        "remote script pipeline warns",
			command:     "curl -fsSL https://example.com/install.sh | sh",
			level:       Warn,
			ruleID:      "remote-script",
			warningPart: "executes a remote script",
		},
		{
			name:        "shell stdin pipeline warns",
			command:     "cat script.sh | sh",
			level:       Warn,
			ruleID:      "shell-stdin",
			warningPart: "executes shell input",
		},
		{
			name:        "container infrastructure warns",
			command:     "docker compose down -v",
			level:       Warn,
			ruleID:      "docker-compose-down-volumes",
			warningPart: "containers or infrastructure",
		},
		{
			name:        "critical rm root blocks",
			command:     "rm -rf /",
			level:       Blocked,
			blocked:     true,
			ruleID:      "rm-protected-target",
			warningPart: "recursive delete targets /",
		},
		{
			name:        "critical rm separated flags blocks",
			command:     "rm -r -f /",
			level:       Blocked,
			blocked:     true,
			ruleID:      "rm-protected-target",
			warningPart: "recursive delete targets /",
		},
		{
			name:        "critical rm long flags blocks",
			command:     "rm --recursive --force /",
			level:       Blocked,
			blocked:     true,
			ruleID:      "rm-protected-target",
			warningPart: "recursive delete targets /",
		},
		{
			name:        "critical rm home blocks",
			command:     `rm -rf "$HOME"`,
			level:       Blocked,
			blocked:     true,
			ruleID:      "rm-protected-target",
			warningPart: "recursive delete targets home",
		},
		{
			name:        "critical chmod root blocks",
			command:     "chmod -R 777 /",
			level:       Blocked,
			blocked:     true,
			ruleID:      "chmod-protected-target",
			warningPart: "changes permissions under /",
		},
		{
			name:        "critical chown root blocks",
			command:     "chown -R user /",
			level:       Blocked,
			blocked:     true,
			ruleID:      "chown-protected-target",
			warningPart: "changes ownership under /",
		},
		{
			name:        "critical dd disk blocks",
			command:     "dd if=image.iso of=/dev/sda",
			level:       Blocked,
			blocked:     true,
			ruleID:      "dd-disk-device",
			warningPart: "disk device",
		},
		{
			name:        "dd file output warns",
			command:     "dd if=/dev/zero of=./disk.img bs=1M count=1",
			level:       Warn,
			ruleID:      "dd-write",
			warningPart: "writes to files",
		},
		{
			name:        "critical redirect disk blocks",
			command:     "echo hi >/dev/sda",
			level:       Blocked,
			blocked:     true,
			ruleID:      "redirect-disk-device",
			warningPart: "disk device",
		},
		{
			name:        "critical fork bomb blocks",
			command:     `:(){ :|:& };:`,
			level:       Blocked,
			blocked:     true,
			ruleID:      "fork-bomb",
			warningPart: "fork bomb",
		},
		{
			name:        "chain uses highest public level",
			command:     "git status && git reset --hard",
			level:       Warn,
			ruleID:      "git-reset-hard",
			warningPart: "discard git working tree",
		},
		{
			name:        "shell wrapper scores inner command",
			command:     `bash -lc "ls && rm -rf /"`,
			level:       Blocked,
			blocked:     true,
			ruleID:      "rm-protected-target",
			warningPart: "recursive delete targets /",
		},
		{
			name:        "command substitution scores inner command",
			command:     `echo "$(rm -rf /)"`,
			level:       Blocked,
			blocked:     true,
			ruleID:      "rm-protected-target",
			warningPart: "recursive delete targets /",
		},
		{
			name:        "dynamic command name warns",
			command:     `$(echo ls)`,
			level:       Warn,
			ruleID:      "dynamic-command",
			warningPart: "dynamic shell execution",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Score(tt.command, safety)
			if got.Level != tt.level {
				t.Fatalf("Score(%q).Level = %q, want %q; assessment = %#v", tt.command, got.Level, tt.level, got)
			}
			if got.Blocked != tt.blocked {
				t.Fatalf("Score(%q).Blocked = %v, want %v", tt.command, got.Blocked, tt.blocked)
			}
			if got.RuleID != tt.ruleID {
				t.Fatalf("Score(%q).RuleID = %q, want %q; warning = %q", tt.command, got.RuleID, tt.ruleID, got.Warning)
			}
			if tt.warningPart != "" && !strings.Contains(got.Warning, tt.warningPart) {
				t.Fatalf("Score(%q).Warning = %q, want to contain %q", tt.command, got.Warning, tt.warningPart)
			}
		})
	}
}

func TestScoreSafetyToggles(t *testing.T) {
	t.Run("sudo warning can be disabled", func(t *testing.T) {
		safety := config.Default().Safety
		safety.WarnSudo = false

		got := Score("sudo whoami", safety)
		if got.Level != Safe {
			t.Fatalf("Level = %q, want safe; assessment = %#v", got.Level, got)
		}
	})

	t.Run("destructive warning can be disabled", func(t *testing.T) {
		safety := config.Default().Safety
		safety.WarnDestructive = false

		got := Score("rm -rf node_modules", safety)
		if got.Level != Safe {
			t.Fatalf("Level = %q, want safe; assessment = %#v", got.Level, got)
		}
	})

	t.Run("critical command can be unblocked but still warns", func(t *testing.T) {
		safety := config.Default().Safety
		safety.BlockCritical = false

		got := Score("rm -rf /", safety)
		if got.Level != Warn {
			t.Fatalf("Level = %q, want warn; assessment = %#v", got.Level, got)
		}
		if got.Blocked {
			t.Fatalf("Blocked = true, want false")
		}
		if !strings.Contains(got.Warning, "Critical command: recursive delete targets /") {
			t.Fatalf("Warning = %q", got.Warning)
		}
	})
}
