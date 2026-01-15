package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rizface/doui/internal/models"
	"github.com/rizface/doui/internal/ui/styles"
)

// Sidebar represents the left sidebar with tabs
type Sidebar struct {
	width        int
	height       int
	currentView  models.ViewType
}

// NewSidebar creates a new sidebar
func NewSidebar() *Sidebar {
	return &Sidebar{
		width:  20,
		currentView: models.ViewContainers,
	}
}

// SetSize sets the sidebar dimensions
func (s *Sidebar) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// SetCurrentView sets the currently active view
func (s *Sidebar) SetCurrentView(view models.ViewType) {
	s.currentView = view
}

// View renders the sidebar
func (s *Sidebar) View() string {
	var b strings.Builder

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

	for _, line := range logoLines {
		style := lipgloss.NewStyle().
			Foreground(line.color).
			Bold(true)
		b.WriteString(style.Render(line.text))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Tabs
	tabs := []struct {
		view  models.ViewType
		label string
	}{
		{models.ViewContainers, "Containers"},
		{models.ViewImages, "Images"},
		{models.ViewGroups, "Groups"},
		{models.ViewVolumes, "Volumes"},
		{models.ViewCompose, "Compose"},
		{models.ViewNetworks, "Networks"},
	}

	for _, tab := range tabs {
		var style lipgloss.Style
		prefix := "  "

		if tab.view == s.currentView {
			prefix = "→ "
			style = lipgloss.NewStyle().
				Bold(true).
				Foreground(styles.ColorPrimary).
				Padding(0, 1)
		} else {
			style = lipgloss.NewStyle().
				Foreground(styles.ColorMuted).
				Padding(0, 1)
		}

		tabText := prefix + tab.label
		b.WriteString(style.Render(tabText))
		b.WriteString("\n")
	}

	// Separator
	separator := lipgloss.NewStyle().
		Foreground(styles.ColorMuted).
		Render("  ──────────────")
	b.WriteString(separator)
	b.WriteString("\n")

	// About tab
	var aboutStyle lipgloss.Style
	aboutPrefix := "  "
	if s.currentView == models.ViewAbout {
		aboutPrefix = "→ "
		aboutStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(styles.ColorPrimary).
			Padding(0, 1)
	} else {
		aboutStyle = lipgloss.NewStyle().
			Foreground(styles.ColorMuted).
			Padding(0, 1)
	}
	b.WriteString(aboutStyle.Render(aboutPrefix + "About"))
	b.WriteString("\n")

	// Wrap in sidebar style
	sidebarStyle := lipgloss.NewStyle().
		Width(s.width).
		Height(s.height - 3). // Reserve space for footer
		BorderRight(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(styles.ColorMuted).
		Padding(1, 1)

	return sidebarStyle.Render(b.String())
}
