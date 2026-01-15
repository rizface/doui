package app

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rizface/doui/internal/config"
	"github.com/rizface/doui/internal/docker"
	"github.com/rizface/doui/internal/models"
	"github.com/rizface/doui/internal/ui/components"
	"github.com/rizface/doui/internal/ui/styles"
	"github.com/rizface/doui/internal/ui/views"
)

// App is the main application model
type App struct {
	// State
	state  *models.AppState
	width  int
	height int
	ready  bool

	// Services
	docker        *docker.Client
	groupManager  *config.GroupManager

	// UI Components
	sidebar *components.Sidebar
	header  *components.Header
	footer  *components.Footer
	modal   *components.Modal

	// Views
	containersView *views.ContainersView
	imagesView     *views.ImagesView
	groupsView     *views.GroupsView
	volumesView    *views.VolumesView
	composeView    *views.ComposeView
	networksView   *views.NetworksView
	logsView       *views.LogsView
	statsView      *views.StatsView
	envVarsView    *views.EnvVarsView
	aboutView      *views.AboutView

	// Status
	statusMessage string
	errorMessage  string

	// Pending operations
	pendingDelete     string // ID of item pending deletion
	pendingDeleteType string // "container", "image", "group"

	// Env var editing state
	pendingEnvContainer *models.ContainerFullConfig
}

// New creates a new application
func New() *App {
	return &App{
		state:   models.NewAppState(),
		sidebar: components.NewSidebar(),
		header:  components.NewHeader(),
		footer:  components.NewFooter(),

		containersView: views.NewContainersView(),
		imagesView:     views.NewImagesView(),
		groupsView:     views.NewGroupsView(),
		volumesView:    views.NewVolumesView(),
		composeView:    views.NewComposeView(),
		networksView:   views.NewNetworksView(),
		logsView:       views.NewLogsView(),
		statsView:      views.NewStatsView(),
		envVarsView:    views.NewEnvVarsView(),
		aboutView:      views.NewAboutView(),
	}
}

// Init initializes the application
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		initDockerClient(),
		initGroupManager(),
		tickRefresh(),
	)
}

