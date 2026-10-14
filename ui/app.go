package ui

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"tui-launcher/executor"
	"tui-launcher/packagejson"
	"tui-launcher/taskfile"
)

type viewMode int

const (
	modeVertical viewMode = iota
	modeHorizontal
)

type TaskNode struct {
	Task     *taskfile.Task
	Children map[string]*TaskNode
	IsGroup  bool
}

type model struct {
	projectPath string
	projectName string
	session     string
	tasks       map[string]*TaskNode
	flatList    []interface{} // Task or group name
	cursorPos   int
	runner      *executor.Runner
	width       int
	height      int
	scanning    bool
	hovered     int
	viewMode    viewMode
	initErr     string
}

func New(session string) *model {
	dir, err := os.Getwd()
	if err != nil {
		dir = os.Getenv("PWD")
	}
	if dir == "" {
		dir = "."
	}

	return &model{
		projectPath: dir,
		projectName: filepath.Base(dir),
		session:     session,
		cursorPos:   0,
		runner:      executor.New(session),
		hovered:     -1,
		viewMode:    modeVertical,
	}
}

func (m *model) Init() tea.Cmd {
	return func() tea.Msg {
		tasks, source, err := m.loadTasks()
		if err != nil {
			m.initErr = err.Error()
			return nil
		}
		_ = source
		m.buildTaskTree(tasks)
		return nil
	}
}

// loadTasks tries Taskfile.yml first, then package.json.
// If both exist, tasks are merged with Taskfile taking priority on name conflicts.
func (m *model) loadTasks() ([]taskfile.Task, string, error) {
	var allTasks []taskfile.Task
	var sources []string

	// Try Taskfile.yml
	if tfTasks, err := taskfile.Parse(m.projectPath); err == nil {
		allTasks = append(allTasks, tfTasks...)
		sources = append(sources, "Taskfile")
	}

	// Try package.json
	if pjTasks, err := packagejson.Parse(m.projectPath); err == nil {
		// Merge: Taskfile names take priority
		existing := make(map[string]struct{})
		for _, t := range allTasks {
			existing[t.Name] = struct{}{}
		}
		for _, t := range pjTasks {
			if _, ok := existing[t.Name]; !ok {
				allTasks = append(allTasks, t)
			}
		}
		sources = append(sources, "package.json")
	}

	if len(allTasks) == 0 {
		return nil, "", fmt.Errorf("no Taskfile.yml or package.json found")
	}

	sourceStr := sources[0]
	if len(sources) > 1 {
		sourceStr = sources[0] + " + " + sources[1]
	}
	return allTasks, sourceStr, nil
}

