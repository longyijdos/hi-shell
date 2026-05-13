package prompt

import (
	"strings"
	"testing"

	shellcontext "github.com/longyijdos/hi-shell/internal/context"
)

func TestBuildReviseSessionIncludesHistoryAndFeedback(t *testing.T) {
	built := BuildReviseSession(ReviseSession{
		InitialPrompt: "list large files",
		Turns: []ReviseTurn{
			{
				Command:  "find . -type f -size +100M",
				Risk:     "safe",
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
		"Risk: safe",
		"User feedback after this command: sort by size",
		"2. Command: find . -type f -size +100M -printf '%s %p\\n' | sort -nr",
		"User feedback after this command: show human readable sizes",
		"pwd: /tmp/project",
		"os: linux/amd64",
		"Generate a revised command",
	}
	for _, fragment := range wantFragments {
		if !strings.Contains(built.User, fragment) {
			t.Fatalf("BuildReviseSession().User missing %q:\n%s", fragment, built.User)
		}
	}
	if built.System != commandSystemPrompt {
		t.Fatalf("System prompt changed")
	}
}

func TestBuildReviseSessionWithoutTurnsUsesGeneratePromptShape(t *testing.T) {
	built := BuildReviseSession(ReviseSession{InitialPrompt: "list go files"}, shellcontext.Snapshot{})

	if !strings.Contains(built.User, "User request:\nlist go files") {
		t.Fatalf("User prompt = %q, want initial request shape", built.User)
	}
	if strings.Contains(built.User, "Command revision session") {
		t.Fatalf("User prompt = %q, should not use revision shape without history or feedback", built.User)
	}
}

func TestBuildAskSessionIncludesCommandAndQuestion(t *testing.T) {
	built := BuildAskSession(AskSession{
		InitialPrompt: "list large files",
		Turns: []AskTurn{
			{
				Command:  "find . -type f -size +100M",
				Risk:     "safe",
				Question: "will this modify files?",
				Answer:   "No. It only lists matching files.",
			},
			{
				Command:  "find . -type f -size +100M",
				Question: "what does -size mean?",
			},
		},
	}, shellcontext.Snapshot{
		WorkingDir: "/tmp/project",
		OS:         "linux",
		Arch:       "amd64",
	})

	wantFragments := []string{
		"Command question session:",
		"Initial user request:\nlist large files",
		"1. Command: find . -type f -size +100M",
		"Risk: safe",
		"User question: will this modify files?",
		"Previous answer: No. It only lists matching files.",
		"2. Command: find . -type f -size +100M",
		"User question: what does -size mean?",
		"pwd: /tmp/project",
		"Answer the user's latest question",
	}
	for _, fragment := range wantFragments {
		if !strings.Contains(built.User, fragment) {
			t.Fatalf("BuildAskSession().User missing %q:\n%s", fragment, built.User)
		}
	}
	if built.System != askSystemPrompt {
		t.Fatalf("System prompt changed")
	}
}
