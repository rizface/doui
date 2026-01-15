package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rizface/doui/internal/models"
	"github.com/rizface/doui/internal/ui/styles"
)

// GroupItem implements list.Item for groups
type GroupItem struct {
	group models.Group
}

func (i GroupItem) FilterValue() string {
	return i.group.Name
}

func (i GroupItem) Title() string {
	return fmt.Sprintf("%s (%d containers)", i.group.Name, len(i.group.ContainerIDs))
}

func (i GroupItem) Description() string {
	if i.group.Description != "" {
		return i.group.Description
	}
	return "No description"
}

// ContainerItemForGroup implements list.Item for containers in groups view
type ContainerItemForGroup struct {
	container models.Container
}

func (i ContainerItemForGroup) FilterValue() string {
	return i.container.Name
}

func (i ContainerItemForGroup) Title() string {
	status := styles.GetStatusStyle(i.container.State).Render(i.container.State)
	return fmt.Sprintf("%s  %s", i.container.Name, status)
}

func (i ContainerItemForGroup) Description() string {
	return fmt.Sprintf("ID: %s | Image: %s", i.container.ShortID, i.container.Image)
}

// GroupsView displays the tabbed groups management interface
type GroupsView struct {
	// Tab state
	currentTab models.GroupsTabType

	// Data
	groups        []models.Group
	allContainers []models.Container
	selectedGroup *models.Group

	// List models for each tab
	groupsList              list.Model
	containersInGroupList   list.Model
	availableContainersList list.Model

	// Dimensions
	width  int
	height int
}

// NewGroupsView creates a new groups view
func NewGroupsView() *GroupsView {
	// Initialize groups list
	groupsDelegate := list.NewDefaultDelegate()
	groupsDelegate.SetHeight(2)
	groupsDelegate.SetSpacing(1)

	groupsList := list.New([]list.Item{}, groupsDelegate, 0, 0)
	groupsList.Title = "Container Groups"
	groupsList.SetShowStatusBar(true)
	groupsList.SetFilteringEnabled(true)
	groupsList.Styles.Title = styles.TitleStyle

	// Initialize containers in group list
	containersDelegate := list.NewDefaultDelegate()
	containersDelegate.SetHeight(2)
	containersDelegate.SetSpacing(1)

	containersInGroupList := list.New([]list.Item{}, containersDelegate, 0, 0)
	containersInGroupList.Title = "Containers in Group"
	containersInGroupList.SetShowStatusBar(true)
	containersInGroupList.SetFilteringEnabled(true)
	containersInGroupList.Styles.Title = styles.TitleStyle

	// Initialize available containers list
	availableDelegate := list.NewDefaultDelegate()
	availableDelegate.SetHeight(2)
	availableDelegate.SetSpacing(1)

	availableContainersList := list.New([]list.Item{}, availableDelegate, 0, 0)
	availableContainersList.Title = "Available Containers"
	availableContainersList.SetShowStatusBar(true)
	availableContainersList.SetFilteringEnabled(true)
	availableContainersList.Styles.Title = styles.TitleStyle

	return &GroupsView{
		currentTab:              models.GroupsListTab,
		groupsList:              groupsList,
		containersInGroupList:   containersInGroupList,
		availableContainersList: availableContainersList,
	}
}

// SetGroups updates the list of groups
func (v *GroupsView) SetGroups(groups []models.Group) {
	v.groups = groups

	items := make([]list.Item, len(groups))
	for i, g := range groups {
		items[i] = GroupItem{group: g}
	}

	v.groupsList.SetItems(items)
}

// SetAllContainers updates the list of all containers
func (v *GroupsView) SetAllContainers(containers []models.Container) {
	v.allContainers = containers
	v.updateContainerLists()
}

// SetSize updates the view dimensions
func (v *GroupsView) SetSize(width, height int) {
	v.width = width
	v.height = height

	// Account for tab bar (3 lines) and reduce height accordingly
	listHeight := height - 9
	v.groupsList.SetSize(width, listHeight)
	v.containersInGroupList.SetSize(width, listHeight)
	v.availableContainersList.SetSize(width, listHeight)
}

// GetSelectedGroup returns the currently selected group
func (v *GroupsView) GetSelectedGroup() *models.Group {
	if len(v.groups) == 0 || v.groupsList.Index() >= len(v.groups) {
		return nil
	}
	return &v.groups[v.groupsList.Index()]
}

// GetSelectedInGroupContainer returns the selected container from the "In Group" tab
func (v *GroupsView) GetSelectedInGroupContainer() *models.Container {
	containers := v.GetContainersInGroup()
	if len(containers) == 0 || v.containersInGroupList.Index() >= len(containers) {
		return nil
	}
	return &containers[v.containersInGroupList.Index()]
}

