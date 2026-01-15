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

// NetworkItem implements list.Item for networks
type NetworkItem struct {
	network models.Network
}

func (i NetworkItem) FilterValue() string {
	return i.network.Name
}

func (i NetworkItem) Title() string {
	info := ""
	if i.network.IsSystemNetwork() {
		info = styles.PausedStyle.Render("system")
	} else {
		info = styles.RunningStyle.Render(i.network.Driver)
	}
	return fmt.Sprintf("%s  %s", i.network.Name, info)
}

func (i NetworkItem) Description() string {
	containerCount := i.network.GetContainerCount()
	subnet := ""
	if i.network.IPAM.Subnet != "" {
		subnet = fmt.Sprintf(" | Subnet: %s", i.network.IPAM.Subnet)
	}
	return fmt.Sprintf("%d containers | Scope: %s%s", containerCount, i.network.Scope, subnet)
}

// ContainerItemForNetwork implements list.Item for containers in networks view
type ContainerItemForNetwork struct {
	container models.Container
}

func (i ContainerItemForNetwork) FilterValue() string {
	return i.container.Name
}

func (i ContainerItemForNetwork) Title() string {
	status := styles.GetStatusStyle(i.container.State).Render(i.container.State)
	return fmt.Sprintf("%s  %s", i.container.Name, status)
}

func (i ContainerItemForNetwork) Description() string {
	return fmt.Sprintf("ID: %s | Image: %s", i.container.ShortID, i.container.Image)
}

// NetworksView displays the tabbed networks management interface
type NetworksView struct {
	// Tab state
	currentTab models.NetworksTabType

	// Data
	networks        []models.Network
	allContainers   []models.Container
	selectedNetwork *models.Network

	// List models for each tab
	networksList            list.Model
	containersInNetworkList list.Model
	availableContainersList list.Model

	// Dimensions
	width  int
	height int
}

// NewNetworksView creates a new networks view
func NewNetworksView() *NetworksView {
	// Initialize networks list
	networksDelegate := list.NewDefaultDelegate()
	networksDelegate.SetHeight(2)
	networksDelegate.SetSpacing(1)

	networksList := list.New([]list.Item{}, networksDelegate, 0, 0)
	networksList.Title = "Docker Networks"
	networksList.SetShowStatusBar(true)
	networksList.SetFilteringEnabled(true)
	networksList.Styles.Title = styles.TitleStyle

	// Initialize containers in network list
	containersDelegate := list.NewDefaultDelegate()
	containersDelegate.SetHeight(2)
	containersDelegate.SetSpacing(1)

	containersInNetworkList := list.New([]list.Item{}, containersDelegate, 0, 0)
	containersInNetworkList.Title = "Containers in Network"
	containersInNetworkList.SetShowStatusBar(true)
	containersInNetworkList.SetFilteringEnabled(true)
	containersInNetworkList.Styles.Title = styles.TitleStyle

	// Initialize available containers list
	availableDelegate := list.NewDefaultDelegate()
	availableDelegate.SetHeight(2)
	availableDelegate.SetSpacing(1)

	availableContainersList := list.New([]list.Item{}, availableDelegate, 0, 0)
	availableContainersList.Title = "Available Containers"
	availableContainersList.SetShowStatusBar(true)
	availableContainersList.SetFilteringEnabled(true)
	availableContainersList.Styles.Title = styles.TitleStyle

	return &NetworksView{
		currentTab:              models.NetworksListTab,
		networksList:            networksList,
		containersInNetworkList: containersInNetworkList,
		availableContainersList: availableContainersList,
	}
}