// Update handles messages
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

		// Calculate layout dimensions
		sidebarWidth := 22
		mainWidth := msg.Width - sidebarWidth

		// Update component sizes
		a.sidebar.SetSize(sidebarWidth, msg.Height)
		a.header.SetSize(mainWidth)
		a.footer.SetSize(msg.Width)

		if a.modal != nil {
			a.modal.SetSize(msg.Width, msg.Height)
		}

		// Update view sizes (main area)
		a.containersView.SetSize(mainWidth, msg.Height-4) // Reserve for header+footer
		a.imagesView.SetSize(mainWidth, msg.Height-4)
		a.groupsView.SetSize(mainWidth, msg.Height-4)
		a.volumesView.SetSize(mainWidth, msg.Height-4)
		a.composeView.SetSize(mainWidth, msg.Height-4)
		a.networksView.SetSize(mainWidth, msg.Height-4)
		a.logsView.SetSize(mainWidth, msg.Height-4)
		a.statsView.SetSize(mainWidth, msg.Height-4)
		a.envVarsView.SetSize(mainWidth, msg.Height-4)
		a.aboutView.SetSize(msg.Width, msg.Height-4) // Full width for about page

	case tea.KeyMsg:
		// Handle modal first if visible
		if a.modal != nil && a.modal.IsVisible() {
			var cmd tea.Cmd
			a.modal, cmd = a.modal.Update(msg)

			// Check if modal was confirmed
			if !a.modal.IsVisible() {
				if a.modal.IsConfirmed() {
					return a.handleModalConfirmed()
				}
				// Modal cancelled
				a.modal = nil
				a.pendingDelete = ""
				a.pendingDeleteType = ""
			}

			return a, cmd
		}

		// If any view is currently filtering, skip command handling and let the view handle all input
		if (a.state.CurrentView == models.ViewContainers && a.containersView.IsFiltering()) ||
		   (a.state.CurrentView == models.ViewImages && a.imagesView.IsFiltering()) ||
		   (a.state.CurrentView == models.ViewGroups && a.groupsView.IsFiltering()) ||
		   (a.state.CurrentView == models.ViewVolumes && a.volumesView.IsFiltering()) ||
		   (a.state.CurrentView == models.ViewCompose && a.composeView.IsFiltering()) ||
		   (a.state.CurrentView == models.ViewNetworks && a.networksView.IsFiltering()) {
			// Delegate directly to the view to handle filter input
			var cmd tea.Cmd
			switch a.state.CurrentView {
			case models.ViewContainers:
				a.containersView, cmd = a.containersView.Update(msg)
			case models.ViewImages:
				a.imagesView, cmd = a.imagesView.Update(msg)
			case models.ViewGroups:
				a.groupsView, cmd = a.groupsView.Update(msg)
			case models.ViewVolumes:
				a.volumesView, cmd = a.volumesView.Update(msg)
			case models.ViewCompose:
				a.composeView, cmd = a.composeView.Update(msg)
			case models.ViewNetworks:
				a.networksView, cmd = a.networksView.Update(msg)
			}
			return a, cmd
		}

		// Global keybindings
		switch msg.String() {
		case "ctrl+c", "q":
			// Don't quit if in logs/stats/shell/about views, return to containers instead
			if a.state.CurrentView == models.ViewLogs || a.state.CurrentView == models.ViewStats || a.state.CurrentView == models.ViewAbout {
				a.state.CurrentView = models.ViewContainers
				a.sidebar.SetCurrentView(models.ViewContainers)
				return a, nil
			}

			if a.docker != nil {
				a.docker.Close()
			}
			return a, tea.Quit

		case "?":
			// Open About page
			a.state.PreviousView = a.state.CurrentView
			a.state.CurrentView = models.ViewAbout
			a.sidebar.SetCurrentView(models.ViewAbout)
			return a, nil

		case "esc":
			// Handle About view - go back
			if a.state.CurrentView == models.ViewAbout {
				a.state.CurrentView = a.state.PreviousView
				a.sidebar.SetCurrentView(a.state.PreviousView)
				return a, nil
			}

			// Handle env vars view - back without saving
			if a.state.CurrentView == models.ViewEnvVars {
				if a.envVarsView.IsEditing() {
					// Let editor handle esc first (cancel add/edit mode)
					var cmd tea.Cmd
					a.envVarsView, cmd = a.envVarsView.Update(msg)
					return a, cmd
				}
				// Return to previous view without saving
				a.pendingEnvContainer = nil
				a.state.CurrentView = a.state.PreviousView
				a.sidebar.SetCurrentView(a.state.PreviousView)
				return a, nil
			}

			// Let compose view handle esc if viewing services or containers
			if a.state.CurrentView == models.ViewCompose && (a.composeView.IsViewingServices() || a.composeView.IsViewingContainers()) {
				// Delegate to compose view to handle internal navigation
				var cmd tea.Cmd
				a.composeView, cmd = a.composeView.Update(msg)
				return a, cmd
			}

			// Return to previous view or containers
			if a.state.CurrentView != models.ViewContainers {
				a.state.PreviousView = a.state.CurrentView
				a.state.CurrentView = models.ViewContainers
				a.sidebar.SetCurrentView(models.ViewContainers)
				return a, nil
			}

		case "1":
			a.state.PreviousView = a.state.CurrentView
			a.state.CurrentView = models.ViewContainers
			a.sidebar.SetCurrentView(models.ViewContainers)
			return a, nil

		case "2":
			a.state.PreviousView = a.state.CurrentView
			a.state.CurrentView = models.ViewImages
			a.sidebar.SetCurrentView(models.ViewImages)
			return a, tea.Batch(fetchImages(a.docker))

		case "3":
			a.state.PreviousView = a.state.CurrentView
			a.state.CurrentView = models.ViewGroups
			a.sidebar.SetCurrentView(models.ViewGroups)
			return a, loadGroups(a.groupManager)

		case "4":
			a.state.PreviousView = a.state.CurrentView
			a.state.CurrentView = models.ViewVolumes
			a.sidebar.SetCurrentView(models.ViewVolumes)
			return a, tea.Batch(fetchVolumes(a.docker), fetchContainers(a.docker))

		case "5":
			a.state.PreviousView = a.state.CurrentView
			a.state.CurrentView = models.ViewCompose
			a.sidebar.SetCurrentView(models.ViewCompose)
			return a, fetchComposeProjects(a.docker)

		case "6":
			a.state.PreviousView = a.state.CurrentView
			a.state.CurrentView = models.ViewNetworks
			a.sidebar.SetCurrentView(models.ViewNetworks)
			return a, tea.Batch(fetchNetworks(a.docker), fetchContainers(a.docker))

		case "7":
			a.state.PreviousView = a.state.CurrentView
			a.state.CurrentView = models.ViewAbout
			a.sidebar.SetCurrentView(models.ViewAbout)
			return a, nil

		case "tab", "right":
			// Cycle forward through tabs (only in main views, not logs/stats)
			if a.state.CurrentView == models.ViewContainers ||
			   a.state.CurrentView == models.ViewImages ||
			   a.state.CurrentView == models.ViewGroups ||
			   a.state.CurrentView == models.ViewVolumes ||
			   a.state.CurrentView == models.ViewCompose ||
			   a.state.CurrentView == models.ViewNetworks ||
			   a.state.CurrentView == models.ViewAbout {
				return a.cycleTabForward()
			}

		case "shift+tab", "left":
			// Cycle backward through tabs (only in main views, not logs/stats)
			if a.state.CurrentView == models.ViewContainers ||
			   a.state.CurrentView == models.ViewImages ||
			   a.state.CurrentView == models.ViewGroups ||
			   a.state.CurrentView == models.ViewVolumes ||
			   a.state.CurrentView == models.ViewCompose ||
			   a.state.CurrentView == models.ViewNetworks ||
			   a.state.CurrentView == models.ViewAbout {
				return a.cycleTabBackward()
			}

		case "n":
			// Create new group (only in groups view, list tab)
			if a.state.CurrentView == models.ViewGroups && a.groupsView.GetCurrentTab() == models.GroupsListTab {
				a.modal = components.NewFormModal("Create New Group", []string{"Name", "Description"})
				a.modal.SetSize(a.width, a.height)
				a.pendingDeleteType = "create_group"
				return a, nil
			}
			// Create new network (only in networks view, list tab)
			if a.state.CurrentView == models.ViewNetworks && a.networksView.GetCurrentTab() == models.NetworksListTab {
				a.modal = components.NewFormModal("Create New Network", []string{"Name", "Driver (default: bridge)"})
				a.modal.SetSize(a.width, a.height)
				a.pendingDeleteType = "create_network"
				return a, nil
			}

		case "enter":
			// In Groups view, Available tab: Add container to group
			if a.state.CurrentView == models.ViewGroups && a.groupsView.GetCurrentTab() == models.GroupsAvailableTab {
				if container := a.groupsView.GetSelectedAvailableContainer(); container != nil {
					if selectedGroup := a.groupsView.GetSelectedGroupForApp(); selectedGroup != nil {
						return a, addContainerToGroup(a.groupManager, selectedGroup.ID, container.ID)
					}
				}
				return a, nil
			}
			// In Networks view, Available tab: Connect container to network
			if a.state.CurrentView == models.ViewNetworks && a.networksView.GetCurrentTab() == models.NetworksAvailableTab {
				if container := a.networksView.GetSelectedAvailableContainer(); container != nil {
					if selectedNetwork := a.networksView.GetSelectedNetworkForApp(); selectedNetwork != nil {
						if !selectedNetwork.IsSystemNetwork() {
							return a, connectContainerToNetwork(a.docker, selectedNetwork.ID, container.ID)
						}
					}
				}
				return a, nil
			}

		case "u":
			// In Groups view, In Group tab: Unlink/remove container from group
			if a.state.CurrentView == models.ViewGroups && a.groupsView.GetCurrentTab() == models.GroupsContainersTab {
				if container := a.groupsView.GetSelectedInGroupContainer(); container != nil {
					if selectedGroup := a.groupsView.GetSelectedGroupForApp(); selectedGroup != nil {
						a.modal = components.NewConfirmModal(
							"Remove from Group",
							fmt.Sprintf("Remove '%s' from group '%s'?", container.Name, selectedGroup.Name),
						)
						a.modal.SetSize(a.width, a.height)
						a.pendingDelete = container.ID
						a.pendingDeleteType = "remove_from_group"
						return a, nil
					}
				}
				return a, nil
			}
			// In Networks view, In Network tab: Disconnect container from network
			if a.state.CurrentView == models.ViewNetworks && a.networksView.GetCurrentTab() == models.NetworksContainersTab {
				if container := a.networksView.GetSelectedInNetworkContainer(); container != nil {
					if selectedNetwork := a.networksView.GetSelectedNetworkForApp(); selectedNetwork != nil {
						a.modal = components.NewConfirmModal(
							"Disconnect from Network",
							fmt.Sprintf("Disconnect '%s' from network '%s'?", container.Name, selectedNetwork.Name),
						)
						a.modal.SetSize(a.width, a.height)
						a.pendingDelete = container.ID
						a.pendingDeleteType = "disconnect_from_network"
						return a, nil
					}
				}
				return a, nil
			}

		case "r":
			// Restart container (in containers view, group tab, compose services/containers, or networks containers tab)
			if a.state.CurrentView == models.ViewContainers {
				if container := a.containersView.GetSelectedContainer(); container != nil {
					return a, restartContainer(a.docker, container.ID)
				}
			} else if a.state.CurrentView == models.ViewGroups && a.groupsView.GetCurrentTab() == models.GroupsContainersTab {
				if container := a.groupsView.GetSelectedInGroupContainer(); container != nil {
					return a, restartContainer(a.docker, container.ID)
				}
			} else if a.state.CurrentView == models.ViewCompose {
				if a.composeView.IsViewingServices() || a.composeView.IsViewingContainers() {
					// Restart individual container
					if container := a.composeView.GetSelectedContainer(); container != nil {
						return a, restartContainer(a.docker, container.ID)
					}
				} else {
					// Restart all containers in compose project
					if project := a.composeView.GetSelectedProject(); project != nil {
						return a, restartComposeProject(a.docker, project.Name)
					}
				}
			} else if a.state.CurrentView == models.ViewNetworks && a.networksView.GetCurrentTab() == models.NetworksContainersTab {
				if container := a.networksView.GetSelectedInNetworkContainer(); container != nil {
					return a, restartContainer(a.docker, container.ID)
				}
			}

		// Container operations (containers view, group tab, and compose services/containers)
		case "s":
			if a.state.CurrentView == models.ViewContainers {
				if container := a.containersView.GetSelectedContainer(); container != nil {
					return a, startContainer(a.docker, container.ID)
				}
			} else if a.state.CurrentView == models.ViewGroups && a.groupsView.GetCurrentTab() == models.GroupsListTab {
				// Start all containers in group
				if group := a.groupsView.GetSelectedGroup(); group != nil {
					return a, startGroup(a.docker, a.groupManager, group.ID)
				}
			} else if a.state.CurrentView == models.ViewGroups && a.groupsView.GetCurrentTab() == models.GroupsContainersTab {
				// Start individual container in group
				if container := a.groupsView.GetSelectedInGroupContainer(); container != nil {
					return a, startContainer(a.docker, container.ID)
				}
			} else if a.state.CurrentView == models.ViewCompose {
				if a.composeView.IsViewingServices() || a.composeView.IsViewingContainers() {
					// Start individual container
					if container := a.composeView.GetSelectedContainer(); container != nil {
						return a, startContainer(a.docker, container.ID)
					}
				} else {
					// Start all containers in compose project
					if project := a.composeView.GetSelectedProject(); project != nil {
						return a, startComposeProject(a.docker, project.Name)
					}
				}
			} else if a.state.CurrentView == models.ViewNetworks && a.networksView.GetCurrentTab() == models.NetworksContainersTab {
				if container := a.networksView.GetSelectedInNetworkContainer(); container != nil {
					return a, startContainer(a.docker, container.ID)
				}
			}

		case "x":
			if a.state.CurrentView == models.ViewContainers {
				if container := a.containersView.GetSelectedContainer(); container != nil {
					return a, stopContainer(a.docker, container.ID)
				}
			} else if a.state.CurrentView == models.ViewGroups && a.groupsView.GetCurrentTab() == models.GroupsListTab {
				// Stop all containers in group
				if group := a.groupsView.GetSelectedGroup(); group != nil {
					return a, stopGroup(a.docker, a.groupManager, group.ID)
				}
			} else if a.state.CurrentView == models.ViewGroups && a.groupsView.GetCurrentTab() == models.GroupsContainersTab {
				// Stop individual container in group
				if container := a.groupsView.GetSelectedInGroupContainer(); container != nil {
					return a, stopContainer(a.docker, container.ID)
				}
			} else if a.state.CurrentView == models.ViewCompose {
				if a.composeView.IsViewingServices() || a.composeView.IsViewingContainers() {
					// Stop individual container
					if container := a.composeView.GetSelectedContainer(); container != nil {
						return a, stopContainer(a.docker, container.ID)
					}
				} else {
					// Stop all containers in compose project
					if project := a.composeView.GetSelectedProject(); project != nil {
						return a, stopComposeProject(a.docker, project.Name)
					}
				}
			} else if a.state.CurrentView == models.ViewNetworks && a.networksView.GetCurrentTab() == models.NetworksContainersTab {
				if container := a.networksView.GetSelectedInNetworkContainer(); container != nil {
					return a, stopContainer(a.docker, container.ID)
				}
			}

		case "l":
			// View logs (containers view, group tab, or compose services/containers)
			if a.state.CurrentView == models.ViewContainers {
				if container := a.containersView.GetSelectedContainer(); container != nil {
					a.state.PreviousView = a.state.CurrentView
					a.state.CurrentView = models.ViewLogs
					a.state.SelectedContainer = container
					return a, startLogStreaming(a.docker, a.logsView, container)
				}
			} else if a.state.CurrentView == models.ViewGroups && a.groupsView.GetCurrentTab() == models.GroupsContainersTab {
				if container := a.groupsView.GetSelectedInGroupContainer(); container != nil {
					a.state.PreviousView = a.state.CurrentView
					a.state.CurrentView = models.ViewLogs
					a.state.SelectedContainer = container
					return a, startLogStreaming(a.docker, a.logsView, container)
				}
			} else if a.state.CurrentView == models.ViewCompose && (a.composeView.IsViewingServices() || a.composeView.IsViewingContainers()) {
				if container := a.composeView.GetSelectedContainer(); container != nil {
					a.state.PreviousView = a.state.CurrentView
					a.state.CurrentView = models.ViewLogs
					a.state.SelectedContainer = container
					return a, startLogStreaming(a.docker, a.logsView, container)
				}
			} else if a.state.CurrentView == models.ViewNetworks && a.networksView.GetCurrentTab() == models.NetworksContainersTab {
				if container := a.networksView.GetSelectedInNetworkContainer(); container != nil {
					a.state.PreviousView = a.state.CurrentView
					a.state.CurrentView = models.ViewLogs
					a.state.SelectedContainer = container
					return a, startLogStreaming(a.docker, a.logsView, container)
				}
			}

		case "t":
			// View stats (containers view, group tab, or compose services/containers)
			if a.state.CurrentView == models.ViewContainers {
				if container := a.containersView.GetSelectedContainer(); container != nil {
					a.state.PreviousView = a.state.CurrentView
					a.state.CurrentView = models.ViewStats
					a.state.SelectedContainer = container
					return a, startStatsStreaming(a.docker, a.statsView, container)
				}
			} else if a.state.CurrentView == models.ViewGroups && a.groupsView.GetCurrentTab() == models.GroupsContainersTab {
				if container := a.groupsView.GetSelectedInGroupContainer(); container != nil {
					a.state.PreviousView = a.state.CurrentView
					a.state.CurrentView = models.ViewStats
					a.state.SelectedContainer = container
					return a, startStatsStreaming(a.docker, a.statsView, container)
				}
			} else if a.state.CurrentView == models.ViewCompose && (a.composeView.IsViewingServices() || a.composeView.IsViewingContainers()) {
				if container := a.composeView.GetSelectedContainer(); container != nil {
					a.state.PreviousView = a.state.CurrentView
					a.state.CurrentView = models.ViewStats
					a.state.SelectedContainer = container
					return a, startStatsStreaming(a.docker, a.statsView, container)
				}
			} else if a.state.CurrentView == models.ViewNetworks && a.networksView.GetCurrentTab() == models.NetworksContainersTab {
				if container := a.networksView.GetSelectedInNetworkContainer(); container != nil {
					a.state.PreviousView = a.state.CurrentView
					a.state.CurrentView = models.ViewStats
					a.state.SelectedContainer = container
					return a, startStatsStreaming(a.docker, a.statsView, container)
				}
			}

		case "e":
			// Enter shell (containers view, group tab, or compose services/containers)
			if a.state.CurrentView == models.ViewContainers {
				if container := a.containersView.GetSelectedContainer(); container != nil {
					return a, execShell(container.ID, container.Name)
				}
			} else if a.state.CurrentView == models.ViewGroups && a.groupsView.GetCurrentTab() == models.GroupsContainersTab {
				if container := a.groupsView.GetSelectedInGroupContainer(); container != nil {
					return a, execShell(container.ID, container.Name)
				}
			} else if a.state.CurrentView == models.ViewCompose && (a.composeView.IsViewingServices() || a.composeView.IsViewingContainers()) {
				if container := a.composeView.GetSelectedContainer(); container != nil {
					return a, execShell(container.ID, container.Name)
				}
			} else if a.state.CurrentView == models.ViewNetworks && a.networksView.GetCurrentTab() == models.NetworksContainersTab {
				if container := a.networksView.GetSelectedInNetworkContainer(); container != nil {
					return a, execShell(container.ID, container.Name)
				}
			}

		case "v":
			// View/Edit environment variables (containers view, group tab, compose, or networks)
			if a.state.CurrentView == models.ViewContainers {
				if container := a.containersView.GetSelectedContainer(); container != nil {
					return a, loadContainerConfig(a.docker, container.ID)
				}
			} else if a.state.CurrentView == models.ViewGroups && a.groupsView.GetCurrentTab() == models.GroupsContainersTab {
				if container := a.groupsView.GetSelectedInGroupContainer(); container != nil {
					return a, loadContainerConfig(a.docker, container.ID)
				}
			} else if a.state.CurrentView == models.ViewCompose && (a.composeView.IsViewingServices() || a.composeView.IsViewingContainers()) {
				if container := a.composeView.GetSelectedContainer(); container != nil {
					return a, loadContainerConfig(a.docker, container.ID)
				}
			} else if a.state.CurrentView == models.ViewNetworks && a.networksView.GetCurrentTab() == models.NetworksContainersTab {
				if container := a.networksView.GetSelectedInNetworkContainer(); container != nil {
					return a, loadContainerConfig(a.docker, container.ID)
				}
			}

		case "ctrl+s":
			// Save env vars and rebuild container
			if a.state.CurrentView == models.ViewEnvVars && a.envVarsView.IsModified() {
				if a.pendingEnvContainer != nil {
					// Update env vars in pending config
					a.pendingEnvContainer.Env = a.envVarsView.GetEnvVars()
					return a, recreateContainer(a.docker, a.state.SelectedContainer.ID, a.pendingEnvContainer)
				}
			}

		case "d":
			// Delete with confirmation
			if a.state.CurrentView == models.ViewContainers {
				if container := a.containersView.GetSelectedContainer(); container != nil {
					a.modal = components.NewConfirmModal(
						"Delete Container",
						fmt.Sprintf("Are you sure you want to remove container '%s'?", container.Name),
					)
					a.modal.SetSize(a.width, a.height)
					a.pendingDelete = container.ID
					a.pendingDeleteType = "container"
					return a, nil
				}
			} else if a.state.CurrentView == models.ViewImages {
				if image := a.imagesView.GetSelectedImage(); image != nil {
					a.modal = components.NewConfirmModal(
						"Delete Image",
						fmt.Sprintf("Are you sure you want to remove image '%s'?", image.GetPrimaryTag()),
					)
					a.modal.SetSize(a.width, a.height)
					a.pendingDelete = image.ID
					a.pendingDeleteType = "image"
					return a, nil
				}
			} else if a.state.CurrentView == models.ViewGroups && a.groupsView.GetCurrentTab() == models.GroupsListTab {
				if group := a.groupsView.GetSelectedGroup(); group != nil {
					a.modal = components.NewConfirmModal(
						"Delete Group",
						fmt.Sprintf("Are you sure you want to delete group '%s'?", group.Name),
					)
					a.modal.SetSize(a.width, a.height)
					a.pendingDelete = group.ID
					a.pendingDeleteType = "group"
					return a, nil
				}
			} else if a.state.CurrentView == models.ViewGroups && a.groupsView.GetCurrentTab() == models.GroupsContainersTab {
				if container := a.groupsView.GetSelectedInGroupContainer(); container != nil {
					a.modal = components.NewConfirmModal(
						"Delete Container",
						fmt.Sprintf("Are you sure you want to remove container '%s'?", container.Name),
					)
					a.modal.SetSize(a.width, a.height)
					a.pendingDelete = container.ID
					a.pendingDeleteType = "container"
					return a, nil
				}
			} else if a.state.CurrentView == models.ViewVolumes {
				if volume := a.volumesView.GetSelectedVolume(); volume != nil {
					a.modal = components.NewConfirmModal(
						"Delete Volume",
						fmt.Sprintf("Are you sure you want to remove volume '%s'?", volume.Name),
					)
					a.modal.SetSize(a.width, a.height)
					a.pendingDelete = volume.Name
					a.pendingDeleteType = "volume"
					return a, nil
				}
			} else if a.state.CurrentView == models.ViewCompose && a.composeView.IsViewingContainers() {
				// Only allow delete when viewing containers in a scaled service
				if container := a.composeView.GetSelectedContainer(); container != nil {
					a.modal = components.NewConfirmModal(
						"Delete Container",
						fmt.Sprintf("Are you sure you want to remove container '%s'?", container.Name),
					)
					a.modal.SetSize(a.width, a.height)
					a.pendingDelete = container.ID
					a.pendingDeleteType = "container"
					return a, nil
				}
			} else if a.state.CurrentView == models.ViewNetworks && a.networksView.GetCurrentTab() == models.NetworksContainersTab {
				if container := a.networksView.GetSelectedInNetworkContainer(); container != nil {
					a.modal = components.NewConfirmModal(
						"Delete Container",
						fmt.Sprintf("Are you sure you want to remove container '%s'?", container.Name),
					)
					a.modal.SetSize(a.width, a.height)
					a.pendingDelete = container.ID
					a.pendingDeleteType = "container"
					return a, nil
				}
			} else if a.state.CurrentView == models.ViewNetworks && a.networksView.GetCurrentTab() == models.NetworksListTab {
				if network := a.networksView.GetSelectedNetwork(); network != nil {
					if network.IsSystemNetwork() {
						a.errorMessage = "Cannot delete system network"
						return a, clearStatus(2 * time.Second)
					}
					a.modal = components.NewConfirmModal(
						"Delete Network",
						fmt.Sprintf("Are you sure you want to remove network '%s'?", network.Name),
					)
					a.modal.SetSize(a.width, a.height)
					a.pendingDelete = network.ID
					a.pendingDeleteType = "network"
					return a, nil
				}
			}

		case "p":
			// Pull image (Images view) or Prune volumes (Volumes view)
			if a.state.CurrentView == models.ViewImages {
				a.modal = components.NewFormModal("Pull Image", []string{"Image Name (e.g. nginx:latest)"})
				a.modal.SetSize(a.width, a.height)
				a.pendingDeleteType = "pull_image"
				return a, nil
			} else if a.state.CurrentView == models.ViewVolumes {
				a.modal = components.NewConfirmModal(
					"Prune Unused Volumes",
					"Remove all volumes not used by at least one container?",
				)
				a.modal.SetSize(a.width, a.height)
				a.pendingDeleteType = "prune_volumes"
				return a, nil
			}
		}

	case DockerClientReadyMsg:
		a.docker = msg.client
		a.ready = true
		return a, fetchContainers(a.docker)

	case GroupManagerReadyMsg:
		a.groupManager = msg.manager
		// Load groups into the view
		groups := a.groupManager.GetAllGroups()
		a.groupsView.SetGroups(groups)

	case ContainersLoadedMsg:
		a.containersView.SetContainers(msg.containers)
		// Also pass to groups view, networks view, and volumes view for usage counting
		a.groupsView.SetAllContainers(msg.containers)
		a.networksView.SetAllContainers(msg.containers)
		a.volumesView.SetAllContainers(msg.containers)

	case ImagesLoadedMsg:
		a.imagesView.SetImages(msg.images)

	case GroupsLoadedMsg:
		a.groupsView.SetGroups(msg.groups)

	case VolumesLoadedMsg:
		a.volumesView.SetVolumes(msg.volumes)

	case ComposeProjectsLoadedMsg:
		a.composeView.SetProjects(msg.projects)

	case NetworksLoadedMsg:
		a.networksView.SetNetworks(msg.networks)

	case RefreshTickMsg:
		// Auto-refresh current view
		if !a.ready {
			return a, tickRefresh()
		}

		// Skip refresh if currently filtering to avoid clearing filter input
		if (a.state.CurrentView == models.ViewContainers && a.containersView.IsFiltering()) ||
		   (a.state.CurrentView == models.ViewImages && a.imagesView.IsFiltering()) ||
		   (a.state.CurrentView == models.ViewGroups && a.groupsView.IsFiltering()) ||
		   (a.state.CurrentView == models.ViewVolumes && a.volumesView.IsFiltering()) ||
		   (a.state.CurrentView == models.ViewCompose && a.composeView.IsFiltering()) ||
		   (a.state.CurrentView == models.ViewNetworks && a.networksView.IsFiltering()) {
			return a, tickRefresh()
		}

		var cmd tea.Cmd
		switch a.state.CurrentView {
		case models.ViewContainers:
			cmd = fetchContainers(a.docker)
		case models.ViewImages:
			cmd = fetchImages(a.docker)
		case models.ViewGroups:
			cmd = loadGroups(a.groupManager)
		case models.ViewVolumes:
			cmd = tea.Batch(fetchVolumes(a.docker), fetchContainers(a.docker))
		case models.ViewCompose:
			cmd = fetchComposeProjects(a.docker)
		case models.ViewNetworks:
			cmd = tea.Batch(fetchNetworks(a.docker), fetchContainers(a.docker))
		}

		return a, tea.Batch(cmd, tickRefresh())

	case ContainerStartedMsg:
		if msg.err != nil {
			a.errorMessage = fmt.Sprintf("Failed to start container: %v", msg.err)
		} else {
			a.statusMessage = fmt.Sprintf("Container %s started", msg.containerID[:12])
		}
		return a, tea.Batch(
			fetchContainers(a.docker),
			clearStatus(2*time.Second),
		)

	case ContainerStoppedMsg:
		if msg.err != nil {
			a.errorMessage = fmt.Sprintf("Failed to stop container: %v", msg.err)
		} else {
			a.statusMessage = fmt.Sprintf("Container %s stopped", msg.containerID[:12])
		}
		return a, tea.Batch(
			fetchContainers(a.docker),
			clearStatus(2*time.Second),
		)

	case ContainerRestartedMsg:
		if msg.err != nil {
			a.errorMessage = fmt.Sprintf("Failed to restart container: %v", msg.err)
		} else {
			a.statusMessage = fmt.Sprintf("Container %s restarted", msg.containerID[:12])
		}
		return a, tea.Batch(
			fetchContainers(a.docker),
			clearStatus(2*time.Second),
		)

	case ClearStatusMsg:
		a.statusMessage = ""
		a.errorMessage = ""

	case ImageRemovedMsg:
		if msg.err != nil {
			a.errorMessage = fmt.Sprintf("Failed to remove image: %v", msg.err)
		} else {
			a.statusMessage = fmt.Sprintf("Image %s removed", msg.imageID[:12])
		}
		return a, tea.Batch(
			fetchImages(a.docker),
			clearStatus(2*time.Second),
		)

	case GroupStartedMsg:
		if msg.err != nil {
			a.errorMessage = fmt.Sprintf("Failed to start group: %v", msg.err)
		} else {
			a.statusMessage = "Group started successfully"
		}
		return a, tea.Batch(
			fetchContainers(a.docker),
			clearStatus(2*time.Second),
		)

	case GroupStoppedMsg:
		if msg.err != nil {
			a.errorMessage = fmt.Sprintf("Failed to stop group: %v", msg.err)
		} else {
			a.statusMessage = "Group stopped successfully"
		}
		return a, tea.Batch(
			fetchContainers(a.docker),
			clearStatus(2*time.Second),
		)

	case GroupCreatedMsg:
		a.statusMessage = fmt.Sprintf("Group '%s' created successfully", msg.name)
		return a, tea.Batch(
			loadGroups(a.groupManager),
			clearStatus(2*time.Second),
		)

	case ContainerAddedToGroupMsg:
		if msg.err != nil {
			a.errorMessage = fmt.Sprintf("Failed to add container: %v", msg.err)
		} else {
			a.statusMessage = "Container added to group"
		}
		return a, tea.Batch(
			loadGroups(a.groupManager),
			fetchContainers(a.docker),
			clearStatus(2*time.Second),
		)

	case ContainerRemovedFromGroupMsg:
		if msg.err != nil {
			a.errorMessage = fmt.Sprintf("Failed to remove container: %v", msg.err)
		} else {
			a.statusMessage = "Container removed from group"
		}
		return a, tea.Batch(
			loadGroups(a.groupManager),
			fetchContainers(a.docker),
			clearStatus(2*time.Second),
		)

	case VolumeRemovedMsg:
		if msg.err != nil {
			a.errorMessage = fmt.Sprintf("Failed to remove volume: %v", msg.err)
		} else {
			a.statusMessage = fmt.Sprintf("Volume '%s' removed", msg.volumeName)
		}
		return a, tea.Batch(
			fetchVolumes(a.docker),
			clearStatus(2*time.Second),
		)

	case ComposeProjectStartedMsg:
		if msg.err != nil {
			a.errorMessage = fmt.Sprintf("Failed to start project: %v", msg.err)
		} else {
			a.statusMessage = fmt.Sprintf("Compose project '%s' started", msg.projectName)
		}
		return a, tea.Batch(
			fetchComposeProjects(a.docker),
			fetchContainers(a.docker),
			clearStatus(2*time.Second),
		)

	case ComposeProjectStoppedMsg:
		if msg.err != nil {
			a.errorMessage = fmt.Sprintf("Failed to stop project: %v", msg.err)
		} else {
			a.statusMessage = fmt.Sprintf("Compose project '%s' stopped", msg.projectName)
		}
		return a, tea.Batch(
			fetchComposeProjects(a.docker),
			fetchContainers(a.docker),
			clearStatus(2*time.Second),
		)

	case ComposeProjectRestartedMsg:
		if msg.err != nil {
			a.errorMessage = fmt.Sprintf("Failed to restart project: %v", msg.err)
		} else {
			a.statusMessage = fmt.Sprintf("Compose project '%s' restarted", msg.projectName)
		}
		return a, tea.Batch(
			fetchComposeProjects(a.docker),
			fetchContainers(a.docker),
			clearStatus(2*time.Second),
		)

	case ImagePullCompletedMsg:
		if msg.err != nil {
			a.errorMessage = fmt.Sprintf("Failed to pull image: %v", msg.err)
		} else {
			a.statusMessage = fmt.Sprintf("Image '%s' pulled successfully", msg.imageName)
		}
		return a, tea.Batch(
			fetchImages(a.docker),
			clearStatus(2*time.Second),
		)

	case ContainerConnectedToNetworkMsg:
		if msg.err != nil {
			a.errorMessage = fmt.Sprintf("Failed to connect container: %v", msg.err)
		} else {
			a.statusMessage = "Container connected to network"
		}
		return a, tea.Batch(
			fetchNetworks(a.docker),
			fetchContainers(a.docker),
			clearStatus(2*time.Second),
		)

	case ContainerDisconnectedFromNetworkMsg:
		if msg.err != nil {
			a.errorMessage = fmt.Sprintf("Failed to disconnect container: %v", msg.err)
		} else {
			a.statusMessage = "Container disconnected from network"
		}
		return a, tea.Batch(
			fetchNetworks(a.docker),
			fetchContainers(a.docker),
			clearStatus(2*time.Second),
		)

	case NetworkCreatedMsg:
		if msg.err != nil {
			a.errorMessage = fmt.Sprintf("Failed to create network: %v", msg.err)
		} else {
			a.statusMessage = fmt.Sprintf("Network '%s' created", msg.name)
		}
		return a, tea.Batch(
			fetchNetworks(a.docker),
			clearStatus(2*time.Second),
		)

	case NetworkRemovedMsg:
		if msg.err != nil {
			a.errorMessage = fmt.Sprintf("Failed to remove network: %v", msg.err)
		} else {
			a.statusMessage = "Network removed"
		}
		return a, tea.Batch(
			fetchNetworks(a.docker),
			clearStatus(2*time.Second),
		)

	case ContainerConfigLoadedMsg:
		if msg.err != nil {
			a.errorMessage = fmt.Sprintf("Failed to load container config: %v", msg.err)
			return a, clearStatus(3 * time.Second)
		}

		// Store config and switch to env vars view
		a.pendingEnvContainer = msg.config

		// Create a container reference from the config
		a.state.SelectedContainer = &models.Container{
			ID:   msg.containerID,
			Name: msg.config.Name,
		}

		a.state.PreviousView = a.state.CurrentView
		a.state.CurrentView = models.ViewEnvVars
		a.envVarsView.SetContainer(msg.containerID, msg.config.Name, msg.config.Env)
		return a, nil

	case ContainerRecreatedMsg:
		if msg.err != nil {
			a.errorMessage = fmt.Sprintf("Failed to rebuild container: %v", msg.err)
		} else {
			a.statusMessage = fmt.Sprintf("Container '%s' rebuilt with new environment", msg.containerName)
		}
		// Return to containers view
		a.pendingEnvContainer = nil
		a.state.CurrentView = models.ViewContainers
		a.sidebar.SetCurrentView(models.ViewContainers)
		return a, tea.Batch(
			fetchContainers(a.docker),
			clearStatus(3*time.Second),
		)

	case StatusMsg:
		a.statusMessage = msg.message
		return a, clearStatus(2 * time.Second)

	case ErrorMsg:
		a.errorMessage = msg.err.Error()
		return a, clearStatus(3 * time.Second)
	}

	// Delegate to current view
	var cmd tea.Cmd
	switch a.state.CurrentView {
	case models.ViewContainers:
		a.containersView, cmd = a.containersView.Update(msg)
	case models.ViewImages:
		a.imagesView, cmd = a.imagesView.Update(msg)
	case models.ViewGroups:
		a.groupsView, cmd = a.groupsView.Update(msg)
	case models.ViewVolumes:
		a.volumesView, cmd = a.volumesView.Update(msg)
	case models.ViewCompose:
		a.composeView, cmd = a.composeView.Update(msg)
	case models.ViewNetworks:
		a.networksView, cmd = a.networksView.Update(msg)
	case models.ViewLogs:
		a.logsView, cmd = a.logsView.Update(msg)
	case models.ViewStats:
		a.statsView, cmd = a.statsView.Update(msg)
	case models.ViewEnvVars:
		a.envVarsView, cmd = a.envVarsView.Update(msg)
	}

	return a, cmd
}