func (m *model) buildTaskTree(tasks []taskfile.Task) {
	m.tasks = make(map[string]*TaskNode)
	m.flatList = []interface{}{}

	// Group tasks by prefix
	groups := make(map[string][]*taskfile.Task)

	for i := range tasks {
		t := &tasks[i]
		parts := strings.Split(t.Name, ":")
		if len(parts) > 1 {
			// Has group
			group := parts[0]
			if _, ok := groups[group]; !ok {
				groups[group] = []*taskfile.Task{}
			}
			groups[group] = append(groups[group], t)
		} else {
			// No group - add directly
			m.flatList = append(m.flatList, t)
		}
	}

	// Sort groups alphabetically
	var groupNames []string
	for name := range groups {
		groupNames = append(groupNames, name)
	}
	sort.Strings(groupNames)

	// Add groups and their tasks
	for _, groupName := range groupNames {
		groupTasks := groups[groupName]
		m.flatList = append(m.flatList, groupName) // Add group header
		for _, t := range groupTasks {
			m.flatList = append(m.flatList, t)
		}
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		keyStr := msg.String()

		if keyStr == "ctrl+c" {
			m.runner.Stop()
			return m, tea.Quit
		}

		if keyStr == "ctrl+r" {
			m.scanning = true
			return m, func() tea.Msg {
				tasks, _, err := m.loadTasks()
				if err != nil {
					m.initErr = err.Error()
				} else {
					m.buildTaskTree(tasks)
				}
				m.scanning = false
				return nil
			}
		}

		switch keyStr {
		case "q":
			return m, tea.Quit

		case "tab":
			if m.viewMode == modeVertical {
				m.viewMode = modeHorizontal
			} else {
				m.viewMode = modeVertical
			}
			m.cursorPos = 0

		case "up":
			if m.cursorPos > 0 {
				m.cursorPos--
			}

		case "down":
			if m.cursorPos < len(m.flatList)-1 {
				m.cursorPos++
			}

		case "enter":
			m.runSelectedTask()
		}

	case tea.MouseMsg:
		m.handleMouse(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m *model) handleMouse(msg tea.MouseMsg) {
	if msg.Type == tea.MouseLeft {
		itemIndex := msg.Y - 3
		if itemIndex >= 0 && itemIndex < len(m.flatList) {
			m.cursorPos = itemIndex
			m.runSelectedTask()
		}
	}

	if msg.Type == tea.MouseMotion || msg.Type == tea.MouseLeft {
		m.hovered = -1
		itemIndex := msg.Y - 3
		if itemIndex >= 0 && itemIndex < len(m.flatList) {
			m.hovered = itemIndex
		}
	}
}

func (m *model) runSelectedTask() {
	if m.cursorPos < 0 || m.cursorPos >= len(m.flatList) {
		return
	}

	item := m.flatList[m.cursorPos]
	task, ok := item.(*taskfile.Task)
	if !ok {
		return
	}

	if task.Command == "" {
		return
	}

	m.runner = executor.New(m.session)
	go func() {
		if err := m.runner.Run(task.Name, m.projectPath, task.Command, task.Idempotent); err != nil {
			// For debugging: write error to a temp file
			// In production, this could be shown in the UI
			_ = err
		}
	}()
}

func (m *model) View() string {
	width := m.width
	if width < 60 {
		width = 100
	}

	if m.viewMode == modeVertical {
		return m.viewVertical(width)
	}
	return m.viewHorizontal(width)
}

func (m *model) viewVertical(width int) string {
	var lines []string

	lines = append(lines, lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Render(m.projectName))
	lines = append(lines, dimTextStyle.Render(m.projectPath))
	lines = append(lines, "")

	if m.initErr != "" {
		lines = append(lines, ErrorStyle.Render("[ERROR] "+m.initErr))
		lines = append(lines, "")
	}

	if len(m.flatList) == 0 && m.initErr == "" {
		lines = append(lines, dimTextStyle.Render("No tasks"))
		lines = append(lines, dimTextStyle.Render("Add Taskfile.yml or package.json"))
	} else {
		for i, item := range m.flatList {
			isSelected := i == m.cursorPos
			isHovered := i == m.hovered && !isSelected

			switch v := item.(type) {
			case string:
				if isSelected {
					lines = append(lines, lipgloss.NewStyle().
						Foreground(lipgloss.Color("0")).
						Background(primaryColor).
						Render(" "+v))
				} else if isHovered {
					lines = append(lines, lipgloss.NewStyle().
						Background(dimColor).
						Foreground(lipgloss.Color("0")).
						Render(" "+v))
				} else {
					lines = append(lines, GroupStyle.Render(" "+v))
				}
			case *taskfile.Task:
				prefix := "  "
				if isSelected {
					prefix = lipgloss.NewStyle().
						Background(primaryColor).
						Foreground(lipgloss.Color("0")).
						Render(" >")
				} else if isHovered {
					prefix = lipgloss.NewStyle().
						Background(dimColor).
						Foreground(lipgloss.Color("0")).
						Render(">")
				}
				name := m.getTaskName(v.Name, isSelected, isHovered)
				desc := ""
				if v.Description != "" {
					desc = " " + TaskDescStyle.Render(v.Description)
				}
				lines = append(lines, fmt.Sprintf("%s %s%s", prefix, name, desc))
			}
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return m.wrapWithHelp(content, width)
}

func (m *model) viewHorizontal(width int) string {
	// For horizontal mode, flatten to just task names
	var lines []string

	lines = append(lines, lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Render(m.projectName))
	lines = append(lines, dimTextStyle.Render(m.projectPath))
	lines = append(lines, "")

	var buttons []string
	for i, item := range m.flatList {
		if task, ok := item.(*taskfile.Task); ok {
			isSelected := i == m.cursorPos
			isHovered := i == m.hovered && !isSelected

			style := lipgloss.NewStyle()
			if isSelected {
				style = style.Background(primaryColor).Foreground(lipgloss.Color("0"))
			} else if isHovered {
				style = style.Background(dimColor).Foreground(lipgloss.Color("0"))
			} else {
				style = style.Border(lipgloss.RoundedBorder()).Padding(0, 1)
			}

			buttons = append(buttons, style.Render(" "+task.Name+" "))
		}
	}

	if len(buttons) > 0 {
		buttonsRow := lipgloss.JoinHorizontal(lipgloss.Top, buttons...)
		lines = append(lines, buttonsRow)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return m.wrapWithHelp(content, width)
}

func (m *model) getTaskName(name string, isSelected, isHovered bool) string {
	// Remove group prefix for display
	displayName := name
	if strings.Contains(name, ":") {
		parts := strings.Split(name, ":")
		displayName = parts[len(parts)-1]
	}

	if isSelected {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(primaryColor).
			Render(displayName)
	}
	if isHovered {
		return lipgloss.NewStyle().
			Background(dimColor).
			Foreground(lipgloss.Color("0")).
			Render(displayName)
	}
	return TaskNameStyle.Render(displayName)
}

func (m *model) wrapWithHelp(content string, width int) string {
	modeLabel := "list"
	if m.viewMode == modeHorizontal {
		modeLabel = "buttons"
	}
	helpText := fmt.Sprintf("[%s] ↑↓ navigate  Enter run  Tab mode  Ctrl+R reload  q quit", modeLabel)

	help := lipgloss.NewStyle().
		Width(width).
		Foreground(lipgloss.Color("245")).
		Background(lipgloss.Color("235")).
		Render(helpText)

	if m.scanning {
		status := lipgloss.NewStyle().
			Width(width).
			Foreground(lipgloss.Color("63")).
			Background(lipgloss.Color("235")).
			Render("Loading...")
		return lipgloss.JoinVertical(lipgloss.Left, status, content, help)
	}

	return lipgloss.JoinVertical(lipgloss.Left, content, help)
}

func Run(session string) error {
	p := tea.NewProgram(New(session),
		tea.WithMouseAllMotion(),
	)

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGWINCH)
		for {
			<-c
		}
	}()

	_, err := p.Run()
	return err
}