// GetSelectedAvailableContainer returns the selected container from the "Available" tab
func (v *GroupsView) GetSelectedAvailableContainer() *models.Container {
	containers := v.GetAvailableContainers()
	if len(containers) == 0 || v.availableContainersList.Index() >= len(containers) {
		return nil
	}
	return &containers[v.availableContainersList.Index()]
}

// GetContainersInGroup returns containers that are in the selected group
func (v *GroupsView) GetContainersInGroup() []models.Container {
	if v.selectedGroup == nil {
		return []models.Container{}
	}

	// Build set of container IDs in group
	inGroup := make(map[string]bool)
	for _, id := range v.selectedGroup.ContainerIDs {
		inGroup[id] = true
	}

	// Filter
	var result []models.Container
	for _, c := range v.allContainers {
		if inGroup[c.ID] {
			result = append(result, c)
		}
	}
	return result
}

// GetAvailableContainers returns containers NOT in the selected group
func (v *GroupsView) GetAvailableContainers() []models.Container {
	if v.selectedGroup == nil {
		return []models.Container{}
	}

	// Build set of container IDs in group
	inGroup := make(map[string]bool)
	for _, id := range v.selectedGroup.ContainerIDs {
		inGroup[id] = true
	}

	// Filter - return containers NOT in group
	var result []models.Container
	for _, c := range v.allContainers {
		if !inGroup[c.ID] {
			result = append(result, c)
		}
	}
	return result
}

// updateContainerLists updates the container lists based on selected group
func (v *GroupsView) updateContainerLists() {
	// Update containers in group
	inGroupContainers := v.GetContainersInGroup()
	inGroupItems := make([]list.Item, len(inGroupContainers))
	for i, c := range inGroupContainers {
		inGroupItems[i] = ContainerItemForGroup{container: c}
	}
	v.containersInGroupList.SetItems(inGroupItems)

	// Update available containers
	availableContainers := v.GetAvailableContainers()
	availableItems := make([]list.Item, len(availableContainers))
	for i, c := range availableContainers {
		availableItems[i] = ContainerItemForGroup{container: c}
	}
	v.availableContainersList.SetItems(availableItems)
}

// SwitchTab switches to the next or previous tab
func (v *GroupsView) SwitchTab(direction int) {
	newTab := int(v.currentTab) + direction

	// Wrap around
	if newTab < 0 {
		newTab = int(models.GroupsAvailableTab)
	} else if newTab > int(models.GroupsAvailableTab) {
		newTab = int(models.GroupsListTab)
	}

	v.currentTab = models.GroupsTabType(newTab)
}

// IsFiltering returns true if the currently active list is in filtering mode
func (v *GroupsView) IsFiltering() bool {
	switch v.currentTab {
	case models.GroupsListTab:
		return v.groupsList.FilterState() == list.Filtering
	case models.GroupsContainersTab:
		return v.containersInGroupList.FilterState() == list.Filtering
	case models.GroupsAvailableTab:
		return v.availableContainersList.FilterState() == list.Filtering
	default:
		return false
	}
}

// Update handles messages
func (v *GroupsView) Update(msg tea.Msg) (*GroupsView, tea.Cmd) {
	// If filtering, pass to active list
	if v.IsFiltering() {
		var cmd tea.Cmd
		switch v.currentTab {
		case models.GroupsListTab:
			v.groupsList, cmd = v.groupsList.Update(msg)
		case models.GroupsContainersTab:
			v.containersInGroupList, cmd = v.containersInGroupList.Update(msg)
		case models.GroupsAvailableTab:
			v.availableContainersList, cmd = v.availableContainersList.Update(msg)
		}
		return v, cmd
	}

	// Handle key messages
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "a", "left":
			// Switch to previous tab
			v.SwitchTab(-1)
			return v, nil

		case "d", "right":
			// Switch to next tab
			v.SwitchTab(+1)
			return v, nil

		case "enter":
			if v.currentTab == models.GroupsListTab {
				// Select group and switch to "In Group" tab
				v.selectedGroup = v.GetSelectedGroup()
				if v.selectedGroup != nil {
					v.currentTab = models.GroupsContainersTab
					v.updateContainerLists()
				}
				return v, nil
			}
			// enter key in Available tab is handled in app.go for adding container

		case "esc":
			// Return to Groups list tab
			if v.currentTab != models.GroupsListTab {
				v.currentTab = models.GroupsListTab
				return v, nil
			}
		}
	}

	// Delegate to current list
	var cmd tea.Cmd
	switch v.currentTab {
	case models.GroupsListTab:
		v.groupsList, cmd = v.groupsList.Update(msg)
	case models.GroupsContainersTab:
		v.containersInGroupList, cmd = v.containersInGroupList.Update(msg)
	case models.GroupsAvailableTab:
		v.availableContainersList, cmd = v.availableContainersList.Update(msg)
	}

	return v, cmd
}

