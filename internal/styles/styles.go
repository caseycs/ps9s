package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("86")
	secondaryColor = lipgloss.Color("205")
	successColor   = lipgloss.Color("42")
	errorColor     = lipgloss.Color("196")
	subtleColor    = lipgloss.Color("240")

	// Styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	LabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(secondaryColor)

	ValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtleColor)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	HelpStyle = lipgloss.NewStyle().
			Foreground(subtleColor).
			MarginTop(1)

	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
)
