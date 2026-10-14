package ui

import "github.com/charmbracelet/lipgloss"

var (
	primaryColor   = lipgloss.Color("63")
	secondaryColor = lipgloss.Color("245")
	dimColor       = lipgloss.Color("240")
	errorColor     = lipgloss.Color("196")

	NormalItemStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	TaskNameStyle = lipgloss.NewStyle().
			Foreground(primaryColor)

	TaskDescStyle = lipgloss.NewStyle().
			Foreground(dimColor)

	OutputStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor)

	GroupStyle = lipgloss.NewStyle().
			Foreground(accentColor)
)

var accentColor = lipgloss.Color("75")

var dimTextStyle = lipgloss.NewStyle().Foreground(dimColor)
