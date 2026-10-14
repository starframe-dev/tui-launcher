package executor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Runner sends commands to viewer via CLI
type Runner struct {
	session string
}

// New creates a runner that sends commands to the given viewer session
func New(session string) *Runner {
	return &Runner{session: session}
}

// viewerPath returns the path to the viewer binary (tui-launcher-viewer).
// It searches in this order:
// 1. PATH via exec.LookPath("tui-launcher-viewer")
// 2. Directory of the current executable (for symlinked installs)
// 3. Fallback to "tui-launcher-viewer"
func viewerPath() string {
	// 1. Search PATH for "tui-launcher-viewer"
	if vp, err := exec.LookPath("tui-launcher-viewer"); err == nil {
		return vp
	}

	// 2. Try to find viewer in the same directory as tui-launcher
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		viewer := filepath.Join(dir, "tui-launcher-viewer")
		if info, err := os.Stat(viewer); err == nil && !info.IsDir() {
			return viewer
		}
	}

	// 3. Fallback to PATH
	return "tui-launcher-viewer"
}

// Run sends a command to viewer.
// If idempotent, restarts existing tab with the same name.
// If not idempotent, creates a new tab after the existing one.
// After the command finishes, an interactive bash is started so the tab stays open.
func (r *Runner) Run(name, dir, command string, idempotent bool) error {
	// Add "exec bash -i" so the tab stays open with interactive shell after command completes
	fullCmd := fmt.Sprintf("cd %s && %s; exec bash -i", shellEscape(dir), command)

	vp := viewerPath()
	var cmd *exec.Cmd
	if idempotent {
		cmd = exec.Command(vp, "-s", r.session, "restart-tab", name, fullCmd)
	} else {
		cmd = exec.Command(vp, "-s", r.session, "new-tab", name, fullCmd)
	}
	return cmd.Run()
}

// RunRaw sends a raw command to viewer without cd prefix or bash -i suffix.
// Used for CLI mode.
func (r *Runner) RunRaw(command string) error {
	vp := viewerPath()
	cmd := exec.Command(vp, "-s", r.session, "send-cmd", command)
	return cmd.Run()
}

// Stop is a no-op for viewer-based runner
func (r *Runner) Stop() {}

func shellEscape(s string) string {
	if strings.Contains(s, " ") {
		return "\"" + s + "\""
	}
	return s
}
