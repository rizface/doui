package views

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rizface/doui/internal/models"
	"github.com/rizface/doui/internal/ui/styles"
)

// StatsView displays container statistics
type StatsView struct {
	stats         *models.ContainerStats
	history       []models.ContainerStats
	maxHistory    int
	containerID   string
	containerName string
	statsChan     <-chan *models.ContainerStats
	errorChan     <-chan error
	ready         bool
	width         int
	height        int
}

// NewStatsView creates a new stats view
func NewStatsView() *StatsView {
	return &StatsView{
		history:    []models.ContainerStats{},
		maxHistory: 60, // Keep last 60 data points
		ready:      false,
	}
}

// SetContainer sets the container to monitor
func (v *StatsView) SetContainer(containerID, containerName string) {
	v.containerID = containerID
	v.containerName = containerName
	v.stats = nil
	v.history = []models.ContainerStats{}
}

// StartStreaming starts streaming stats
func (v *StatsView) StartStreaming(statsChan <-chan *models.ContainerStats, errorChan <-chan error) {
	v.statsChan = statsChan
	v.errorChan = errorChan
	v.ready = true
}

// SetSize updates the view dimensions
func (v *StatsView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// Update handles messages
func (v *StatsView) Update(msg tea.Msg) (*StatsView, tea.Cmd) {
	switch msg := msg.(type) {
	case *models.ContainerStats:
		v.stats = msg

		// Add to history
		v.history = append(v.history, *msg)
		if len(v.history) > v.maxHistory {
			v.history = v.history[1:]
		}

		// Wait for next stats update
		return v, waitForStats(v.statsChan, v.errorChan)
	}

	return v, nil
}

// View renders the view
func (v *StatsView) View() string {
	if !v.ready {
		return "Loading stats..."
	}

	if v.stats == nil {
		return "Waiting for stats data..."
	}

	var b strings.Builder

	// Header
	title := fmt.Sprintf("Stats: %s (%s)", v.containerName, v.containerID[:12])
	b.WriteString(styles.TitleStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(styles.SubtitleStyle.Render(fmt.Sprintf("Updated: %s", v.stats.Timestamp.Format(time.RFC3339))))
	b.WriteString("\n\n")

	// CPU Usage
	b.WriteString(v.renderMetric("CPU Usage", v.stats.CPUPercent, "%", 100))
	b.WriteString("\n")

	// Memory Usage
	memUsageMB := float64(v.stats.MemoryUsage) / 1024 / 1024
	memLimitMB := float64(v.stats.MemoryLimit) / 1024 / 1024
	memLabel := fmt.Sprintf("Memory (%.1f MB / %.1f MB)", memUsageMB, memLimitMB)
	b.WriteString(v.renderMetric(memLabel, v.stats.MemoryPercent, "%", 100))
	b.WriteString("\n")

	// Network I/O
	netRxMB := float64(v.stats.NetworkRx) / 1024 / 1024
	netTxMB := float64(v.stats.NetworkTx) / 1024 / 1024
	b.WriteString(styles.KeyStyle.Render("Network I/O: "))
	b.WriteString(fmt.Sprintf("↓ %.2f MB  ↑ %.2f MB", netRxMB, netTxMB))
	b.WriteString("\n")

	// Block I/O
	blockReadMB := float64(v.stats.BlockRead) / 1024 / 1024
	blockWriteMB := float64(v.stats.BlockWrite) / 1024 / 1024
	b.WriteString(styles.KeyStyle.Render("Block I/O:   "))
	b.WriteString(fmt.Sprintf("Read: %.2f MB  Write: %.2f MB", blockReadMB, blockWriteMB))
	b.WriteString("\n")

	// PIDs
	b.WriteString(styles.KeyStyle.Render("PIDs:        "))
	b.WriteString(fmt.Sprintf("%d", v.stats.PIDs))
	b.WriteString("\n")

	return b.String()
}

// renderMetric renders a metric with a progress bar
func (v *StatsView) renderMetric(label string, value float64, unit string, max float64) string {
	// Calculate percentage
	percent := value
	if max > 0 && value > max {
		percent = 100
	}

	// Create progress bar
	barWidth := 40
	filled := int((percent / 100) * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	// Color based on usage
	var barStyle lipgloss.Style
	if percent >= 90 {
		barStyle = lipgloss.NewStyle().Foreground(styles.ColorDanger)
	} else if percent >= 70 {
		barStyle = lipgloss.NewStyle().Foreground(styles.ColorWarning)
	} else {
		barStyle = lipgloss.NewStyle().Foreground(styles.ColorSuccess)
	}

	return fmt.Sprintf("%s [%s] %.1f%s",
		styles.KeyStyle.Render(label+":"),
		barStyle.Render(bar),
		value,
		unit,
	)
}

// GetHelpText returns help text for the stats view
func (v *StatsView) GetHelpText() string {
	helps := []string{
		styles.KeyStyle.Render("esc") + " back",
		styles.KeyStyle.Render("q") + " quit",
	}

	return strings.Join(helps, styles.SeparatorStyle.String())
}

// waitForStats returns a command that waits for the next stats update
func waitForStats(statsChan <-chan *models.ContainerStats, errorChan <-chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case stats, ok := <-statsChan:
			if !ok {
				return nil
			}
			return stats
		case err, ok := <-errorChan:
			if !ok {
				return nil
			}
			return err
		}
	}
}
