package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Color palette
	ColorPrimary   = lipgloss.Color("#7C3AED") // Purple
	ColorSecondary = lipgloss.Color("#10B981") // Green
	ColorAccent    = lipgloss.Color("#F59E0B") // Orange
	ColorDanger    = lipgloss.Color("#EF4444") // Red
	ColorMuted     = lipgloss.Color("#6B7280") // Gray
	ColorSuccess   = lipgloss.Color("#10B981") // Green
	ColorWarning   = lipgloss.Color("#F59E0B") // Orange
	ColorInfo      = lipgloss.Color("#3B82F6") // Blue

	// Text styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorDanger).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	StatusStyle = lipgloss.NewStyle().
			Foreground(ColorInfo)

	// Component styles
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorPrimary).
			Padding(0, 1)

	FooterStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			Padding(0, 1)

	TabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorPrimary).
			Padding(0, 2)

	TabInactiveStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Padding(0, 2)

	// List/Table styles
	SelectedItemStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true).
				PaddingLeft(2)

	NormalItemStyle = lipgloss.NewStyle().
			PaddingLeft(4)

	// Container status colors
	RunningStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	StoppedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	PausedStyle = lipgloss.NewStyle().
			Foreground(ColorWarning)

	// Borders and containers
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted).
			Padding(1, 2)

	ModalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2).
			Background(lipgloss.Color("#1F2937"))

	// Key binding hints
	KeyStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	DescStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	SeparatorStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			SetString(" • ")

	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true)
)

// GetStatusStyle returns appropriate style for container status
func GetStatusStyle(status string) lipgloss.Style {
	switch status {
	case "running":
		return RunningStyle
	case "exited", "stopped":
		return StoppedStyle
	case "paused":
		return PausedStyle
	default:
		return NormalItemStyle
	}
}
