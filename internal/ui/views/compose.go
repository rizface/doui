package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rizface/doui/internal/models"
	"github.com/rizface/doui/internal/ui/styles"
)

// ComposeProjectItem implements list.Item for compose projects
type ComposeProjectItem struct {
	project models.ComposeProject
}

func (i ComposeProjectItem) FilterValue() string {
	return i.project.Name
}

func (i ComposeProjectItem) Title() string {
	status := ""
	if i.project.AllRunning() {
		status = styles.RunningStyle.Render("all running")
	} else if i.project.GetRunningCount() > 0 {
		status = styles.PausedStyle.Render(fmt.Sprintf("%d/%d running", i.project.GetRunningCount(), i.project.GetContainerCount()))
	} else {
		status = styles.StoppedStyle.Render("stopped")
	}
	return fmt.Sprintf("%s  %s", i.project.Name, status)
}

func (i ComposeProjectItem) Description() string {
	return fmt.Sprintf("%d services, %d containers", i.project.GetServiceCount(), i.project.GetContainerCount())
}

// ComposeServiceItem implements list.Item for services within a project
type ComposeServiceItem struct {
	service models.ComposeService
}

func (i ComposeServiceItem) FilterValue() string {
	return i.service.Name
}

func (i ComposeServiceItem) Title() string {
	runningCount := 0
	for _, c := range i.service.Containers {
		if c.IsRunning() {
			runningCount++
		}
	}

	status := ""
	totalCount := len(i.service.Containers)
	if runningCount == totalCount && totalCount > 0 {
		status = styles.RunningStyle.Render("running")
	} else if runningCount > 0 {
		status = styles.PausedStyle.Render(fmt.Sprintf("%d/%d running", runningCount, totalCount))
	} else {
		status = styles.StoppedStyle.Render("stopped")
	}

	return fmt.Sprintf("%s  %s", i.service.Name, status)
}

func (i ComposeServiceItem) Description() string {
	if len(i.service.Containers) == 1 {
		c := i.service.Containers[0]
		return fmt.Sprintf("ID: %s | Image: %s", c.ShortID, c.Image)
	}
	return fmt.Sprintf("%d containers (scaled)", len(i.service.Containers))
}

// ComposeContainerItem implements list.Item for containers within a service
type ComposeContainerItem struct {
	container models.Container
}

func (i ComposeContainerItem) FilterValue() string {
	return i.container.Name
}

func (i ComposeContainerItem) Title() string {
	status := styles.GetStatusStyle(i.container.State).Render(i.container.State)
	return fmt.Sprintf("%s  %s", i.container.Name, status)
}

func (i ComposeContainerItem) Description() string {
	return fmt.Sprintf("ID: %s | Image: %s | %s", i.container.ShortID, i.container.Image, i.container.Status)
}

// ComposeView displays Docker Compose projects
type ComposeView struct {
	projectsList   list.Model
	servicesList   list.Model
	containersList list.Model

	projects        []models.ComposeProject
	selectedProject *models.ComposeProject
	selectedService *models.ComposeService
	viewingServices   bool
	viewingContainers bool

	width  int
	height int
}

// NewComposeView creates a new compose view
func NewComposeView() *ComposeView {
	// Projects list
	projectsDelegate := list.NewDefaultDelegate()
	projectsDelegate.SetHeight(2)
	projectsDelegate.SetSpacing(1)

	projectsList := list.New([]list.Item{}, projectsDelegate, 0, 0)
	projectsList.Title = "Docker Compose Projects"
	projectsList.SetShowStatusBar(true)
	projectsList.SetFilteringEnabled(true)
	projectsList.Styles.Title = styles.TitleStyle

	// Services list
	servicesDelegate := list.NewDefaultDelegate()
	servicesDelegate.SetHeight(2)
	servicesDelegate.SetSpacing(1)

	servicesList := list.New([]list.Item{}, servicesDelegate, 0, 0)
	servicesList.Title = "Services"
	servicesList.SetShowStatusBar(true)
	servicesList.SetFilteringEnabled(true)
	servicesList.Styles.Title = styles.TitleStyle

	// Containers list (for scaled services)
	containersDelegate := list.NewDefaultDelegate()
	containersDelegate.SetHeight(2)
	containersDelegate.SetSpacing(1)

	containersList := list.New([]list.Item{}, containersDelegate, 0, 0)
	containersList.Title = "Containers"
	containersList.SetShowStatusBar(true)
	containersList.SetFilteringEnabled(true)
	containersList.Styles.Title = styles.TitleStyle

	return &ComposeView{
		projectsList:      projectsList,
		servicesList:      servicesList,
		containersList:    containersList,
		viewingServices:   false,
		viewingContainers: false,
	}
}

