package prompt

import (
	"strings"
	"testing"

	shellcontext "github.com/longyijdos/hi-shell/internal/context"
)

func TestBuildSessionIncludesHistoryAndFeedback(t *testing.T) {
	built := BuildSession(Session{
		InitialPrompt: "list large files",
		Turns: []SessionTurn{
			{
				Command:  "find . -type f -size +100M",
				Risk:     "low",
				Feedback: "sort by size",
			},
			{
				Command:  "find . -type f -size +100M -printf '%s %p\\n' | sort -nr",
				Feedback: "show human readable sizes",
			},
		},
	}, shellcontext.Snapshot{
		WorkingDir: "/tmp/project",
		OS:         "linux",
		Arch:       "amd64",
	})

	wantFragments := []string{
		"Command revision session:",
		"Initial user request:\nlist large files",
		"1. Command: find . -type f -size +100M",
		"Risk: low",
		"User feedback after this command: sort by size",
		"2. Command: find . -type f -size +100M -printf '%s %p\\n' | sort -nr",
		"User feedback after this command: show human readable sizes",
		"pwd: /tmp/project",
		"os: linux/amd64",
		"Generate a revised command",
	}
	for _, fragment := range wantFragments {
		if !strings.Contains(built.User, fragment) {
			t.Fatalf("BuildSession().User missing %q:\n%s", fragment, built.User)
		}
	}
	if built.System != systemPrompt {
		t.Fatalf("System prompt changed")
	}
}

func TestBuildSessionWithoutTurnsUsesInitialPromptShape(t *testing.T) {
	built := BuildSession(Session{InitialPrompt: "list go files"}, shellcontext.Snapshot{})

	if !strings.Contains(built.User, "User request:\nlist go files") {
		t.Fatalf("User prompt = %q, want initial request shape", built.User)
	}
	if strings.Contains(built.User, "Command revision session") {
		t.Fatalf("User prompt = %q, should not use revision shape without history or feedback", built.User)
	}
}
