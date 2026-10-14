package packagejson

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	tmpDir := t.TempDir()
	pkgPath := filepath.Join(tmpDir, "package.json")
	content := `{
		"name": "test-project",
		"scripts": {
			"build": "tsc && vite build",
			"test": "vitest run",
			"lint": "eslint .",
			"migrate": "prisma migrate deploy",
			"dev": "vite dev"
		}
	}`
	if err := os.WriteFile(pkgPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	tasks, err := Parse(tmpDir)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(tasks) != 5 {
		t.Fatalf("expected 5 tasks, got %d", len(tasks))
	}

	// Check stable ordering (alphabetical by name)
	expectedNames := []string{"build", "dev", "lint", "migrate", "test"}
	for i, name := range expectedNames {
		if tasks[i].Name != name {
			t.Errorf("task[%d].Name = %q, want %q", i, tasks[i].Name, name)
		}
	}

	// Check idempotency auto-detection
	for _, task := range tasks {
		switch task.Name {
		case "build", "test", "lint", "dev":
			if !task.Idempotent {
				t.Errorf("task %q should be idempotent", task.Name)
			}
		case "migrate":
			if task.Idempotent {
				t.Errorf("task %q should NOT be idempotent", task.Name)
			}
		}
	}

	// Check command format (default = npm, no lock file)
	for _, task := range tasks {
		wantCmd := "npm run " + task.Name
		if task.Command != wantCmd {
			t.Errorf("task %q Command = %q, want %q", task.Name, task.Command, wantCmd)
		}
	}

	// Check description = raw script content
	if tasks[0].Description != "tsc && vite build" {
		t.Errorf("build.Description = %q, want %q", tasks[0].Description, "tsc && vite build")
	}
}

func TestParse_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := Parse(tmpDir)
	if err == nil {
		t.Fatal("expected error for missing package.json")
	}
}

func TestParse_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	pkgPath := filepath.Join(tmpDir, "package.json")
	if err := os.WriteFile(pkgPath, []byte("not json"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	_, err := Parse(tmpDir)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParse_Pnpm(t *testing.T) {
	tmpDir := t.TempDir()
	pkgPath := filepath.Join(tmpDir, "package.json")
	lockPath := filepath.Join(tmpDir, "pnpm-lock.yaml")
	content := `{"scripts": {"build": "tsc"}}`
	if err := os.WriteFile(pkgPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}
	if err := os.WriteFile(lockPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write pnpm-lock.yaml: %v", err)
	}

	tasks, err := Parse(tmpDir)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if tasks[0].Command != "pnpm run build" {
		t.Errorf("Command = %q, want %q", tasks[0].Command, "pnpm run build")
	}
}

func TestParse_Yarn(t *testing.T) {
	tmpDir := t.TempDir()
	pkgPath := filepath.Join(tmpDir, "package.json")
	lockPath := filepath.Join(tmpDir, "yarn.lock")
	content := `{"scripts": {"build": "tsc"}}`
	if err := os.WriteFile(pkgPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}
	if err := os.WriteFile(lockPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write yarn.lock: %v", err)
	}

	tasks, err := Parse(tmpDir)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if tasks[0].Command != "yarn build" {
		t.Errorf("Command = %q, want %q", tasks[0].Command, "yarn build")
	}
}

func TestParse_Bun(t *testing.T) {
	tmpDir := t.TempDir()
	pkgPath := filepath.Join(tmpDir, "package.json")
	lockPath := filepath.Join(tmpDir, "bun.lockb")
	content := `{"scripts": {"build": "tsc"}}`
	if err := os.WriteFile(pkgPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}
	if err := os.WriteFile(lockPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write bun.lockb: %v", err)
	}

	tasks, err := Parse(tmpDir)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if tasks[0].Command != "bun run build" {
		t.Errorf("Command = %q, want %q", tasks[0].Command, "bun run build")
	}
}
