package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/rizface/doui/internal/ui/styles"
)

// Footer represents the application footer/legend
type Footer struct {
	width int
}

// NewFooter creates a new footer
func NewFooter() *Footer {
	return &Footer{}
}

// SetSize sets the footer dimensions
func (f *Footer) SetSize(width int) {
	f.width = width
}

// View renders the footer with help text or status message
func (f *Footer) View(content string) string {
	footerStyle := lipgloss.NewStyle().
		Width(f.width).
		BorderTop(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(styles.ColorMuted).
		Padding(0, 1)

	return footerStyle.Render(content)
}