// RenderTabBar renders the tab bar
func (v *GroupsView) RenderTabBar() string {
	tabs := []string{
		v.renderTab("Groups", models.GroupsListTab),
		v.renderTab("In Group", models.GroupsContainersTab),
		v.renderTab("Available", models.GroupsAvailableTab),
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...) + "\n"
}

// renderTab renders a single tab
func (v *GroupsView) renderTab(label string, tab models.GroupsTabType) string {
	if v.currentTab == tab {
		return styles.TabActiveStyle.Render(" " + label + " ")
	}
	return styles.TabInactiveStyle.Render(" " + label + " ")
}

// View renders the view
func (v *GroupsView) View() string {
	// Tab bar at top
	tabBar := v.RenderTabBar()

	// Content based on current tab
	var content string
	switch v.currentTab {
	case models.GroupsListTab:
		if len(v.groups) == 0 {
			content = v.renderEmpty()
		} else {
			content = v.groupsList.View()
		}

	case models.GroupsContainersTab:
		if v.selectedGroup == nil {
			content = v.renderEmptyState("Select a group from the Groups tab")
		} else if len(v.GetContainersInGroup()) == 0 {
			content = v.renderEmptyState(fmt.Sprintf("No containers in '%s'", v.selectedGroup.Name))
		} else {
			v.containersInGroupList.Title = fmt.Sprintf("Containers in '%s'", v.selectedGroup.Name)
			content = v.containersInGroupList.View()
		}

	case models.GroupsAvailableTab:
		if v.selectedGroup == nil {
			content = v.renderEmptyState("Select a group from the Groups tab")
		} else if len(v.GetAvailableContainers()) == 0 {
			content = v.renderEmptyState(fmt.Sprintf("All containers are in '%s'", v.selectedGroup.Name))
		} else {
			v.availableContainersList.Title = fmt.Sprintf("Available for '%s'", v.selectedGroup.Name)
			content = v.availableContainersList.View()
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, tabBar, content)
}

func (v *GroupsView) renderEmpty() string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("Container Groups"))
	b.WriteString("\n\n")
	b.WriteString(styles.SubtitleStyle.Render("No groups found. Create a group to manage multiple containers together."))
	b.WriteString("\n\n")
	b.WriteString(styles.DescStyle.Render("Press 'n' to create a new group"))

	return b.String()
}

func (v *GroupsView) renderEmptyState(message string) string {
	var b strings.Builder

	b.WriteString("\n\n")
	b.WriteString(styles.SubtitleStyle.Render(message))
	b.WriteString("\n\n")

	return b.String()
}

// GetHelpText returns help text for the groups view based on current tab
func (v *GroupsView) GetHelpText() string {
	var helps []string

	switch v.currentTab {
	case models.GroupsListTab:
		helps = []string{
			styles.KeyStyle.Render("↑/↓") + " navigate",
			styles.KeyStyle.Render("enter") + " select",
			styles.KeyStyle.Render("n") + " new",
			styles.KeyStyle.Render("s") + " start all",
			styles.KeyStyle.Render("x") + " stop all",
			styles.KeyStyle.Render("d") + " delete",
			styles.KeyStyle.Render("a/d") + " tabs",
			styles.KeyStyle.Render("/") + " filter",
		}

	case models.GroupsContainersTab:
		helps = []string{
			styles.KeyStyle.Render("↑/↓") + " navigate",
			styles.KeyStyle.Render("s") + " start",
			styles.KeyStyle.Render("x") + " stop",
			styles.KeyStyle.Render("r") + " restart",
			styles.KeyStyle.Render("d") + " delete",
			styles.KeyStyle.Render("e") + " shell",
			styles.KeyStyle.Render("l") + " logs",
			styles.KeyStyle.Render("t") + " stats",
			styles.KeyStyle.Render("v") + " env",
			styles.KeyStyle.Render("u") + " unlink",
			styles.KeyStyle.Render("/") + " filter",
		}

	case models.GroupsAvailableTab:
		helps = []string{
			styles.KeyStyle.Render("↑/↓") + " navigate",
			styles.KeyStyle.Render("enter") + " add",
			styles.KeyStyle.Render("a/d") + " tabs",
			styles.KeyStyle.Render("esc") + " back",
			styles.KeyStyle.Render("/") + " filter",
		}
	}

	helps = append(helps, styles.KeyStyle.Render("q") + " quit")
	return strings.Join(helps, styles.SeparatorStyle.String())
}

// GetCurrentTab returns the current tab type
func (v *GroupsView) GetCurrentTab() models.GroupsTabType {
	return v.currentTab
}

// GetSelectedGroupForApp returns the selected group for app operations
func (v *GroupsView) GetSelectedGroupForApp() *models.Group {
	return v.selectedGroup
}