// View renders the application
func (a *App) View() string {
	if !a.ready {
		return "Initializing Docker UI...\n\nConnecting to Docker daemon..."
	}

	// If modal is visible, show it on top
	if a.modal != nil && a.modal.IsVisible() {
		return a.modal.View()
	}

	var mainContent string

	// Render current view based on state
	switch a.state.CurrentView {
	case models.ViewContainers:
		mainContent = a.containersView.View()
	case models.ViewImages:
		mainContent = a.imagesView.View()
	case models.ViewGroups:
		mainContent = a.groupsView.View()
	case models.ViewVolumes:
		mainContent = a.volumesView.View()
	case models.ViewCompose:
		mainContent = a.composeView.View()
	case models.ViewNetworks:
		mainContent = a.networksView.View()
	case models.ViewLogs:
		// Logs and stats take full screen (no sidebar)
		return lipgloss.JoinVertical(
			lipgloss.Left,
			a.logsView.View(),
			a.renderFooter(),
		)
	case models.ViewStats:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			a.statsView.View(),
			a.renderFooter(),
		)
	case models.ViewEnvVars:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			a.envVarsView.View(),
			a.renderFooter(),
		)
	case models.ViewAbout:
		// About page takes full screen (no sidebar)
		return lipgloss.JoinVertical(
			lipgloss.Left,
			a.aboutView.View(),
			a.renderFooter(),
		)
	default:
		mainContent = "Unknown view"
	}

	// Layout with sidebar for main views
	sidebar := a.sidebar.View()

	// Combine sidebar and main content horizontally
	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebar,
		mainContent,
	)

	// Add footer
	footer := a.renderFooter()

	// Combine vertically
	return lipgloss.JoinVertical(
		lipgloss.Left,
		body,
		footer,
	)
}

