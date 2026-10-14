package taskfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Task struct {
	Name        string
	Description string
	Command     string
	Idempotent  bool   // if true, command can be safely re-run; viewer will reuse existing tab
	TabName     string // optional: custom tab name in viewer (defaults to task name)
}

type Taskfile struct {
	Version string             `yaml:"version"`
	Tasks   map[string]TaskDef `yaml:"tasks"`
}

type TaskDef struct {
	Cmd        interface{} `yaml:"cmds"` // can be string or []string
	Desc       string      `yaml:"desc"`
	Directory  string      `yaml:"dir"`
	Idempotent *bool       `yaml:"idempotent"` // nil = auto-detect from cmd name
	TabName    string      `yaml:"tab"`        // custom tab name in viewer
}

// isIdempotent auto-detects idempotency from common command patterns
func isIdempotent(name, cmd string) bool {
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

func Parse(projectPath string) ([]Task, error) {
	taskfilePath := filepath.Join(projectPath, "Taskfile.yml")

	data, err := os.ReadFile(taskfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Taskfile: %w", err)
	}

	var tf Taskfile
	if err := yaml.Unmarshal(data, &tf); err != nil {
		return nil, fmt.Errorf("failed to parse Taskfile: %w", err)
	}

	var tasks []Task
	for name, def := range tf.Tasks {
		cmdStr := ""
		switch cmd := def.Cmd.(type) {
		case string:
			cmdStr = cmd
		case []interface{}:
			for i, c := range cmd {
				if s, ok := c.(string); ok {
					if i > 0 {
						cmdStr += " && "
					}
					cmdStr += s
				}
			}
		}

		idempotent := isIdempotent(name, cmdStr)
		if def.Idempotent != nil {
			idempotent = *def.Idempotent
		}

		tabName := def.TabName
		if tabName == "" {
			tabName = name
		}

		tasks = append(tasks, Task{
			Name:        name,
			Description: def.Desc,
			Command:     cmdStr,
			Idempotent:  idempotent,
			TabName:     tabName,
		})
	}

	return tasks, nil
}
