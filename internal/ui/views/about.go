package views

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rizface/doui/internal/ui/styles"
)

// AboutView displays the about page with logo and app description
type AboutView struct {
	width  int
	height int
}

// NewAboutView creates a new about view
func NewAboutView() *AboutView {
	return &AboutView{}
}

// SetSize updates the view dimensions
func (v *AboutView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// Update handles messages
func (v *AboutView) Update(msg tea.Msg) (*AboutView, tea.Cmd) {
	return v, nil
}

// View renders the about page
func (v *AboutView) View() string {
	var content strings.Builder

	// ASCII Art Logo with gradient colors
	logoLines := []struct {
		text  string
		color lipgloss.Color
	}{
		{"    _            _", lipgloss.Color("#06B6D4")},  // Cyan
		{" __| | ___  _  _(_)", lipgloss.Color("#3B82F6")}, // Blue
		{"/ _` |/ _ \\| || | |", lipgloss.Color("#8B5CF6")}, // Purple
		{"\\__,_|\\___/ \\_,_|_|", lipgloss.Color("#EC4899")}, // Pink
	}

	// Build the logo
	var logoBuilder strings.Builder
	for _, line := range logoLines {
		style := lipgloss.NewStyle().
			Foreground(line.color).
			Bold(true)
		logoBuilder.WriteString(style.Render(line.text))
		logoBuilder.WriteString("\n")
	}

	// Version and tagline
	version := lipgloss.NewStyle().
		Foreground(styles.ColorMuted).
		Render("v1.0.0")

	tagline := lipgloss.NewStyle().
		Foreground(styles.ColorPrimary).
		Bold(true).
		Render("Docker UI for the Terminal")

	// Description
	description := `doui is a powerful Terminal User Interface (TUI) for managing
Docker resources. Built with Go and the Charm libraries, it provides
an intuitive way to interact with your Docker environment.`

	descStyle := lipgloss.NewStyle().
		Foreground(styles.ColorMuted).
		Width(60).
		Align(lipgloss.Center)

	// Features section
	featuresTitle := lipgloss.NewStyle().
		Foreground(styles.ColorInfo).
		Bold(true).
		Render("Features")

	features := []struct {
		icon string
		text string
	}{
		{"[ctr]", "Manage containers - start, stop, restart, remove"},
		{"[img]", "Browse and manage Docker images"},
		{"[grp]", "Group containers for batch operations"},
		{"[vol]", "View and manage Docker volumes"},
		{"[cmp]", "Docker Compose project management"},
		{"[net]", "Network management and container connections"},
		{"[log]", "Real-time container log streaming"},
		{"[sta]", "Live container resource statistics"},
	}

	var featuresBuilder strings.Builder
	for _, f := range features {
		icon := lipgloss.NewStyle().
			Foreground(styles.ColorAccent).
			Bold(true).
			Render(f.icon)
		text := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB")).
			Render(" " + f.text)
		featuresBuilder.WriteString("  " + icon + text + "\n")
	}

	// Author/credits
	credits := lipgloss.NewStyle().
		Foreground(styles.ColorMuted).
		Render("Built with Bubbletea, Bubbles & Lipgloss")

	github := lipgloss.NewStyle().
		Foreground(styles.ColorInfo).
		Render("github.com/rizface/doui")

	// Compose the content
	content.WriteString("\n\n")
	content.WriteString(logoBuilder.String())
	content.WriteString("\n")
	content.WriteString(version)
	content.WriteString("\n\n")
	content.WriteString(tagline)
	content.WriteString("\n\n")
	content.WriteString(descStyle.Render(description))
	content.WriteString("\n\n")
	content.WriteString(featuresTitle)
	content.WriteString("\n\n")
	content.WriteString(featuresBuilder.String())
	content.WriteString("\n")
	content.WriteString(credits)
	content.WriteString("\n")
	content.WriteString(github)

	// Center everything
	centered := lipgloss.NewStyle().
		Width(v.width).
		Height(v.height - 4).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content.String())

	return centered
}

// GetHelpText returns help text for the about view
func (v *AboutView) GetHelpText() string {
	helps := []string{
		styles.KeyStyle.Render("esc") + " back",
		styles.KeyStyle.Render("q") + " quit",
	}
	return strings.Join(helps, styles.SeparatorStyle.String())
}