func (a *App) renderFooter() string {
	var footer string

	// Status message
	if a.errorMessage != "" {
		footer += styles.ErrorStyle.Render("✗ " + a.errorMessage)
	} else if a.statusMessage != "" {
		footer += styles.SuccessStyle.Render("✓ " + a.statusMessage)
	} else {
		// Help text
		switch a.state.CurrentView {
		case models.ViewContainers:
			footer += a.containersView.GetHelpText()
		case models.ViewImages:
			footer += a.imagesView.GetHelpText()
		case models.ViewGroups:
			footer += a.groupsView.GetHelpText()
		case models.ViewVolumes:
			footer += a.volumesView.GetHelpText()
		case models.ViewCompose:
			footer += a.composeView.GetHelpText()
		case models.ViewNetworks:
			footer += a.networksView.GetHelpText()
		case models.ViewLogs:
			footer += a.logsView.GetHelpText()
		case models.ViewStats:
			footer += a.statsView.GetHelpText()
		case models.ViewEnvVars:
			footer += a.envVarsView.GetHelpText()
		case models.ViewAbout:
			footer += a.aboutView.GetHelpText()
		}
	}

	return footer
}

// cycleTabForward cycles to the next tab
func (a *App) cycleTabForward() (tea.Model, tea.Cmd) {
	a.state.PreviousView = a.state.CurrentView

	switch a.state.CurrentView {
	case models.ViewContainers:
		a.state.CurrentView = models.ViewImages
		a.sidebar.SetCurrentView(models.ViewImages)
		return a, fetchImages(a.docker)
	case models.ViewImages:
		a.state.CurrentView = models.ViewGroups
		a.sidebar.SetCurrentView(models.ViewGroups)
		return a, loadGroups(a.groupManager)
	case models.ViewGroups:
		a.state.CurrentView = models.ViewVolumes
		a.sidebar.SetCurrentView(models.ViewVolumes)
		return a, tea.Batch(fetchVolumes(a.docker), fetchContainers(a.docker))
	case models.ViewVolumes:
		a.state.CurrentView = models.ViewCompose
		a.sidebar.SetCurrentView(models.ViewCompose)
		return a, fetchComposeProjects(a.docker)
	case models.ViewCompose:
		a.state.CurrentView = models.ViewNetworks
		a.sidebar.SetCurrentView(models.ViewNetworks)
		return a, tea.Batch(fetchNetworks(a.docker), fetchContainers(a.docker))
	case models.ViewNetworks:
		a.state.CurrentView = models.ViewAbout
		a.sidebar.SetCurrentView(models.ViewAbout)
		return a, nil
	case models.ViewAbout:
		a.state.CurrentView = models.ViewContainers
		a.sidebar.SetCurrentView(models.ViewContainers)
		return a, fetchContainers(a.docker)
	}

	return a, nil
}

