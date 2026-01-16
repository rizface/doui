package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rizface/doui/internal/models"
	"github.com/rizface/doui/internal/ui/styles"
)

// ContainerItem implements list.Item for containers
type ContainerItem struct {
	container  models.Container
	rebuilding bool
}

func (i ContainerItem) FilterValue() string {
	return i.container.Name
}

func (i ContainerItem) Title() string {
	if i.rebuilding {
		status := styles.WarningStyle.Render("rebuilding...")
		return fmt.Sprintf("%s  %s", i.container.Name, status)
	}
	status := styles.GetStatusStyle(i.container.State).Render(i.container.State)
	return fmt.Sprintf("%s  %s", i.container.Name, status)
}

func (i ContainerItem) Description() string {
	if i.rebuilding {
		return styles.SubtitleStyle.Render("Container is being rebuilt, please wait...")
	}
	return fmt.Sprintf("ID: %s | Image: %s | %s",
		i.container.ShortID,
		i.container.Image,
		i.container.Status)
}

// ContainersView displays the list of containers
type ContainersView struct {
	list            list.Model
	containers      []models.Container
	width           int
	height          int
	rebuildingName  string // Name of container currently being rebuilt
}

// NewContainersView creates a new containers view
func NewContainersView() *ContainersView {
	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(2)
	delegate.SetSpacing(1)

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Docker Containers"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = styles.TitleStyle

	return &ContainersView{
		list: l,
	}
}

// SetContainers updates the list of containers
func (v *ContainersView) SetContainers(containers []models.Container) {
	v.containers = containers

	items := make([]list.Item, len(containers))
	for i, c := range containers {
		rebuilding := v.rebuildingName != "" && c.Name == v.rebuildingName
		items[i] = ContainerItem{container: c, rebuilding: rebuilding}
	}

	v.list.SetItems(items)
}

// SetRebuilding marks a container as being rebuilt
func (v *ContainersView) SetRebuilding(containerName string) {
	v.rebuildingName = containerName
	// Refresh the list to show rebuilding status
	v.rebuildList()
}

// ClearRebuilding clears the rebuilding state
func (v *ContainersView) ClearRebuilding() {
	v.rebuildingName = ""
	v.rebuildList()
}

// IsRebuilding returns true if the given container is being rebuilt
func (v *ContainersView) IsRebuilding(containerName string) bool {
	return v.rebuildingName != "" && v.rebuildingName == containerName
}

// IsAnyRebuilding returns true if any container is being rebuilt
func (v *ContainersView) IsAnyRebuilding() bool {
	return v.rebuildingName != ""
}

// rebuildList rebuilds the list items with current state
func (v *ContainersView) rebuildList() {
	items := make([]list.Item, len(v.containers))
	for i, c := range v.containers {
		rebuilding := v.rebuildingName != "" && c.Name == v.rebuildingName
		items[i] = ContainerItem{container: c, rebuilding: rebuilding}
	}
	v.list.SetItems(items)
}

// SetSize updates the view dimensions
func (v *ContainersView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.list.SetSize(width, height-6) // Reserve space for header and footer
}

// Update handles messages
func (v *ContainersView) Update(msg tea.Msg) (*ContainersView, tea.Cmd) {
	// If filtering, pass all input directly to the list (for filter textinput)
	if v.IsFiltering() {
		var cmd tea.Cmd
		v.list, cmd = v.list.Update(msg)
		return v, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "s":
			// Start container
			if len(v.containers) > 0 && v.list.Index() < len(v.containers) {
				// Will be handled by parent app
				return v, nil
			}
		case "x":
			// Stop container
			if len(v.containers) > 0 && v.list.Index() < len(v.containers) {
				// Will be handled by parent app
				return v, nil
			}
		case "r":
			// Restart container
			if len(v.containers) > 0 && v.list.Index() < len(v.containers) {
				// Will be handled by parent app
				return v, nil
			}
		}
	}

	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}

// View renders the view
func (v *ContainersView) View() string {
	if len(v.containers) == 0 {
		return v.renderEmpty()
	}

	return v.list.View()
}

// GetSelectedContainer returns the currently selected container
func (v *ContainersView) GetSelectedContainer() *models.Container {
	item := v.list.SelectedItem()
	if item == nil {
		return nil
	}
	if containerItem, ok := item.(ContainerItem); ok {
		return &containerItem.container
	}
	return nil
}

// SelectByID selects a container by its ID
// Returns true if the container was found and selected
func (v *ContainersView) SelectByID(containerID string) bool {
	for i, c := range v.containers {
		if c.ID == containerID {
			v.list.Select(i)
			return true
		}
	}
	return false
}

func (v *ContainersView) renderEmpty() string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("Docker Containers"))
	b.WriteString("\n\n")
	b.WriteString(styles.SubtitleStyle.Render("No containers found. Start some Docker containers to see them here."))

	return b.String()
}

// IsFiltering returns true if the list is in filtering mode
func (v *ContainersView) IsFiltering() bool {
	return v.list.FilterState() == list.Filtering
}

// GetHelpText returns help text for the containers view
func (v *ContainersView) GetHelpText() string {
	helps := []string{
		styles.KeyStyle.Render("↑/↓") + " navigate",
		styles.KeyStyle.Render("s") + " start",
		styles.KeyStyle.Render("x") + " stop",
		styles.KeyStyle.Render("r") + " restart",
		styles.KeyStyle.Render("d") + " remove",
		styles.KeyStyle.Render("e") + " shell",
		styles.KeyStyle.Render("v") + " env",
		styles.KeyStyle.Render("l") + " logs",
		styles.KeyStyle.Render("t") + " stats",
		styles.KeyStyle.Render("/") + " filter",
		styles.KeyStyle.Render("q") + " quit",
	}

	return strings.Join(helps, styles.SeparatorStyle.String())
}
