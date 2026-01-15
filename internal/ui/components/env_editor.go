package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rizface/doui/internal/models"
	"github.com/rizface/doui/internal/ui/styles"
)

// EnvEditorMode represents the current editing mode
type EnvEditorMode int

const (
	EnvModeList EnvEditorMode = iota
	EnvModeAdd
	EnvModeEdit
)

// EnvVarItem implements list.Item for environment variables
type EnvVarItem struct {
	envVar models.EnvVar
}

func (i EnvVarItem) FilterValue() string { return i.envVar.Key }
func (i EnvVarItem) Title() string       { return i.envVar.Key }
func (i EnvVarItem) Description() string {
	// Truncate long values for display
	value := i.envVar.Value
	if len(value) > 60 {
		value = value[:57] + "..."
	}
	return value
}

// EnvEditor is a component for editing environment variables
type EnvEditor struct {
	list     list.Model
	envVars  []models.EnvVar
	original []models.EnvVar // For tracking changes

	// Editing state
	mode       EnvEditorMode
	keyInput   textinput.Model
	valueInput textinput.Model
	editIndex  int // Index being edited (-1 for add)

	// Dimensions
	width  int
	height int

	// State
	modified bool
}

// NewEnvEditor creates a new environment variable editor
func NewEnvEditor(envVars []models.EnvVar) *EnvEditor {
	// Setup list
	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(2)
	delegate.SetSpacing(0)

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Environment Variables"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = styles.TitleStyle

	// Setup text inputs
	keyInput := textinput.New()
	keyInput.Placeholder = "KEY_NAME"
	keyInput.CharLimit = 100
	keyInput.Width = 40

	valueInput := textinput.New()
	valueInput.Placeholder = "value"
	valueInput.CharLimit = 1000
	valueInput.Width = 60

	editor := &EnvEditor{
		list:       l,
		envVars:    make([]models.EnvVar, len(envVars)),
		original:   make([]models.EnvVar, len(envVars)),
		mode:       EnvModeList,
		keyInput:   keyInput,
		valueInput: valueInput,
		editIndex:  -1,
	}

	copy(editor.envVars, envVars)
	copy(editor.original, envVars)
	editor.updateList()

	return editor
}

func (e *EnvEditor) updateList() {
	items := make([]list.Item, len(e.envVars))
	for i, ev := range e.envVars {
		items[i] = EnvVarItem{envVar: ev}
	}
	e.list.SetItems(items)
}

// SetSize updates the editor dimensions
func (e *EnvEditor) SetSize(width, height int) {
	e.width = width
	e.height = height
	e.list.SetSize(width, height-4)
}

// GetEnvVars returns the current environment variables
func (e *EnvEditor) GetEnvVars() []models.EnvVar {
	return e.envVars
}

// IsModified returns true if env vars have been changed
func (e *EnvEditor) IsModified() bool {
	return e.modified
}

// Update handles messages
func (e *EnvEditor) Update(msg tea.Msg) (*EnvEditor, tea.Cmd) {
	switch e.mode {
	case EnvModeList:
		return e.updateListMode(msg)
	case EnvModeAdd, EnvModeEdit:
		return e.updateEditMode(msg)
	}
	return e, nil
}

func (e *EnvEditor) updateListMode(msg tea.Msg) (*EnvEditor, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "a", "n":
			// Add new env var
			e.mode = EnvModeAdd
			e.editIndex = -1
			e.keyInput.SetValue("")
			e.valueInput.SetValue("")
			e.keyInput.Focus()
			return e, nil

		case "e", "enter":
			// Edit selected
			if len(e.envVars) > 0 && e.list.Index() < len(e.envVars) {
				e.mode = EnvModeEdit
				e.editIndex = e.list.Index()
				e.keyInput.SetValue(e.envVars[e.editIndex].Key)
				e.valueInput.SetValue(e.envVars[e.editIndex].Value)
				e.keyInput.Focus()
			}
			return e, nil

		case "d", "delete":
			// Delete selected
			if len(e.envVars) > 0 && e.list.Index() < len(e.envVars) {
				idx := e.list.Index()
				e.envVars = append(e.envVars[:idx], e.envVars[idx+1:]...)
				e.modified = true
				e.updateList()
			}
			return e, nil
		}
	}

	var cmd tea.Cmd
	e.list, cmd = e.list.Update(msg)
	return e, cmd
}

