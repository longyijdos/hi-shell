package shellcontext

import (
	"reflect"
	"strings"
	"testing"

	"github.com/longyijdos/hi-shell/internal/config"
)

func TestCollectHistoryRequiresSetting(t *testing.T) {
	t.Setenv(historyEnv, "git status\nnpm test")

	disabled := Collect(config.ContextConfig{})
	if len(disabled.RecentHistory) != 0 {
		t.Fatalf("Collect() with history disabled got %#v, want none", disabled.RecentHistory)
	}

	enabled := Collect(config.ContextConfig{History: true})
	want := []string{"git status", "npm test"}
	if !reflect.DeepEqual(enabled.RecentHistory, want) {
		t.Fatalf("Collect() history = %#v, want %#v", enabled.RecentHistory, want)
	}
}

func TestSanitizeHistoryFiltersNoiseAndSecrets(t *testing.T) {
	raw := strings.Join([]string{
		" hi list go files ",
		"git status",
		"export OPENAI_API_KEY=sk-test",
		"curl -H 'Authorization: Bearer secret' https://example.com",
		"git status",
		"npm test",
		"",
	}, "\n")

	got := sanitizeHistory(raw)
	want := []string{"git status", "npm test"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sanitizeHistory() = %#v, want %#v", got, want)
	}
}

func TestSnapshotStringIncludesRecentHistory(t *testing.T) {
	got := Snapshot{RecentHistory: []string{"git status", "npm test"}}.String()
	for _, fragment := range []string{
		"recent history:",
		"- git status",
		"- npm test",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("Snapshot.String() = %q, missing %q", got, fragment)
		}
	}
}
