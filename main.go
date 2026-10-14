package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Task represents a task from Taskfile.yml or package.json
type Task struct {
	Name        string
	Description string
	Command     string
	Idempotent  bool
}

// SimpleTerminal wraps a process and output buffer (no PTY to avoid TTY conflicts)
type SimpleTerminal struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
	mu     sync.Mutex
	lines  []string
	width  int
	height int
	done   chan struct{}
}

func NewTerminal(command string, args []string, cwd string) (*SimpleTerminal, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	// Close stdin so bash doesn't wait for terminal input
	stdin, err := os.Open(os.DevNull)
	if err != nil {
		return nil, fmt.Errorf("open devnull: %w", err)
	}
	cmd.Stdin = stdin

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start: %w", err)
	}

	t := &SimpleTerminal{
		cmd:    cmd,
		stdout: stdout,
		lines:  []string{},
		done:   make(chan struct{}),
		width:  80,
		height: 24,
	}

	go t.readLoop()
	return t, nil
}

func (t *SimpleTerminal) readLoop() {
	reader := bufio.NewReader(t.stdout)
	var buf strings.Builder

	for {
		select {
		case <-t.done:
			return
		default:
		}

		ch, err := reader.ReadByte()
		if err != nil {
			return
		}

		if ch == '\n' {
			t.mu.Lock()
			t.lines = append(t.lines, buf.String())
			if len(t.lines) > 1000 {
				t.lines = t.lines[len(t.lines)-1000:]
			}
			t.mu.Unlock()
			buf.Reset()
		} else if ch != '\r' {
			buf.WriteByte(ch)
		}
	}
}

func (t *SimpleTerminal) Resize(width, height int) error {
	t.width = width
	t.height = height
	return nil
}

func (t *SimpleTerminal) Stop() {
	close(t.done)
	if t.stdout != nil {
		t.stdout.Close()
	}
	if t.cmd != nil && t.cmd.Process != nil {
		t.cmd.Process.Kill()
		t.cmd.Process.Wait()
	}
}

func (t *SimpleTerminal) View() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	start := 0
	if len(t.lines) > t.height {
		start = len(t.lines) - t.height
	}
	visible := t.lines[start:]

	var result []string
	for i, line := range visible {
		if i >= t.height {
			break
		}
		result = append(result, line)
	}

	for len(result) < t.height {
		result = append(result, "")
	}

	return strings.Join(result, "\n")
}

// MainApp is the unified launcher
type MainApp struct {
	projectPath     string
	projectName     string
	session         string
	tasks           []Task
	cursorPos       int
	width           int
	height          int
	initialized     bool
	terminal        *SimpleTerminal
	terminalRunning bool
	mu              sync.Mutex
}

func main() {
	var session string
	var projectDir string
	flag.StringVar(&session, "s", "default", "Session name")
	flag.StringVar(&session, "session", "default", "Session name")
	flag.StringVar(&projectDir, "d", "", "Project directory (default: cwd)")
	flag.StringVar(&projectDir, "dir", "", "Project directory (default: cwd)")
	flag.Parse()

	args := flag.Args()
	if len(args) > 0 {
		fmt.Fprintln(os.Stderr, "Unified app: TUI only")
		os.Exit(1)
	}

	projectPath := projectDir
	if projectPath == "" {
		projectPath, _ = os.Getwd()
	} else {
		abs, err := filepath.Abs(projectPath)
		if err == nil {
			projectPath = abs
		}
	}

	app := &MainApp{
		projectPath: projectPath,
		projectName: filepath.Base(projectPath),
		session:     session,
		initialized: false,
	}

	p := tea.NewProgram(app, tea.WithMouseCellMotion())
	_, _ = p.Run()
}

func (m *MainApp) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			tasks, _ := m.loadTasks()
			return tasksLoadedMsg{tasks: tasks}
		},
		tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}),
	)
}

type tasksLoadedMsg struct {
	tasks []Task
}

