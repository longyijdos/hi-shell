package shellcontext

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/longyijdos/hi-shell/internal/config"
)

type Snapshot struct {
	WorkingDir     string
	OS             string
	Arch           string
	Shell          string
	ZshVersion     string
	GitBranch      string
	GitDirty       bool
	GitAvailable   bool
	ProjectTypes   []string
	PackageScripts map[string]string
}

func Collect(settings config.ContextConfig) Snapshot {
	var snap Snapshot

	wd, _ := os.Getwd()
	if settings.PWD {
		snap.WorkingDir = wd
	}
	if settings.OS {
		snap.OS = runtime.GOOS
		snap.Arch = runtime.GOARCH
	}
	if settings.Shell {
		snap.Shell = os.Getenv("SHELL")
		snap.ZshVersion = os.Getenv("ZSH_VERSION")
	}
	if settings.Git && wd != "" {
		branch, dirty, ok := gitStatus(wd)
		snap.GitBranch = branch
		snap.GitDirty = dirty
		snap.GitAvailable = ok
	}
	if settings.ProjectFiles && wd != "" {
		snap.ProjectTypes = detectProjectTypes(wd)
	}
	if settings.PackageScripts && wd != "" {
		snap.PackageScripts = readPackageScripts(wd)
	}

	return snap
}

func (s Snapshot) String() string {
	var lines []string

	if s.WorkingDir != "" {
		lines = append(lines, "pwd: "+s.WorkingDir)
	}
	if s.OS != "" {
		lines = append(lines, "os: "+s.OS+"/"+s.Arch)
	}
	if s.Shell != "" {
		shell := "shell: " + s.Shell
		if s.ZshVersion != "" {
			shell += " (zsh " + s.ZshVersion + ")"
		}
		lines = append(lines, shell)
	}
	if s.GitAvailable {
		state := "clean"
		if s.GitDirty {
			state = "dirty"
		}
		lines = append(lines, "git: branch "+s.GitBranch+", "+state)
	}
	if len(s.ProjectTypes) > 0 {
		lines = append(lines, "project files: "+strings.Join(s.ProjectTypes, ", "))
	}
	if len(s.PackageScripts) > 0 {
		keys := make([]string, 0, len(s.PackageScripts))
		for key := range s.PackageScripts {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		var scripts []string
		for _, key := range keys {
			scripts = append(scripts, key+"="+s.PackageScripts[key])
		}
		lines = append(lines, "package scripts: "+strings.Join(scripts, "; "))
	}

	return strings.Join(lines, "\n")
}

func gitStatus(dir string) (string, bool, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 700*time.Millisecond)
	defer cancel()

	branchOut, err := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", false, false
	}

	statusOut, err := exec.CommandContext(ctx, "git", "-C", dir, "status", "--porcelain").Output()
	if err != nil {
		return strings.TrimSpace(string(branchOut)), false, true
	}

	return strings.TrimSpace(string(branchOut)), len(strings.TrimSpace(string(statusOut))) > 0, true
}

func detectProjectTypes(dir string) []string {
	candidates := map[string]string{
		"package.json":   "node",
		"go.mod":         "go",
		"pyproject.toml": "python",
		"Cargo.toml":     "rust",
		"composer.json":  "php",
		"Gemfile":        "ruby",
		"pom.xml":        "java-maven",
		"build.gradle":   "java-gradle",
	}

	var found []string
	for file, projectType := range candidates {
		if _, err := os.Stat(filepath.Join(dir, file)); err == nil {
			found = append(found, projectType+"("+file+")")
		}
	}

	sort.Strings(found)
	return found
}

func readPackageScripts(dir string) map[string]string {
	path := filepath.Join(dir, "package.json")
	info, err := os.Stat(path)
	if err != nil || info.Size() > 1_000_000 {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}
	if len(pkg.Scripts) == 0 {
		return nil
	}
	return pkg.Scripts
}