// cycleTabBackward cycles to the previous tab
func (a *App) cycleTabBackward() (tea.Model, tea.Cmd) {
	a.state.PreviousView = a.state.CurrentView

	switch a.state.CurrentView {
	case models.ViewContainers:
		a.state.CurrentView = models.ViewAbout
		a.sidebar.SetCurrentView(models.ViewAbout)
		return a, nil
	case models.ViewAbout:
		a.state.CurrentView = models.ViewNetworks
		a.sidebar.SetCurrentView(models.ViewNetworks)
		return a, tea.Batch(fetchNetworks(a.docker), fetchContainers(a.docker))
	case models.ViewImages:
		a.state.CurrentView = models.ViewContainers
		a.sidebar.SetCurrentView(models.ViewContainers)
		return a, fetchContainers(a.docker)
	case models.ViewGroups:
		a.state.CurrentView = models.ViewImages
		a.sidebar.SetCurrentView(models.ViewImages)
		return a, fetchImages(a.docker)
	case models.ViewVolumes:
		a.state.CurrentView = models.ViewGroups
		a.sidebar.SetCurrentView(models.ViewGroups)
		return a, loadGroups(a.groupManager)
	case models.ViewCompose:
		a.state.CurrentView = models.ViewVolumes
		a.sidebar.SetCurrentView(models.ViewVolumes)
		return a, tea.Batch(fetchVolumes(a.docker), fetchContainers(a.docker))
	case models.ViewNetworks:
		a.state.CurrentView = models.ViewCompose
		a.sidebar.SetCurrentView(models.ViewCompose)
		return a, fetchComposeProjects(a.docker)
	}

	return a, nil
}

