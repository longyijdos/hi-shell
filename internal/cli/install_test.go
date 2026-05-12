package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestManagedBlockUsesInstalledPluginPath(t *testing.T) {
	hiHome := filepath.Join(t.TempDir(), "custom hi's home")
	wantPluginPath := filepath.Join(hiHome, "shell", "hi.zsh")

	got := managedBlock(hiHome)

	if !strings.Contains(got, beginMarker+"\n") || !strings.Contains(got, "\n"+endMarker+"\n") {
		t.Fatalf("managedBlock() = %q, want managed markers", got)
	}
	if !strings.Contains(got, "source "+shellQuote(wantPluginPath)+"\n") {
		t.Fatalf("managedBlock() = %q, want source path %q", got, wantPluginPath)
	}
	if strings.Contains(got, "$HOME/.hi-shell") {
		t.Fatalf("managedBlock() = %q, should not hard-code default home", got)
	}
}
