package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("#7D56F4")
	secondaryColor = lipgloss.Color("#5A4FCF")
	successColor   = lipgloss.Color("#04B575")
	warningColor   = lipgloss.Color("#FFCC00")
	errorColor     = lipgloss.Color("#FF5555")
	subtleColor    = lipgloss.Color("#626262")
	textColor      = lipgloss.Color("#FAFAFA")
	dimTextColor   = lipgloss.Color("#A0A0A0")

	// Title style
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	// Version style (dimmer, next to title)
	versionStyle = lipgloss.NewStyle().
			Foreground(subtleColor)

	// List styles
	listItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(primaryColor).
				Bold(true)

	// Host info styles
	hostAliasStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Bold(true)

	hostDetailsStyle = lipgloss.NewStyle().
				Foreground(dimTextColor)

	// Status indicator styles
	connectedStyle = lipgloss.NewStyle().
			Foreground(successColor)

	disconnectedStyle = lipgloss.NewStyle().
				Foreground(subtleColor)

	// Help style
	helpStyle = lipgloss.NewStyle().
			Foreground(subtleColor).
			MarginTop(1)

	// Form styles
	focusedStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	blurredStyle = lipgloss.NewStyle().
			Foreground(dimTextColor)

	cursorStyle = lipgloss.NewStyle().
			Foreground(primaryColor)

	inputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1)

	// Error/Status messages
	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(warningColor)

	// Container styles
	containerStyle = lipgloss.NewStyle().
			Padding(1, 2)

	// Button styles
	buttonStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Background(primaryColor).
			Padding(0, 2).
			MarginRight(1)

	activeButtonStyle = lipgloss.NewStyle().
				Foreground(textColor).
				Background(secondaryColor).
				Padding(0, 2).
				MarginRight(1).
				Bold(true)
)
