package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	hishell "github.com/longyijdos/hi-shell"
	"github.com/longyijdos/hi-shell/internal/config"
)

const (
	beginMarker = "# >>> hi-shell initialize >>>"
	endMarker   = "# <<< hi-shell initialize <<<"
)

func commandInstall(args []string, stdout, stderr io.Writer) int {
	if wantsHelp(args) {
		return printHelp(stdout, installUsage)
	}
	if len(args) != 1 || args[0] != "zsh" {
		fmt.Fprintln(stderr, installUsage)
		return 2
	}

	_, cfgPath, err := config.Ensure()
	if err != nil {
		fmt.Fprintf(stderr, "install: %v\n", err)
		return 1
	}

	hiHome, err := config.HomeDir()
	if err != nil {
		fmt.Fprintf(stderr, "install: %v\n", err)
		return 1
	}

	shellDir := filepath.Join(hiHome, "shell")
	if err := os.MkdirAll(shellDir, 0o755); err != nil {
		fmt.Fprintf(stderr, "install: %v\n", err)
		return 1
	}

	pluginPath := filepath.Join(shellDir, "hi.zsh")
	if err := os.WriteFile(pluginPath, []byte(hishell.ShellPlugin), 0o644); err != nil {
		fmt.Fprintf(stderr, "install: %v\n", err)
		return 1
	}

	zshrcPath, err := zshrcPath()
	if err != nil {
		fmt.Fprintf(stderr, "install: %v\n", err)
		return 1
	}
	if err := ensureManagedBlock(zshrcPath, managedBlock(hiHome)); err != nil {
		fmt.Fprintf(stderr, "install: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Installed zsh plugin: %s\n", pluginPath)
	fmt.Fprintf(stdout, "Config: %s\n", cfgPath)
	fmt.Fprintln(stdout, "Restart zsh or run: exec zsh")
	return 0
}

func commandUninstall(args []string, stdout, stderr io.Writer) int {
	if wantsHelp(args) {
		return printHelp(stdout, uninstallUsage)
	}

	fs := newFlagSet("uninstall", uninstallUsage, stderr)

	var purge bool
	fs.BoolVar(&purge, "purge", false, "remove ~/.hi-shell, including config")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if rejectUnexpectedArgs(stderr, "uninstall", fs.Args()) {
		return 2
	}

	zshrc, err := zshrcPath()
	if err == nil {
		if err := removeManagedBlock(zshrc); err != nil {
			fmt.Fprintf(stderr, "uninstall: %v\n", err)
			return 1
		}
	}

	home, err := os.UserHomeDir()
	if err == nil {
		_ = os.Remove(filepath.Join(home, ".local", "bin", "hi-shell"))
	}

	hiHome, err := config.HomeDir()
	if err == nil {
		if purge {
			_ = os.RemoveAll(hiHome)
		} else {
			_ = os.Remove(filepath.Join(hiHome, "shell", "hi.zsh"))
		}
	}

	if purge {
		fmt.Fprintln(stdout, "Uninstalled hi-shell and removed ~/.hi-shell.")
	} else {
		fmt.Fprintln(stdout, "Uninstalled hi-shell. Config was preserved.")
	}
	return 0
}

func managedBlock(hiHome string) string {
	pluginPath := filepath.Join(hiHome, "shell", "hi.zsh")
	return beginMarker + "\n" +
		"source " + shellQuote(pluginPath) + "\n" +
		endMarker + "\n"
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func zshrcPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".zshrc"), nil
}

func ensureManagedBlock(path, block string) error {
	existing, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	text, _ := removeBlock(string(existing))
	text = strings.TrimRight(text, "\n")
	if text != "" {
		text += "\n\n"
	}
	text += block

	return os.WriteFile(path, []byte(text), 0o644)
}

func removeManagedBlock(path string) error {
	existing, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	text, changed := removeBlock(string(existing))
	if !changed {
		return nil
	}
	return os.WriteFile(path, []byte(text), 0o644)
}

func removeBlock(text string) (string, bool) {
	start := strings.Index(text, beginMarker)
	if start < 0 {
		return text, false
	}
	endRel := strings.Index(text[start:], endMarker)
	if endRel < 0 {
		return text, false
	}
	end := start + endRel + len(endMarker)
	if end < len(text) && text[end] == '\n' {
		end++
	}

	before := strings.TrimRight(text[:start], "\n")
	after := strings.TrimLeft(text[end:], "\n")
	switch {
	case before == "":
		return after, true
	case after == "":
		return before + "\n", true
	default:
		return before + "\n\n" + after, true
	}
}

func fileContains(path, needle string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), needle)
}