// handleModalConfirmed handles the confirmed modal action
func (a *App) handleModalConfirmed() (tea.Model, tea.Cmd) {
	defer func() {
		a.modal = nil
	}()

	switch a.pendingDeleteType {
	case "container":
		return a, removeContainer(a.docker, a.pendingDelete)

	case "image":
		return a, removeImage(a.docker, a.pendingDelete)

	case "group":
		return a, deleteGroup(a.groupManager, a.pendingDelete)

	case "remove_from_group":
		if selectedGroup := a.groupsView.GetSelectedGroupForApp(); selectedGroup != nil {
			return a, removeContainerFromGroup(a.groupManager, selectedGroup.ID, a.pendingDelete)
		}

	case "create_group":
		// Get form values
		values := a.modal.GetInputValues()
		if len(values) >= 2 {
			name := values[0]
			description := values[1]

			// Create group with selected containers
			// For now, create empty group - user can add containers later
			return a, createGroup(a.groupManager, name, description, []string{})
		}

	case "volume":
		return a, removeVolume(a.docker, a.pendingDelete)

	case "prune_volumes":
		return a, pruneVolumes(a.docker)

	case "pull_image":
		// Get form values
		values := a.modal.GetInputValues()
		if len(values) >= 1 && values[0] != "" {
			imageName := values[0]
			return a, pullImage(a.docker, imageName)
		}

	case "create_network":
		// Get form values
		values := a.modal.GetInputValues()
		if len(values) >= 1 && values[0] != "" {
			networkName := values[0]
			driver := "bridge" // Default driver
			if len(values) >= 2 && values[1] != "" {
				driver = values[1]
			}
			return a, createNetwork(a.docker, networkName, driver)
		}

	case "network":
		return a, removeNetwork(a.docker, a.pendingDelete)

	case "disconnect_from_network":
		if selectedNetwork := a.networksView.GetSelectedNetworkForApp(); selectedNetwork != nil {
			return a, disconnectContainerFromNetwork(a.docker, selectedNetwork.ID, a.pendingDelete)
		}
	}

	a.pendingDelete = ""
	a.pendingDeleteType = ""
	return a, nil
}