func (e *EnvEditor) updateEditMode(msg tea.Msg) (*EnvEditor, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Cancel edit
			e.mode = EnvModeList
			return e, nil

		case "enter":
			// Save edit
			key := strings.TrimSpace(e.keyInput.Value())
			value := e.valueInput.Value()

			if key != "" {
				newVar := models.EnvVar{Key: key, Value: value}

				if e.mode == EnvModeAdd {
					e.envVars = append(e.envVars, newVar)
				} else if e.editIndex >= 0 && e.editIndex < len(e.envVars) {
					e.envVars[e.editIndex] = newVar
				}

				e.modified = true
				e.updateList()
			}

			e.mode = EnvModeList
			return e, nil

		case "tab":
			// Switch between key and value inputs
			if e.keyInput.Focused() {
				e.keyInput.Blur()
				e.valueInput.Focus()
			} else {
				e.valueInput.Blur()
				e.keyInput.Focus()
			}
			return e, nil
		}
	}

	// Update focused input
	var cmd tea.Cmd
	if e.keyInput.Focused() {
		e.keyInput, cmd = e.keyInput.Update(msg)
	} else {
		e.valueInput, cmd = e.valueInput.Update(msg)
	}
	return e, cmd
}

// View renders the editor
func (e *EnvEditor) View() string {
	var b strings.Builder

	switch e.mode {
	case EnvModeList:
		if len(e.envVars) == 0 {
			b.WriteString(styles.TitleStyle.Render("Environment Variables"))
			b.WriteString("\n\n")
			b.WriteString(styles.SubtitleStyle.Render("No environment variables defined."))
			b.WriteString("\n\n")
			b.WriteString(styles.DescStyle.Render("Press 'a' to add a new variable."))
		} else {
			b.WriteString(e.list.View())
		}

	case EnvModeAdd, EnvModeEdit:
		title := "Add Environment Variable"
		if e.mode == EnvModeEdit {
			title = "Edit Environment Variable"
		}
		b.WriteString(styles.TitleStyle.Render(title))
		b.WriteString("\n\n")

		// Key input
		keyLabel := "Key:   "
		if e.keyInput.Focused() {
			keyLabel = styles.KeyStyle.Render("Key:   ")
		}
		b.WriteString(keyLabel)
		b.WriteString(e.keyInput.View())
		b.WriteString("\n\n")

		// Value input
		valueLabel := "Value: "
		if e.valueInput.Focused() {
			valueLabel = styles.KeyStyle.Render("Value: ")
		}
		b.WriteString(valueLabel)
		b.WriteString(e.valueInput.View())
		b.WriteString("\n\n")

		b.WriteString(styles.DescStyle.Render("Tab: Switch field • Enter: Save • Esc: Cancel"))
	}

	// Wrap in a container
	content := b.String()
	return lipgloss.NewStyle().
		Padding(1, 2).
		Render(content)
}

// GetHelpText returns context-sensitive help
func (e *EnvEditor) GetHelpText() string {
	if e.mode != EnvModeList {
		return ""
	}

	helps := []string{
		styles.KeyStyle.Render("↑/↓") + " navigate",
		styles.KeyStyle.Render("a") + " add",
		styles.KeyStyle.Render("e") + " edit",
		styles.KeyStyle.Render("d") + " delete",
		styles.KeyStyle.Render("/") + " filter",
	}

	return strings.Join(helps, styles.SeparatorStyle.String())
}

// IsFiltering returns true if list is in filter mode
func (e *EnvEditor) IsFiltering() bool {
	return e.list.FilterState() == list.Filtering
}

// IsEditing returns true if in add/edit mode
func (e *EnvEditor) IsEditing() bool {
	return e.mode != EnvModeList
}