// SetProjects updates the list of compose projects
func (v *ComposeView) SetProjects(projects []models.ComposeProject) {
	v.projects = projects

	items := make([]list.Item, len(projects))
	for i, p := range projects {
		items[i] = ComposeProjectItem{project: p}
	}

	v.projectsList.SetItems(items)

	// Update selectedProject to point to new data (if still exists)
	if v.selectedProject != nil {
		found := false
		for i, p := range projects {
			if p.Name == v.selectedProject.Name {
				v.selectedProject = &v.projects[i]
				found = true
				break
			}
		}
		if !found {
			v.selectedProject = nil
			v.selectedService = nil
			v.viewingServices = false
			v.viewingContainers = false
			return
		}

		// Update services list with new data
		v.updateServicesList()

		// Update selectedService to point to new data (if still exists)
		if v.selectedService != nil {
			serviceFound := false
			for i, s := range v.selectedProject.Services {
				if s.Name == v.selectedService.Name {
					v.selectedService = &v.selectedProject.Services[i]
					serviceFound = true
					break
				}
			}
			if !serviceFound {
				v.selectedService = nil
				v.viewingContainers = false
			} else if v.viewingContainers {
				// Update containers list with new data
				v.updateContainersList()
			}
		}
	}
}

// SetSize updates the view dimensions
func (v *ComposeView) SetSize(width, height int) {
	v.width = width
	v.height = height
	listHeight := height - 6
	v.projectsList.SetSize(width, listHeight)
	v.servicesList.SetSize(width, listHeight)
	v.containersList.SetSize(width, listHeight)
}

// GetSelectedProject returns the currently selected project
func (v *ComposeView) GetSelectedProject() *models.ComposeProject {
	if len(v.projects) == 0 || v.projectsList.Index() >= len(v.projects) {
		return nil
	}
	return &v.projects[v.projectsList.Index()]
}

// GetSelectedService returns the currently selected service
func (v *ComposeView) GetSelectedService() *models.ComposeService {
	if v.selectedProject == nil {
		return nil
	}
	if len(v.selectedProject.Services) == 0 || v.servicesList.Index() >= len(v.selectedProject.Services) {
		return nil
	}
	return &v.selectedProject.Services[v.servicesList.Index()]
}

// GetSelectedContainer returns the currently selected container
// When viewing containers list: returns the selected container from the list
// When viewing services with single container: returns that container
func (v *ComposeView) GetSelectedContainer() *models.Container {
	if v.viewingContainers && v.selectedService != nil {
		// We're viewing containers in a scaled service
		if len(v.selectedService.Containers) == 0 || v.containersList.Index() >= len(v.selectedService.Containers) {
			return nil
		}
		return &v.selectedService.Containers[v.containersList.Index()]
	}

	// We're viewing services - check if selected service has exactly 1 container
	service := v.GetSelectedService()
	if service != nil && len(service.Containers) == 1 {
		return &service.Containers[0]
	}

	return nil
}

// updateContainersList updates the containers list based on selected service
func (v *ComposeView) updateContainersList() {
	if v.selectedService == nil {
		v.containersList.SetItems([]list.Item{})
		return
	}

	items := make([]list.Item, len(v.selectedService.Containers))
	for i, c := range v.selectedService.Containers {
		items[i] = ComposeContainerItem{container: c}
	}

	v.containersList.SetItems(items)
	v.containersList.Title = fmt.Sprintf("Containers in '%s'", v.selectedService.Name)
}

// Update handles messages
func (v *ComposeView) Update(msg tea.Msg) (*ComposeView, tea.Cmd) {
	// If filtering, pass to active list
	if v.IsFiltering() {
		var cmd tea.Cmd
		if v.viewingContainers {
			v.containersList, cmd = v.containersList.Update(msg)
		} else if v.viewingServices {
			v.servicesList, cmd = v.servicesList.Update(msg)
		} else {
			v.projectsList, cmd = v.projectsList.Update(msg)
		}
		return v, cmd
	}

	// Handle key messages
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if v.viewingContainers {
				// Already at container level, enter does nothing
				return v, nil
			} else if v.viewingServices {
				// Select service and show containers if scaled
				v.selectedService = v.GetSelectedService()
				if v.selectedService != nil && len(v.selectedService.Containers) > 1 {
					// Scaled service - show containers list
					v.viewingContainers = true
					v.updateContainersList()
				}
				// If single container, operations work directly on that container
				return v, nil
			} else {
				// Select project and switch to services view
				v.selectedProject = v.GetSelectedProject()
				if v.selectedProject != nil {
					v.viewingServices = true
					v.updateServicesList()
				}
				return v, nil
			}

		case "esc":
			if v.viewingContainers {
				// Return to services list
				v.viewingContainers = false
				v.selectedService = nil
				return v, nil
			} else if v.viewingServices {
				// Return to projects list
				v.viewingServices = false
				v.selectedProject = nil
				return v, nil
			}
		}
	}

	// Delegate to current list
	var cmd tea.Cmd
	if v.viewingContainers {
		v.containersList, cmd = v.containersList.Update(msg)
	} else if v.viewingServices {
		v.servicesList, cmd = v.servicesList.Update(msg)
	} else {
		v.projectsList, cmd = v.projectsList.Update(msg)
	}

	return v, cmd
}

