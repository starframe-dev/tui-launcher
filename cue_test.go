package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/starframe-dev/cue-tty/pkg/cue"
)

func TestTUILauncher(t *testing.T) {
	projectDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	binaryPath := filepath.Join(t.TempDir(), "tui-launcher")
	build := exec.Command("go", "build", "-o", binaryPath, ".")
	build.Dir = projectDir
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build tui-launcher: %v\n%s", err, output)
	}

	app, err := cue.Launch(binaryPath,
		cue.WithArgs("-s", "test-session", "-d", projectDir),
		cue.WithSize(120, 30),
		cue.WithEnv("TERM=xterm-256color"),
		cue.WithTimeout(30*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	page := app.Page()
	page.EnableMouse()
	page.WaitStable(500 * time.Millisecond)

	if err := cue.Expect(page).ToContain("tui-launcher"); err != nil {
		t.Fatal(err)
	}
	if err := cue.Expect(page).ToContain("build"); err != nil {
		t.Fatal(err)
	}

	for range 4 {
		page.Press("Down")
	}
	page.Press("Enter")
	page.WaitStable(500 * time.Millisecond)

	if err := cue.Expect(page).ToContain("CONTEXT.md"); err != nil {
		t.Fatal(err)
	}
}
