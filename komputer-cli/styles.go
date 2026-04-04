package main

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7C3AED"))
	labelStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#6B7280"))
	valueStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#F9FAFB"))
	successStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#10B981"))
	errorStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EF4444"))
	warnStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F59E0B"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	busyStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#3B82F6"))
	idleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#10B981"))

	// Event type styles
	eventTextStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#F9FAFB"))
	eventThinkStyle    = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#A78BFA"))
	eventToolStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#10B981"))
	eventResultStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#6EE7B7"))
	eventErrorStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EF4444"))
	eventStartStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#3B82F6"))
	eventCompleteStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#10B981"))

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED")).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(lipgloss.Color("#4C1D95")).
			PaddingBottom(1).
			MarginBottom(1)
)