// SetNetworks updates the list of networks
func (v *NetworksView) SetNetworks(networks []models.Network) {
	v.networks = networks

	// Sync container counts from container data (if available)
	v.syncNetworkContainerCounts()

	// Build list items (syncNetworkContainerCounts already does this if containers exist)
	if len(v.allContainers) == 0 {
		items := make([]list.Item, len(networks))
		for i, n := range networks {
			items[i] = NetworkItem{network: n}
		}
		v.networksList.SetItems(items)
	}

	// Update the selected network if it still exists
	if v.selectedNetwork != nil {
		found := false
		for i, n := range v.networks {
			if n.ID == v.selectedNetwork.ID {
				v.selectedNetwork = &v.networks[i]
				found = true
				break
			}
		}
		if found {
			v.updateContainerLists()
		} else {
			v.selectedNetwork = nil
		}
	}
}

// SetAllContainers updates the list of all containers
func (v *NetworksView) SetAllContainers(containers []models.Container) {
	v.allContainers = containers

	// Update network container counts based on container data
	// (NetworkList API doesn't populate Containers field)
	v.syncNetworkContainerCounts()

	v.updateContainerLists()
}

// syncNetworkContainerCounts populates each network's Containers field from container data
func (v *NetworksView) syncNetworkContainerCounts() {
	if len(v.networks) == 0 || len(v.allContainers) == 0 {
		return
	}

	// Build map of network name -> container IDs
	networkContainers := make(map[string][]string)
	for _, c := range v.allContainers {
		for _, netName := range c.Networks {
			networkContainers[netName] = append(networkContainers[netName], c.ID)
		}
	}

	// Update each network's Containers field
	for i := range v.networks {
		v.networks[i].Containers = networkContainers[v.networks[i].Name]
	}

	// Rebuild the list items with updated counts
	items := make([]list.Item, len(v.networks))
	for i, n := range v.networks {
		items[i] = NetworkItem{network: n}
	}
	v.networksList.SetItems(items)

	// Update selected network reference if it exists
	if v.selectedNetwork != nil {
		for i, n := range v.networks {
			if n.ID == v.selectedNetwork.ID {
				v.selectedNetwork = &v.networks[i]
				break
			}
		}
	}
}

// SetSize updates the view dimensions
func (v *NetworksView) SetSize(width, height int) {
	v.width = width
	v.height = height

	// Account for tab bar (3 lines) and reduce height accordingly
	listHeight := height - 9
	v.networksList.SetSize(width, listHeight)
	v.containersInNetworkList.SetSize(width, listHeight)
	v.availableContainersList.SetSize(width, listHeight)
}

// GetSelectedNetwork returns the currently selected network
func (v *NetworksView) GetSelectedNetwork() *models.Network {
	if len(v.networks) == 0 || v.networksList.Index() >= len(v.networks) {
		return nil
	}
	return &v.networks[v.networksList.Index()]
}

// GetSelectedInNetworkContainer returns the selected container from the "In Network" tab
func (v *NetworksView) GetSelectedInNetworkContainer() *models.Container {
	containers := v.GetContainersInNetwork()
	if len(containers) == 0 || v.containersInNetworkList.Index() >= len(containers) {
		return nil
	}
	return &containers[v.containersInNetworkList.Index()]
}

// GetSelectedAvailableContainer returns the selected container from the "Available" tab
func (v *NetworksView) GetSelectedAvailableContainer() *models.Container {
	containers := v.GetAvailableContainers()
	if len(containers) == 0 || v.availableContainersList.Index() >= len(containers) {
		return nil
	}
	return &containers[v.availableContainersList.Index()]
}

// GetContainersInNetwork returns containers that are in the selected network
func (v *NetworksView) GetContainersInNetwork() []models.Container {
	if v.selectedNetwork == nil {
		return []models.Container{}
	}

	// Filter - return containers that have this network in their Networks list
	var result []models.Container
	for _, c := range v.allContainers {
		for _, netName := range c.Networks {
			if netName == v.selectedNetwork.Name {
				result = append(result, c)
				break
			}
		}
	}
	return result
}