// Commands

func initDockerClient() tea.Cmd {
	return func() tea.Msg {
		client, err := docker.NewClient()
		if err != nil {
			return ErrorMsg{err: fmt.Errorf("failed to initialize Docker client: %w", err)}
		}
		return DockerClientReadyMsg{client: client}
	}
}

func initGroupManager() tea.Cmd {
	return func() tea.Msg {
		gm, err := config.NewGroupManager()
		if err != nil {
			// Non-fatal, just log
			return ErrorMsg{err: fmt.Errorf("failed to load groups: %w", err)}
		}
		return GroupManagerReadyMsg{manager: gm}
	}
}

func fetchContainers(client *docker.Client) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		containers, err := client.ListContainers(ctx, true)
		if err != nil {
			return ErrorMsg{err: err}
		}

		return ContainersLoadedMsg{containers: containers}
	}
}

func startContainer(client *docker.Client, containerID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := client.StartContainer(ctx, containerID)
		return ContainerStartedMsg{containerID: containerID, err: err}
	}
}

func stopContainer(client *docker.Client, containerID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := client.StopContainer(ctx, containerID, 10)
		return ContainerStoppedMsg{containerID: containerID, err: err}
	}
}

func restartContainer(client *docker.Client, containerID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := client.RestartContainer(ctx, containerID, 10)
		return ContainerRestartedMsg{containerID: containerID, err: err}
	}
}

func tickRefresh() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return RefreshTickMsg{}
	})
}

func clearStatus(duration time.Duration) tea.Cmd {
	return tea.Tick(duration, func(t time.Time) tea.Msg {
		return ClearStatusMsg{}
	})
}

func fetchImages(client *docker.Client) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		images, err := client.ListImages(ctx)
		if err != nil {
			return ErrorMsg{err: err}
		}

		return ImagesLoadedMsg{images: images}
	}
}

func removeImage(client *docker.Client, imageID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := client.RemoveImage(ctx, imageID, false)
		return ImageRemovedMsg{imageID: imageID, err: err}
	}
}

func loadGroups(groupManager *config.GroupManager) tea.Cmd {
	return func() tea.Msg {
		if groupManager == nil {
			return nil
		}

		groups := groupManager.GetAllGroups()
		return GroupsLoadedMsg{groups: groups}
	}
}

