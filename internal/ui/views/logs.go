package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rizface/doui/internal/docker"
	"github.com/rizface/doui/internal/ui/styles"
)

// LogsView displays container logs
type LogsView struct {
	viewport     viewport.Model
	lines        []string
	follow       bool
	maxLines     int
	containerID  string
	containerName string
	logsChan     <-chan docker.LogEntry
	errorChan    <-chan error
	ready        bool
	width        int
	height       int
}

// NewLogsView creates a new logs view
func NewLogsView() *LogsView {
	vp := viewport.New(0, 0)
	vp.Style = styles.BorderStyle

	return &LogsView{
		viewport: vp,
		lines:    []string{},
		follow:   true,
		maxLines: 1000,
		ready:    false,
	}
}

// SetContainer sets the container to view logs for
func (v *LogsView) SetContainer(containerID, containerName string) {
	v.containerID = containerID
	v.containerName = containerName
	v.lines = []string{}
	v.ready = false // Reset ready so View() shows loading state until StartStreaming is called
}

// StartStreaming starts streaming logs
func (v *LogsView) StartStreaming(logsChan <-chan docker.LogEntry, errorChan <-chan error) {
	v.logsChan = logsChan
	v.errorChan = errorChan
	v.ready = true
}

// SetSize updates the view dimensions
func (v *LogsView) SetSize(width, height int) {
	v.width = width
	v.height = height
	headerHeight := 3
	v.viewport.Width = width - 4
	v.viewport.Height = height - headerHeight - 4
}

// ToggleFollow toggles follow mode
func (v *LogsView) ToggleFollow() {
	v.follow = !v.follow
}

// Update handles messages
func (v *LogsView) Update(msg tea.Msg) (*LogsView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "f":
			v.ToggleFollow()
			if v.follow {
				v.viewport.GotoBottom()
			}
			return v, nil
		case "g":
			v.viewport.GotoTop()
			return v, nil
		case "G":
			v.viewport.GotoBottom()
			return v, nil
		}

	case docker.LogEntry:
		// Add new log line
		v.lines = append(v.lines, msg.Line)

		// Limit lines to maxLines (circular buffer)
		if len(v.lines) > v.maxLines {
			v.lines = v.lines[len(v.lines)-v.maxLines:]
		}

		// Update viewport content
		v.viewport.SetContent(strings.Join(v.lines, "\n"))

		// Auto-scroll if follow mode is enabled
		if v.follow {
			v.viewport.GotoBottom()
		}

		// Wait for next log entry
		return v, waitForLogEntry(v.logsChan, v.errorChan)
	}

	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
}

// View renders the view
func (v *LogsView) View() string {
	if !v.ready {
		return "Loading logs..."
	}

	var b strings.Builder

	// Header
	shortID := v.containerID
	if len(shortID) > 12 {
		shortID = shortID[:12]
	}
	title := fmt.Sprintf("Logs: %s (%s)", v.containerName, shortID)
	b.WriteString(styles.TitleStyle.Render(title))
	b.WriteString("\n")

	// Follow mode indicator
	followStatus := "Follow: OFF"
	if v.follow {
		followStatus = styles.SuccessStyle.Render("Follow: ON")
	}
	b.WriteString(followStatus)
	b.WriteString("\n\n")

	// Viewport with logs
	b.WriteString(v.viewport.View())

	return b.String()
}

// GetHelpText returns help text for the logs view
func (v *LogsView) GetHelpText() string {
	helps := []string{
		styles.KeyStyle.Render("↑/↓") + " scroll",
		styles.KeyStyle.Render("f") + " toggle follow",
		styles.KeyStyle.Render("g/G") + " top/bottom",
		styles.KeyStyle.Render("esc") + " back",
		styles.KeyStyle.Render("q") + " quit",
	}

	return strings.Join(helps, styles.SeparatorStyle.String())
}

// waitForLogEntry returns a command that waits for the next log entry
func waitForLogEntry(logsChan <-chan docker.LogEntry, errorChan <-chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case entry, ok := <-logsChan:
			if !ok {
				return nil
			}
			return entry
		case err, ok := <-errorChan:
			if !ok {
				return nil
			}
			return err
		}
	}
}
