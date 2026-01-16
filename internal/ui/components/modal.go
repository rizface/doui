package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rizface/doui/internal/ui/styles"
)

// ModalType represents the type of modal
type ModalType int

const (
	ModalConfirm ModalType = iota
	ModalForm
)

// Modal represents a modal dialog
type Modal struct {
	visible     bool
	modalType   ModalType
	title       string
	message     string
	confirmText string
	cancelText  string
	confirmed   bool
	width       int
	height      int

	// For form modals
	inputs         []textinput.Model
	focusIndex     int
	requiredFields []bool // true if field is required
}

// NewConfirmModal creates a new confirmation modal
func NewConfirmModal(title, message string) *Modal {
	return &Modal{
		visible:     true,
		modalType:   ModalConfirm,
		title:       title,
		message:     message,
		confirmText: "Yes",
		cancelText:  "No",
	}
}

// NewFormModal creates a new form modal with text inputs (all required)
func NewFormModal(title string, fieldLabels []string) *Modal {
	return NewFormModalWithOptional(title, fieldLabels, nil)
}

// NewFormModalWithOptional creates a new form modal with optional fields
// optionalFields is a slice of field indices that are optional
func NewFormModalWithOptional(title string, fieldLabels []string, optionalFields []int) *Modal {
	inputs := make([]textinput.Model, len(fieldLabels))
	requiredFields := make([]bool, len(fieldLabels))

	// Build set of optional field indices
	optionalSet := make(map[int]bool)
	for _, idx := range optionalFields {
		optionalSet[idx] = true
	}

	for i, label := range fieldLabels {
		ti := textinput.New()
		if optionalSet[i] {
			ti.Placeholder = label + " (optional)"
			requiredFields[i] = false
		} else {
			ti.Placeholder = label + " (required)"
			requiredFields[i] = true
		}
		ti.CharLimit = 200
		ti.Width = 50

		if i == 0 {
			ti.Focus()
		}

		inputs[i] = ti
	}

	return &Modal{
		visible:        true,
		modalType:      ModalForm,
		title:          title,
		confirmText:    "Create",
		cancelText:     "Cancel",
		inputs:         inputs,
		requiredFields: requiredFields,
	}
}

// Show shows the modal
func (m *Modal) Show() {
	m.visible = true
}

// Hide hides the modal
func (m *Modal) Hide() {
	m.visible = false
}

// IsVisible returns whether the modal is visible
func (m *Modal) IsVisible() bool {
	return m.visible
}

// IsConfirmed returns whether the user confirmed
func (m *Modal) IsConfirmed() bool {
	return m.confirmed
}

// GetInputValues returns the values from form inputs
func (m *Modal) GetInputValues() []string {
	values := make([]string, len(m.inputs))
	for i, input := range m.inputs {
		values[i] = input.Value()
	}
	return values
}

// Update handles messages
func (m *Modal) Update(msg tea.Msg) (*Modal, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.modalType == ModalConfirm {
				m.confirmed = true
				m.visible = false
				return m, nil
			} else if m.modalType == ModalForm {
				// Only confirm if all required fields are filled
				allFilled := true
				for i, input := range m.inputs {
					if m.requiredFields[i] && input.Value() == "" {
						allFilled = false
						break
					}
				}
				if allFilled {
					m.confirmed = true
					m.visible = false
					return m, nil
				}
			}

		case "y":
			// 'y' only confirms for confirmation modals, not form modals
			if m.modalType == ModalConfirm {
				m.confirmed = true
				m.visible = false
				return m, nil
			}
			// For form modals, let 'y' pass through to the text input

		case "esc":
			// Esc always cancels
			m.confirmed = false
			m.visible = false
			return m, nil

		case "n":
			// 'n' only cancels for confirmation modals, not form modals
			if m.modalType == ModalConfirm {
				m.confirmed = false
				m.visible = false
				return m, nil
			}
			// For form modals, let 'n' pass through to the text input

		case "tab", "shift+tab", "up", "down":
			if m.modalType == ModalForm {
				// Navigate between inputs
				if msg.String() == "tab" || msg.String() == "down" {
					m.focusIndex++
					if m.focusIndex >= len(m.inputs) {
						m.focusIndex = 0
					}
				} else {
					m.focusIndex--
					if m.focusIndex < 0 {
						m.focusIndex = len(m.inputs) - 1
					}
				}

				// Update focus
				for i := range m.inputs {
					if i == m.focusIndex {
						m.inputs[i].Focus()
					} else {
						m.inputs[i].Blur()
					}
				}
			}
		}
	}

	// Update active input
	if m.modalType == ModalForm && m.focusIndex < len(m.inputs) {
		var cmd tea.Cmd
		m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the modal
func (m *Modal) View() string {
	if !m.visible {
		return ""
	}

	var content strings.Builder

	// Title
	content.WriteString(styles.TitleStyle.Render(m.title))
	content.WriteString("\n\n")

	// Content based on type
	switch m.modalType {
	case ModalConfirm:
		content.WriteString(m.message)
		content.WriteString("\n\n")

		// Buttons
		confirmBtn := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(styles.ColorSuccess).
			Padding(0, 2).
			Render(m.confirmText)

		cancelBtn := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(styles.ColorMuted).
			Padding(0, 2).
			Render(m.cancelText)

		content.WriteString(confirmBtn + "  " + cancelBtn)

	case ModalForm:
		// Render inputs
		for i, input := range m.inputs {
			content.WriteString(input.View())
			if i < len(m.inputs)-1 {
				content.WriteString("\n")
			}
		}
		content.WriteString("\n\n")

		// Buttons
		confirmBtn := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(styles.ColorPrimary).
			Padding(0, 2).
			Render(m.confirmText)

		cancelBtn := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(styles.ColorMuted).
			Padding(0, 2).
			Render(m.cancelText)

		content.WriteString(confirmBtn + "  " + cancelBtn)
		content.WriteString("\n\n")
		content.WriteString(styles.DescStyle.Render("Tab: Next field • Enter: Submit • Esc: Cancel"))
	}

	// Wrap in modal style
	modalContent := styles.ModalStyle.Render(content.String())

	// Center the modal
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modalContent,
	)
}

// SetSize sets the modal dimensions for centering
func (m *Modal) SetSize(width, height int) {
	m.width = width
	m.height = height
}
