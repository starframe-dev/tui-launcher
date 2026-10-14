package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	primaryColor = lipgloss.Color("63")
	dimColor     = lipgloss.Color("240")
	greenColor   = lipgloss.Color("82")
)

type model struct {
	running  bool
	finished bool
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "enter":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		// Ignore resize
	}
	return m, nil
}

func (m *model) View() string {
	status := lipgloss.NewStyle().
		Foreground(greenColor).
		Render("● Running")

	if m.finished {
		status = lipgloss.NewStyle().
			Foreground(dimColor).
			Render("○ Done")
	}

	closeBtn := lipgloss.NewStyle().
		Background(primaryColor).
		Foreground(lipgloss.Color("0")).
		Padding(0, 2).
		Render("[ q ] close")

	return lipgloss.NewStyle().
		Width(50).
		Border(lipgloss.RoundedBorder()).
		Padding(1).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				status,
				"",
				closeBtn,
				"",
				lipgloss.NewStyle().
					Foreground(dimColor).
					Render("Press q or Enter"),
			),
		)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: wrapper <command>")
		os.Exit(1)
	}

	command := os.Args[1]

	// Start command in background, inheriting stdin/stdout/stderr
	cmd := exec.Command("sh", "-c", command+"; exit 0")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		fmt.Println("Error starting command:", err)
		os.Exit(1)
	}

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		cmd.Wait()
	}()

	p := tea.NewProgram(&model{})

	// Wait for either command to finish or user to quit
	go func() {
		<-sigChan
		cmd.Process.Kill()
		p.Quit()
	}()

	if err := p.Start(); err != nil {
		cmd.Process.Kill()
	}
}
