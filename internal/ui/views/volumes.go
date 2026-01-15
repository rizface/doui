package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rizface/doui/internal/models"
	"github.com/rizface/doui/internal/ui/styles"
)

// VolumeItem implements list.Item for volumes
type VolumeItem struct {
	volume models.Volume
}

func (i VolumeItem) FilterValue() string {
	return i.volume.Name
}

func (i VolumeItem) Title() string {
	status := ""
	if i.volume.IsInUse() {
		status = styles.RunningStyle.Render("in use")
	} else {
		status = styles.StoppedStyle.Render("unused")
	}
	return fmt.Sprintf("%s  %s", i.volume.GetShortName(), status)
}

func (i VolumeItem) Description() string {
	driver := i.volume.GetDriver()
	refCount := 0
	if i.volume.UsageData != nil {
		refCount = i.volume.UsageData.RefCount
	}
	return fmt.Sprintf("Driver: %s | Containers: %d | %s", driver, refCount, i.volume.Mountpoint)
}

// VolumesView displays the list of volumes
type VolumesView struct {
	list          list.Model
	volumes       []models.Volume
	allContainers []models.Container
	width         int
	height        int
}

// NewVolumesView creates a new volumes view
func NewVolumesView() *VolumesView {
	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(2)
	delegate.SetSpacing(1)

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Docker Volumes"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = styles.TitleStyle

	return &VolumesView{
		list: l,
	}
}

// SetVolumes updates the list of volumes
func (v *VolumesView) SetVolumes(volumes []models.Volume) {
	v.volumes = volumes
	v.syncVolumeContainerCounts()
}

// SetAllContainers updates the list of all containers for volume usage calculation
func (v *VolumesView) SetAllContainers(containers []models.Container) {
	v.allContainers = containers
	v.syncVolumeContainerCounts()
}

// syncVolumeContainerCounts populates each volume's UsageData from container mount data
func (v *VolumesView) syncVolumeContainerCounts() {
	if len(v.volumes) == 0 {
		return
	}

	// Build map of volume name -> container count
	volumeUsage := make(map[string]int)
	for _, c := range v.allContainers {
		for _, m := range c.Mounts {
			if m.Type == "volume" && m.Name != "" {
				volumeUsage[m.Name]++
			}
		}
	}

	// Update each volume's UsageData
	for i := range v.volumes {
		refCount := volumeUsage[v.volumes[i].Name]
		if v.volumes[i].UsageData == nil {
			v.volumes[i].UsageData = &models.VolumeUsageData{
				RefCount: refCount,
				Size:     -1,
			}
		} else {
			v.volumes[i].UsageData.RefCount = refCount
		}
	}

	// Rebuild the list items with updated counts
	items := make([]list.Item, len(v.volumes))
	for i, vol := range v.volumes {
		items[i] = VolumeItem{volume: vol}
	}
	v.list.SetItems(items)
}

// SetSize updates the view dimensions
func (v *VolumesView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.list.SetSize(width, height-6) // Reserve space for header and footer
}

// Update handles messages
func (v *VolumesView) Update(msg tea.Msg) (*VolumesView, tea.Cmd) {
	// If filtering, pass all input directly to the list
	if v.IsFiltering() {
		var cmd tea.Cmd
		v.list, cmd = v.list.Update(msg)
		return v, cmd
	}

	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}

// View renders the view
func (v *VolumesView) View() string {
	if len(v.volumes) == 0 {
		return v.renderEmpty()
	}

	return v.list.View()
}

// GetSelectedVolume returns the currently selected volume
func (v *VolumesView) GetSelectedVolume() *models.Volume {
	if len(v.volumes) == 0 || v.list.Index() >= len(v.volumes) {
		return nil
	}
	return &v.volumes[v.list.Index()]
}

func (v *VolumesView) renderEmpty() string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("Docker Volumes"))
	b.WriteString("\n\n")
	b.WriteString(styles.SubtitleStyle.Render("No volumes found."))

	return b.String()
}

// IsFiltering returns true if the list is in filtering mode
func (v *VolumesView) IsFiltering() bool {
	return v.list.FilterState() == list.Filtering
}

// GetHelpText returns help text for the volumes view
func (v *VolumesView) GetHelpText() string {
	helps := []string{
		styles.KeyStyle.Render("↑/↓") + " navigate",
		styles.KeyStyle.Render("d") + " remove",
		styles.KeyStyle.Render("p") + " prune unused",
		styles.KeyStyle.Render("/") + " filter",
		styles.KeyStyle.Render("q") + " quit",
	}

	return strings.Join(helps, styles.SeparatorStyle.String())
}