func startGroup(client *docker.Client, groupManager *config.GroupManager, groupID string) tea.Cmd {
	return func() tea.Msg {
		if client == nil || groupManager == nil {
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		operation := func(ctx context.Context, containerID string) error {
			return client.StartContainer(ctx, containerID)
		}

		err := groupManager.ExecuteGroupOperation(ctx, groupID, operation)
		return GroupStartedMsg{groupID: groupID, err: err}
	}
}

func stopGroup(client *docker.Client, groupManager *config.GroupManager, groupID string) tea.Cmd {
	return func() tea.Msg {
		if client == nil || groupManager == nil {
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		operation := func(ctx context.Context, containerID string) error {
			return client.StopContainer(ctx, containerID, 10)
		}

		err := groupManager.ExecuteGroupOperation(ctx, groupID, operation)
		return GroupStoppedMsg{groupID: groupID, err: err}
	}
}

func addContainerToGroup(gm *config.GroupManager, groupID, containerID string) tea.Cmd {
	return func() tea.Msg {
		err := gm.AddContainerToGroup(groupID, containerID)
		return ContainerAddedToGroupMsg{
			groupID:     groupID,
			containerID: containerID,
			err:         err,
		}
	}
}

func removeContainerFromGroup(gm *config.GroupManager, groupID, containerID string) tea.Cmd {
	return func() tea.Msg {
		err := gm.RemoveContainerFromGroup(groupID, containerID)
		return ContainerRemovedFromGroupMsg{
			groupID:     groupID,
			containerID: containerID,
			err:         err,
		}
	}
}

func startLogStreaming(client *docker.Client, logsView *views.LogsView, container *models.Container) tea.Cmd {
	// Set container synchronously to reset the view state before the async Cmd runs
	// This prevents race conditions where View() is called with stale data
	logsView.SetContainer(container.ID, container.Name)

	return func() tea.Msg {
		if client == nil {
			return nil
		}

		ctx := context.Background()
		logsChan, errorChan := client.StreamLogs(ctx, container.ID, true, time.Time{}, "100")
		logsView.StartStreaming(logsChan, errorChan)

		// Return the first log wait command
		return waitForLogEntry(logsChan, errorChan)()
	}
}

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
			return ErrorMsg{err: err}
		}
	}
}

func startStatsStreaming(client *docker.Client, statsView *views.StatsView, container *models.Container) tea.Cmd {
	// Set container synchronously to reset the view state before the async Cmd runs
	// This prevents race conditions where View() is called with stale data
	statsView.SetContainer(container.ID, container.Name)

	return func() tea.Msg {
		if client == nil {
			return nil
		}

		ctx := context.Background()
		statsChan, errorChan := client.StreamStats(ctx, container.ID)
		statsView.StartStreaming(statsChan, errorChan)

		// Return the first stats wait command
		return waitForStats(statsChan, errorChan)()
	}
}

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
			return ErrorMsg{err: err}
		}
	}
}

func execShell(containerID, containerName string) tea.Cmd {
	// Try sh first (most compatible)
	cmd := exec.Command("docker", "exec", "-it", containerID, "sh")

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return ErrorMsg{err: fmt.Errorf("failed to exec shell in %s: %w", containerName, err)}
		}
		return StatusMsg{message: fmt.Sprintf("Exited shell for %s", containerName)}
	})
}

func removeContainer(client *docker.Client, containerID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := client.RemoveContainer(ctx, containerID, true) // force=true
		return ContainerRemovedMsg{containerID: containerID, err: err}
	}
}

func deleteGroup(groupManager *config.GroupManager, groupID string) tea.Cmd {
	return func() tea.Msg {
		if groupManager == nil {
			return ErrorMsg{err: fmt.Errorf("group manager not initialized")}
		}

		err := groupManager.DeleteGroup(groupID)
		if err != nil {
			return ErrorMsg{err: fmt.Errorf("failed to delete group: %w", err)}
		}

		return StatusMsg{message: "Group deleted successfully"}
	}
}

func createGroup(groupManager *config.GroupManager, name, description string, containerIDs []string) tea.Cmd {
	return func() tea.Msg {
		if groupManager == nil {
			return ErrorMsg{err: fmt.Errorf("group manager not initialized")}
		}

		_, err := groupManager.CreateGroup(name, description, containerIDs)
		if err != nil {
			return ErrorMsg{err: fmt.Errorf("failed to create group: %w", err)}
		}

		// Return a message that will trigger group reload
		return GroupCreatedMsg{name: name}
	}
}

// Volume commands
func fetchVolumes(client *docker.Client) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		volumes, err := client.ListVolumes(ctx)
		if err != nil {
			return ErrorMsg{err: err}
		}

		return VolumesLoadedMsg{volumes: volumes}
	}
}

func removeVolume(client *docker.Client, volumeName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := client.RemoveVolume(ctx, volumeName, false)
		return VolumeRemovedMsg{volumeName: volumeName, err: err}
	}
}

func pruneVolumes(client *docker.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := client.PruneUnusedVolumes(ctx)
		if err != nil {
			return ErrorMsg{err: err}
		}

		return StatusMsg{message: "Unused volumes pruned successfully"}
	}
}

// Compose commands
func fetchComposeProjects(client *docker.Client) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		projects, err := client.ListComposeProjects(ctx)
		if err != nil {
			return ErrorMsg{err: err}
		}

		return ComposeProjectsLoadedMsg{projects: projects}
	}
}

func startComposeProject(client *docker.Client, projectName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := client.StartComposeProject(ctx, projectName)
		return ComposeProjectStartedMsg{projectName: projectName, err: err}
	}
}

func stopComposeProject(client *docker.Client, projectName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := client.StopComposeProject(ctx, projectName, 10)
		return ComposeProjectStoppedMsg{projectName: projectName, err: err}
	}
}

func restartComposeProject(client *docker.Client, projectName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := client.RestartComposeProject(ctx, projectName, 10)
		return ComposeProjectRestartedMsg{projectName: projectName, err: err}
	}
}

// Image pull command
func pullImage(client *docker.Client, imageName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		err := client.PullImage(ctx, imageName)
		return ImagePullCompletedMsg{imageName: imageName, err: err}
	}
}

// Network commands
func fetchNetworks(client *docker.Client) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		networks, err := client.ListNetworks(ctx)
		if err != nil {
			return ErrorMsg{err: err}
		}

		return NetworksLoadedMsg{networks: networks}
	}
}

func connectContainerToNetwork(client *docker.Client, networkID, containerID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := client.ConnectContainer(ctx, networkID, containerID)
		return ContainerConnectedToNetworkMsg{networkID: networkID, containerID: containerID, err: err}
	}
}

func disconnectContainerFromNetwork(client *docker.Client, networkID, containerID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := client.DisconnectContainer(ctx, networkID, containerID, false)
		return ContainerDisconnectedFromNetworkMsg{networkID: networkID, containerID: containerID, err: err}
	}
}

func createNetwork(client *docker.Client, name, driver string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := client.CreateNetwork(ctx, name, driver)
		return NetworkCreatedMsg{name: name, err: err}
	}
}

func removeNetwork(client *docker.Client, networkID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := client.RemoveNetwork(ctx, networkID)
		return NetworkRemovedMsg{networkID: networkID, err: err}
	}
}

// Container env var editing commands
func loadContainerConfig(client *docker.Client, containerID string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return ErrorMsg{err: fmt.Errorf("docker client not initialized")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		config, err := client.InspectContainerFull(ctx, containerID)
		return ContainerConfigLoadedMsg{
			containerID: containerID,
			config:      config,
			err:         err,
		}
	}
}

func recreateContainer(client *docker.Client, containerID string, config *models.ContainerFullConfig) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return ErrorMsg{err: fmt.Errorf("docker client not initialized")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		newID, err := client.RecreateContainer(ctx, containerID, config)
		return ContainerRecreatedMsg{
			oldID:         containerID,
			newID:         newID,
			containerName: config.Name,
			err:           err,
		}
	}
}
