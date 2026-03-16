package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	PrimaryBG = lipgloss.Color("#1A1B26")
	Accent    = lipgloss.Color("#7AA2F7")
	Action    = lipgloss.Color("#BB9AF7")
	Match     = lipgloss.Color("#9ECE6A")
	Missing   = lipgloss.Color("#F7768E")
	Muted     = lipgloss.Color("#565F89")

	// Styles
	MainStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Background(PrimaryBG)

	TitleStyle = lipgloss.NewStyle().
			Foreground(Accent).
			Bold(true).
			MarginBottom(1)

	MutedStyle = lipgloss.NewStyle().
			Foreground(Muted)
)
