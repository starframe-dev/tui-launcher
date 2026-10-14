package packagejson

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tui-launcher/taskfile"
)

type packageJSON struct {
	Name    string            `json:"name"`
	Scripts map[string]string `json:"scripts"`
}

// isIdempotent auto-detects idempotency from common script name patterns
func isIdempotent(name string) bool {
	idempotentKeywords := []string{
		"build", "test", "lint", "format", "check", "verify",
		"generate", "compile", "bundle", "sync", "install",
		"dev", "watch", "serve", "start", "run",
	}
	for _, kw := range idempotentKeywords {
		if strings.Contains(strings.ToLower(name), kw) {
			return true
		}
	}
	return false
}

// detectPackageManager checks for lock files to determine which package manager to use.
// Priority: pnpm > yarn > bun > npm.
func detectPackageManager(projectPath string) string {
	if _, err := os.Stat(filepath.Join(projectPath, "pnpm-lock.yaml")); err == nil {
		return "pnpm"
	}
	if _, err := os.Stat(filepath.Join(projectPath, "yarn.lock")); err == nil {
		return "yarn"
	}
	if _, err := os.Stat(filepath.Join(projectPath, "bun.lockb")); err == nil {
		return "bun"
	}
	return "npm"
}

// scriptCommand returns the shell command to run a script for the given package manager.
func scriptCommand(pm, name string) string {
	switch pm {
	case "pnpm":
		return fmt.Sprintf("pnpm run %s", name)
	case "yarn":
		return fmt.Sprintf("yarn %s", name)
	case "bun":
		return fmt.Sprintf("bun run %s", name)
	default:
		return fmt.Sprintf("npm run %s", name)
	}
}

func Parse(projectPath string) ([]taskfile.Task, error) {
	pkgPath := filepath.Join(projectPath, "package.json")

	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read package.json: %w", err)
	}

	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}

	// Sort scripts for stable ordering
	var names []string
	for name := range pkg.Scripts {
		names = append(names, name)
	}
	sort.Strings(names)

	pm := detectPackageManager(projectPath)

	var tasks []taskfile.Task
	for _, name := range names {
		cmd := pkg.Scripts[name]
		tasks = append(tasks, taskfile.Task{
			Name:        name,
			Description: cmd,
			Command:     scriptCommand(pm, name),
			Idempotent:  isIdempotent(name),
			TabName:     name,
		})
	}

	return tasks, nil
}