type tickMsg time.Time

func (m *MainApp) loadTasks() ([]Task, error) {
	tasks := []Task{
		{
			Name:        "build",
			Description: "Compile the project",
			Command:     "go build .",
			Idempotent:  true,
		},
		{
			Name:        "test",
			Description: "Run tests",
			Command:     "go test ./...",
			Idempotent:  true,
		},
		{
			Name:        "lint",
			Description: "Run linters",
			Command:     "go vet ./...",
			Idempotent:  true,
		},
		{
			Name:        "fmt",
			Description: "Format Go code",
			Command:     "gofmt -w .",
			Idempotent:  true,
		},
		{
			Name:        "run",
			Description: "Show current directory",
			Command:     "pwd && ls -la",
			Idempotent:  true,
		},
		{
			Name:        "dev",
			Description: "Run dev check",
			Command:     "go build . && echo \"Dev build OK\"",
			Idempotent:  true,
		},
		{
			Name:        "watch",
			Description: "Watch files (demo)",
			Command:     "echo \"Watching would rebuild here...\" && sleep 5",
			Idempotent:  true,
		},
		{
			Name:        "serve",
			Description: "Start dev server",
			Command:     "python3 -m http.server 8080",
			Idempotent:  true,
		},
		{
			Name:        "clean",
			Description: "Clean build artifacts",
			Command:     "rm -rf bin/",
			Idempotent:  true,
		},
		{
			Name:        "deps:tidy",
			Description: "Tidy Go modules",
			Command:     "go mod tidy",
			Idempotent:  true,
		},
		{
			Name:        "deps:update",
			Description: "Update dependencies",
			Command:     "go get -u ./...",
			Idempotent:  false,
		},
		{
			Name:        "docs:spec",
			Description: "Show architecture spec",
			Command:     "cat SPEC.md",
			Idempotent:  true,
		},
		{
			Name:        "release:build",
			Description: "Build release binaries",
			Command:     "go build -ldflags=\"-s -w\" -o bin/release .",
			Idempotent:  false,
		},
		{
			Name:        "migrate",
			Description: "Run database migrations",
			Command:     "echo 'migrate: not configured'",
			Idempotent:  false,
		},
		{
			Name:        "deploy",
			Description: "Deploy to production",
			Command:     "echo 'deploy: not configured'",
			Idempotent:  false,
		},
		{
			Name:        "setup",
			Description: "Setup project for first time",
			Command:     "go mod download",
			Idempotent:  false,
		},
	}
	return tasks, nil
}

func (m *MainApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tasksLoadedMsg:
		m.tasks = msg.tasks
		m.cursorPos = 0
		m.initialized = true
		return m, nil
	case tea.MouseMsg:
		// Handle mouse in task list
		if m.initialized && msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			clickedLine := msg.Y - 1
			if clickedLine >= 0 && clickedLine < len(m.tasks) && msg.X < 40 {
				m.cursorPos = clickedLine
				startCmd := m.runSelectedTask()
				if startCmd != nil {
					return m, startCmd
				}
				return m, nil
			}
		}
	case tea.KeyMsg:
		// Ctrl+C / Ctrl+Q - exit
		if msg.String() == "ctrl+c" || msg.String() == "ctrl+q" {
			if m.terminal != nil {
				m.terminal.Stop()
			}
			return m, tea.Quit
		}

		if !m.initialized {
			break
		}

		switch msg.Type {
		case tea.KeyUp:
			if m.cursorPos > 0 {
				m.cursorPos--
			}
		case tea.KeyDown:
			if m.cursorPos < len(m.tasks)-1 {
				m.cursorPos++
			}
		case tea.KeyEnter:
			startCmd := m.runSelectedTask()
			if startCmd != nil {
				return m, startCmd
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.terminal != nil {
			m.terminal.Resize(m.width-43, m.height-3)
		}
	case tickMsg:
		return m, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})
	}
	return m, nil
}

