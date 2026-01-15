package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/rizface/doui/internal/ui/styles"
)

// Header represents the application header
type Header struct {
	width int
}

// NewHeader creates a new header
func NewHeader() *Header {
	return &Header{}
}

// SetSize sets the header dimensions
func (h *Header) SetSize(width int) {
	h.width = width
}

// View renders the header
func (h *Header) View(title string) string {
	headerStyle := lipgloss.NewStyle().
		Width(h.width).
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(styles.ColorPrimary).
		Padding(0, 2)

	return headerStyle.Render("🐳 " + title)
}