// GetAvailableContainers returns containers NOT in the selected network
func (v *NetworksView) GetAvailableContainers() []models.Container {
	if v.selectedNetwork == nil {
		return []models.Container{}
	}

	// Filter - return containers NOT in network
	var result []models.Container
	for _, c := range v.allContainers {
		inNetwork := false
		for _, netName := range c.Networks {
			if netName == v.selectedNetwork.Name {
				inNetwork = true
				break
			}
		}
		if !inNetwork {
			result = append(result, c)
		}
	}
	return result
}

// updateContainerLists updates the container lists based on selected network
func (v *NetworksView) updateContainerLists() {
	// Update containers in network
	inNetworkContainers := v.GetContainersInNetwork()
	inNetworkItems := make([]list.Item, len(inNetworkContainers))
	for i, c := range inNetworkContainers {
		inNetworkItems[i] = ContainerItemForNetwork{container: c}
	}
	v.containersInNetworkList.SetItems(inNetworkItems)

	// Update available containers
	availableContainers := v.GetAvailableContainers()
	availableItems := make([]list.Item, len(availableContainers))
	for i, c := range availableContainers {
		availableItems[i] = ContainerItemForNetwork{container: c}
	}
	v.availableContainersList.SetItems(availableItems)
}

// SwitchTab switches to the next or previous tab
func (v *NetworksView) SwitchTab(direction int) {
	newTab := int(v.currentTab) + direction

	// Wrap around
	if newTab < 0 {
		newTab = int(models.NetworksAvailableTab)
	} else if newTab > int(models.NetworksAvailableTab) {
		newTab = int(models.NetworksListTab)
	}

	v.currentTab = models.NetworksTabType(newTab)
}

// IsFiltering returns true if the currently active list is in filtering mode
func (v *NetworksView) IsFiltering() bool {
	switch v.currentTab {
	case models.NetworksListTab:
		return v.networksList.FilterState() == list.Filtering
	case models.NetworksContainersTab:
		return v.containersInNetworkList.FilterState() == list.Filtering
	case models.NetworksAvailableTab:
		return v.availableContainersList.FilterState() == list.Filtering
	default:
		return false
	}
}

// Update handles messages
func (v *NetworksView) Update(msg tea.Msg) (*NetworksView, tea.Cmd) {
	// If filtering, pass to active list
	if v.IsFiltering() {
		var cmd tea.Cmd
		switch v.currentTab {
		case models.NetworksListTab:
			v.networksList, cmd = v.networksList.Update(msg)
		case models.NetworksContainersTab:
			v.containersInNetworkList, cmd = v.containersInNetworkList.Update(msg)
		case models.NetworksAvailableTab:
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
			if v.currentTab == models.NetworksListTab {
				// Select network and switch to "In Network" tab
				v.selectedNetwork = v.GetSelectedNetwork()
				if v.selectedNetwork != nil {
					v.currentTab = models.NetworksContainersTab
					v.updateContainerLists()
				}
				return v, nil
			}
			// enter key in Available tab is handled in app.go for attaching container

		case "esc":
			// Return to Networks list tab
			if v.currentTab != models.NetworksListTab {
				v.currentTab = models.NetworksListTab
				return v, nil
			}
		}
	}

	// Delegate to current list
	var cmd tea.Cmd
	switch v.currentTab {
	case models.NetworksListTab:
		v.networksList, cmd = v.networksList.Update(msg)
	case models.NetworksContainersTab:
		v.containersInNetworkList, cmd = v.containersInNetworkList.Update(msg)
	case models.NetworksAvailableTab:
		v.availableContainersList, cmd = v.availableContainersList.Update(msg)
	}

	return v, cmd
}

// RenderTabBar renders the tab bar
func (v *NetworksView) RenderTabBar() string {
	tabs := []string{
		v.renderTab("Networks", models.NetworksListTab),
		v.renderTab("In Network", models.NetworksContainersTab),
		v.renderTab("Available", models.NetworksAvailableTab),
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...) + "\n"
}