func (m *MainApp) runSelectedTask() tea.Cmd {
	if m.cursorPos < 0 || m.cursorPos >= len(m.tasks) {
		return nil
	}

	task := &m.tasks[m.cursorPos]
	if task.Command == "" {
		return nil
	}

	// Stop previous terminal if running
	if m.terminal != nil {
		m.terminal.Stop()
	}

	termWidth := m.width - 43
	termHeight := m.height - 3
	if termWidth < 20 {
		termWidth = 80
	}
	if termHeight < 10 {
		termHeight = 24
	}

	term, err := NewTerminal("bash", []string{"--norc", "--noediting", "-c", task.Command}, m.projectPath)
	if err != nil {
		return nil
	}
	term.Resize(termWidth, termHeight)
	m.terminal = term
	m.terminalRunning = true

	return func() tea.Msg {
		return nil
	}
}

func (m *MainApp) View() string {
	if m.width == 0 {
		m.width = 120
	}
	if m.height == 0 {
		m.height = 30
	}

	leftWidth := 40
	rightWidth := m.width - leftWidth - 3
	if rightWidth < 20 {
		rightWidth = 20
		leftWidth = m.width - rightWidth - 3
		if leftWidth < 20 {
			leftWidth = 20
		}
	}

	headerText := fmt.Sprintf("  🚀 %s ", m.projectName)
	if m.terminalRunning {
		headerText = fmt.Sprintf("  🚀 %s • ▶ Terminal running ", m.projectName)
	}
	headerPadding := strings.Repeat("═", (m.width-len(headerText))/2)
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00d4ff")).
		Render(headerPadding + headerText + headerPadding)

	leftPanel := m.renderTaskList(leftWidth, m.height-3)
	rightPanel := m.renderTerminal(rightWidth, m.height-3)

	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPanel,
		"│",
		rightPanel,
	)

	hint := "[INFO] ↑↓ navigate  Enter run  q quit  •  click task to run"
	if m.terminalRunning {
		hint = "[TERMINAL] Ctrl+C quit"
	}
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Background(lipgloss.Color("235")).
		Width(m.width).
		Render(hint)

	return strings.Join([]string{header, body, footer}, "\n")
}

func (m *MainApp) renderTaskList(width, height int) string {
	var lines []string

	if !m.initialized {
		lines = append(lines, lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Render("Loading tasks..."))
	} else {
		for i, task := range m.tasks {
			if i >= height {
				break
			}

			prefix := "  "
			if i == m.cursorPos {
				prefix = lipgloss.NewStyle().
					Foreground(lipgloss.Color("0")).
					Background(lipgloss.Color("#00d4ff")).
					Bold(true).
					Render("► ")
			}

			name := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ffffff")).
				Render(task.Name)
			desc := ""
			if task.Description != "" {
				descStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("#aaaaaa"))
				maxDesc := width - len(task.Name) - 6
				if maxDesc > 0 && len(task.Description) > maxDesc {
					desc = " " + descStyle.Render(task.Description[:maxDesc]+"…")
				} else {
					desc = " " + descStyle.Render(task.Description)
				}
			}
			lines = append(lines, fmt.Sprintf("%s%s%s", prefix, name, desc))
		}
	}

	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(content)
}

func (m *MainApp) renderTerminal(width, height int) string {
	if m.terminal == nil {
		placeholder := []string{
			"╔══════════════════════════════════════╗",
			"║                                      ║",
			"║     Terminal Output                  ║",
			"║                                      ║",
			"║   Select a task and press Enter       ║",
			"║   to run it here.                    ║",
			"║                                      ║",
			"║   Output will appear here.           ║",
			"║                                      ║",
			"╚══════════════════════════════════════╝",
		}
		content := strings.Join(placeholder, "\n")
		return lipgloss.NewStyle().
			Width(width).
			Height(height).
			Render(content)
	}

	m.terminal.Resize(width, height)
	return m.terminal.View()
}
