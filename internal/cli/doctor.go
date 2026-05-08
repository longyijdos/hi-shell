package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/longyijdos/hi-shell/internal/config"
)

func commandDoctor(args []string, stdout, stderr io.Writer, version string) int {
	if wantsHelp(args) {
		return printHelp(stdout, doctorUsage)
	}
	if rejectUnexpectedArgs(stderr, "doctor", args) {
		return 2
	}

	ok := true
	check := func(name string, pass bool, detail string) {
		status := "ok"
		if !pass {
			status = "fail"
			ok = false
		}
		if detail == "" {
			fmt.Fprintf(stdout, "%-18s %s\n", name, status)
		} else {
			fmt.Fprintf(stdout, "%-18s %s - %s\n", name, status, detail)
		}
	}

	fmt.Fprintf(stdout, "hi-shell %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)

	cfg, cfgPath, err := config.Load()
	_, cfgStatErr := os.Stat(cfgPath)
	check("config", err == nil && cfgStatErr == nil, cfgPath)
	if err != nil {
		return 1
	}

	check("provider", cfg.Provider == "openai" || cfg.Provider == "deepseek" || cfg.Provider == "openai-compatible", cfg.Provider)
	switch strings.ToLower(cfg.Provider) {
	case "openai", "openai-compatible":
		apiKeyReady := cfg.OpenAI.APIKeyEnv == "" || os.Getenv(cfg.OpenAI.APIKeyEnv) != "" || !strings.Contains(cfg.OpenAI.BaseURL, "api.openai.com")
		detail := cfg.OpenAI.BaseURL
		if cfg.OpenAI.APIKeyEnv != "" {
			detail += " via $" + cfg.OpenAI.APIKeyEnv
		}
		check("openai", apiKeyReady, detail)
	case "deepseek":
		apiKeyReady := cfg.DeepSeek.APIKeyEnv == "" || os.Getenv(cfg.DeepSeek.APIKeyEnv) != "" || !strings.Contains(cfg.DeepSeek.BaseURL, "deepseek.com")
		detail := cfg.DeepSeek.BaseURL + ", thinking=" + cfg.DeepSeek.Thinking
		if cfg.DeepSeek.APIKeyEnv != "" {
			detail += " via $" + cfg.DeepSeek.APIKeyEnv
		}
		check("deepseek", apiKeyReady, detail)
	}

	hiHome, err := config.HomeDir()
	if err == nil {
		_, statErr := os.Stat(filepath.Join(hiHome, "shell", "hi.zsh"))
		check("zsh plugin", statErr == nil, filepath.Join(hiHome, "shell", "hi.zsh"))
	}

	zshrc, err := zshrcPath()
	if err == nil {
		hasBlock := fileContains(zshrc, beginMarker) && fileContains(zshrc, endMarker)
		check("zshrc block", hasBlock, zshrc)
	}

	hiShellPath, err := exec.LookPath("hi-shell")
	check("binary in PATH", err == nil, hiShellPath)

	if !ok {
		return 1
	}
	return 0
}