// renderTab renders a single tab
func (v *NetworksView) renderTab(label string, tab models.NetworksTabType) string {
	if v.currentTab == tab {
		return styles.TabActiveStyle.Render(" " + label + " ")
	}
	return styles.TabInactiveStyle.Render(" " + label + " ")
}

// View renders the view
func (v *NetworksView) View() string {
	// Tab bar at top
	tabBar := v.RenderTabBar()

	// Content based on current tab
	var content string
	switch v.currentTab {
	case models.NetworksListTab:
		if len(v.networks) == 0 {
			content = v.renderEmpty()
		} else {
			content = v.networksList.View()
		}

	case models.NetworksContainersTab:
		if v.selectedNetwork == nil {
			content = v.renderEmptyState("Select a network from the Networks tab")
		} else if len(v.GetContainersInNetwork()) == 0 {
			content = v.renderEmptyState(fmt.Sprintf("No containers in '%s'", v.selectedNetwork.Name))
		} else {
			v.containersInNetworkList.Title = fmt.Sprintf("Containers in '%s'", v.selectedNetwork.Name)
			content = v.containersInNetworkList.View()
		}

	case models.NetworksAvailableTab:
		if v.selectedNetwork == nil {
			content = v.renderEmptyState("Select a network from the Networks tab")
		} else if v.selectedNetwork.IsSystemNetwork() {
			content = v.renderEmptyState(fmt.Sprintf("Cannot attach containers to system network '%s'", v.selectedNetwork.Name))
		} else if len(v.GetAvailableContainers()) == 0 {
			content = v.renderEmptyState(fmt.Sprintf("All containers are in '%s'", v.selectedNetwork.Name))
		} else {
			v.availableContainersList.Title = fmt.Sprintf("Available for '%s'", v.selectedNetwork.Name)
			content = v.availableContainersList.View()
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, tabBar, content)
}

func (v *NetworksView) renderEmpty() string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("Docker Networks"))
	b.WriteString("\n\n")
	b.WriteString(styles.SubtitleStyle.Render("No networks found."))

	return b.String()
}

func (v *NetworksView) renderEmptyState(message string) string {
	var b strings.Builder

	b.WriteString("\n\n")
	b.WriteString(styles.SubtitleStyle.Render(message))
	b.WriteString("\n\n")

	return b.String()
}

// GetHelpText returns help text for the networks view based on current tab
func (v *NetworksView) GetHelpText() string {
	var helps []string

	switch v.currentTab {
	case models.NetworksListTab:
		helps = []string{
			styles.KeyStyle.Render("↑/↓") + " navigate",
			styles.KeyStyle.Render("enter") + " select",
			styles.KeyStyle.Render("n") + " new",
			styles.KeyStyle.Render("d") + " delete",
			styles.KeyStyle.Render("a/d") + " tabs",
			styles.KeyStyle.Render("/") + " filter",
		}

	case models.NetworksContainersTab:
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
			styles.KeyStyle.Render("u") + " disconnect",
			styles.KeyStyle.Render("/") + " filter",
		}

	case models.NetworksAvailableTab:
		helps = []string{
			styles.KeyStyle.Render("↑/↓") + " navigate",
			styles.KeyStyle.Render("enter") + " connect",
			styles.KeyStyle.Render("a/d") + " tabs",
			styles.KeyStyle.Render("esc") + " back",
			styles.KeyStyle.Render("/") + " filter",
		}
	}

	helps = append(helps, styles.KeyStyle.Render("q")+" quit")
	return strings.Join(helps, styles.SeparatorStyle.String())
}

// GetCurrentTab returns the current tab type
func (v *NetworksView) GetCurrentTab() models.NetworksTabType {
	return v.currentTab
}

// GetSelectedNetworkForApp returns the selected network for app operations
func (v *NetworksView) GetSelectedNetworkForApp() *models.Network {
	return v.selectedNetwork
}
