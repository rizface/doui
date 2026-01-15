package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rizface/doui/internal/models"
	"github.com/rizface/doui/internal/ui/components"
	"github.com/rizface/doui/internal/ui/styles"
)

// EnvVarsView is a full-screen view for editing container env vars
type EnvVarsView struct {
	editor        *components.EnvEditor
	containerID   string
	containerName string
	originalEnv   []models.EnvVar
	width         int
	height        int
	ready         bool
}

// NewEnvVarsView creates a new environment variables view
func NewEnvVarsView() *EnvVarsView {
	return &EnvVarsView{
		ready: false,
	}
}

// SetContainer initializes the view with container data
func (v *EnvVarsView) SetContainer(containerID, containerName string, env []string) {
	v.containerID = containerID
	v.containerName = containerName
	v.originalEnv = models.ParseEnvVars(env)
	v.editor = components.NewEnvEditor(v.originalEnv)
	v.editor.SetSize(v.width, v.height-6)
	v.ready = true
}

// SetSize updates the view dimensions
func (v *EnvVarsView) SetSize(width, height int) {
	v.width = width
	v.height = height
	if v.editor != nil {
		v.editor.SetSize(width, height-6)
	}
}

// GetEnvVars returns the current environment variables as strings
func (v *EnvVarsView) GetEnvVars() []string {
	if v.editor == nil {
		return nil
	}
	return models.EnvVarsToStrings(v.editor.GetEnvVars())
}

// IsModified returns true if changes were made
func (v *EnvVarsView) IsModified() bool {
	return v.editor != nil && v.editor.IsModified()
}

// Update handles messages
func (v *EnvVarsView) Update(msg tea.Msg) (*EnvVarsView, tea.Cmd) {
	if v.editor == nil {
		return v, nil
	}

	var cmd tea.Cmd
	v.editor, cmd = v.editor.Update(msg)
	return v, cmd
}

// View renders the view
func (v *EnvVarsView) View() string {
	if !v.ready || v.editor == nil {
		return "Loading environment variables..."
	}

	var b strings.Builder

	// Header
	shortID := v.containerID
	if len(shortID) > 12 {
		shortID = shortID[:12]
	}
	title := fmt.Sprintf("Environment Variables: %s (%s)", v.containerName, shortID)
	b.WriteString(styles.TitleStyle.Render(title))
	b.WriteString("\n")

	// Modified indicator and save hint
	if v.editor.IsModified() {
		b.WriteString(styles.WarningStyle.Render("[Modified] "))
	}
	b.WriteString(styles.DescStyle.Render("Press Ctrl+S to save and rebuild container"))
	b.WriteString("\n\n")

	// Editor
	b.WriteString(v.editor.View())

	return b.String()
}

// GetHelpText returns help text
func (v *EnvVarsView) GetHelpText() string {
	if v.editor == nil {
		return ""
	}

	var helps []string

	editorHelp := v.editor.GetHelpText()
	if editorHelp != "" {
		helps = append(helps, editorHelp)
	}

	if v.editor.IsModified() {
		helps = append(helps, styles.KeyStyle.Render("ctrl+s")+" save & rebuild")
	}
	helps = append(helps, styles.KeyStyle.Render("esc")+" back (discard)")

	return strings.Join(helps, styles.SeparatorStyle.String())
}

// IsFiltering returns true if editor is filtering
func (v *EnvVarsView) IsFiltering() bool {
	return v.editor != nil && v.editor.IsFiltering()
}

// IsEditing returns true if editor is in add/edit mode
func (v *EnvVarsView) IsEditing() bool {
	return v.editor != nil && v.editor.IsEditing()
}