// updateServicesList updates the services list based on selected project
func (v *ComposeView) updateServicesList() {
	if v.selectedProject == nil {
		v.servicesList.SetItems([]list.Item{})
		return
	}

	items := make([]list.Item, len(v.selectedProject.Services))
	for i, s := range v.selectedProject.Services {
		items[i] = ComposeServiceItem{service: s}
	}

	v.servicesList.SetItems(items)
	v.servicesList.Title = fmt.Sprintf("Services in '%s'", v.selectedProject.Name)
}

// View renders the view
func (v *ComposeView) View() string {
	if v.viewingContainers {
		if v.selectedService == nil {
			return v.renderEmpty("Select a service to view its containers")
		}
		if len(v.selectedService.Containers) == 0 {
			return v.renderEmpty(fmt.Sprintf("No containers in '%s'", v.selectedService.Name))
		}
		return v.containersList.View()
	}

	if v.viewingServices {
		if v.selectedProject == nil {
			return v.renderEmpty("Select a project to view its services")
		}
		if len(v.selectedProject.Services) == 0 {
			return v.renderEmpty(fmt.Sprintf("No services in '%s'", v.selectedProject.Name))
		}
		return v.servicesList.View()
	}

	if len(v.projects) == 0 {
		return v.renderEmpty("No Docker Compose projects found.\nCompose projects are detected automatically from running containers.")
	}

	return v.projectsList.View()
}

func (v *ComposeView) renderEmpty(message string) string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("Docker Compose Projects"))
	b.WriteString("\n\n")
	b.WriteString(styles.SubtitleStyle.Render(message))

	return b.String()
}

// IsFiltering returns true if the active list is in filtering mode
func (v *ComposeView) IsFiltering() bool {
	if v.viewingContainers {
		return v.containersList.FilterState() == list.Filtering
	}
	if v.viewingServices {
		return v.servicesList.FilterState() == list.Filtering
	}
	return v.projectsList.FilterState() == list.Filtering
}

// IsViewingServices returns true if currently viewing services detail
func (v *ComposeView) IsViewingServices() bool {
	return v.viewingServices
}

// IsViewingContainers returns true if currently viewing containers in a scaled service
func (v *ComposeView) IsViewingContainers() bool {
	return v.viewingContainers
}

// GetHelpText returns help text for the compose view
func (v *ComposeView) GetHelpText() string {
	var helps []string

	if v.viewingContainers {
		// Viewing containers in a scaled service - full container operations
		helps = []string{
			styles.KeyStyle.Render("↑/↓") + " navigate",
			styles.KeyStyle.Render("s") + " start",
			styles.KeyStyle.Render("x") + " stop",
			styles.KeyStyle.Render("r") + " restart",
			styles.KeyStyle.Render("l") + " logs",
			styles.KeyStyle.Render("t") + " stats",
			styles.KeyStyle.Render("e") + " shell",
			styles.KeyStyle.Render("v") + " env",
			styles.KeyStyle.Render("d") + " remove",
			styles.KeyStyle.Render("esc") + " back",
			styles.KeyStyle.Render("/") + " filter",
		}
	} else if v.viewingServices {
		// Viewing services - show container ops for single-container services
		helps = []string{
			styles.KeyStyle.Render("↑/↓") + " navigate",
			styles.KeyStyle.Render("enter") + " containers",
			styles.KeyStyle.Render("s") + " start",
			styles.KeyStyle.Render("x") + " stop",
			styles.KeyStyle.Render("r") + " restart",
			styles.KeyStyle.Render("l") + " logs",
			styles.KeyStyle.Render("t") + " stats",
			styles.KeyStyle.Render("e") + " shell",
			styles.KeyStyle.Render("v") + " env",
			styles.KeyStyle.Render("esc") + " back",
			styles.KeyStyle.Render("/") + " filter",
		}
	} else {
		// Viewing projects
		helps = []string{
			styles.KeyStyle.Render("↑/↓") + " navigate",
			styles.KeyStyle.Render("enter") + " view services",
			styles.KeyStyle.Render("s") + " start all",
			styles.KeyStyle.Render("x") + " stop all",
			styles.KeyStyle.Render("r") + " restart all",
			styles.KeyStyle.Render("/") + " filter",
		}
	}

	helps = append(helps, styles.KeyStyle.Render("q")+" quit")
	return strings.Join(helps, styles.SeparatorStyle.String())
}
