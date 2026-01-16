package models

// ViewType represents different screens in the application
type ViewType int

const (
	ViewContainers ViewType = iota
	ViewImages
	ViewGroups
	ViewVolumes
	ViewCompose
	ViewNetworks
	ViewLogs
	ViewStats
	ViewEnvVars
	ViewAbout
)

// String returns the string representation of ViewType
func (v ViewType) String() string {
	switch v {
	case ViewContainers:
		return "Containers"
	case ViewImages:
		return "Images"
	case ViewGroups:
		return "Groups"
	case ViewVolumes:
		return "Volumes"
	case ViewCompose:
		return "Compose"
	case ViewNetworks:
		return "Networks"
	case ViewLogs:
		return "Logs"
	case ViewStats:
		return "Stats"
	case ViewEnvVars:
		return "Environment Variables"
	case ViewAbout:
		return "About"
	default:
		return "Unknown"
	}
}

// GroupsTabType represents tabs within the Groups view
type GroupsTabType int

const (
	GroupsListTab       GroupsTabType = iota // Tab 1: List of groups
	GroupsContainersTab                      // Tab 2: Containers in selected group
	GroupsAvailableTab                       // Tab 3: Containers available to add
)

// NetworksTabType represents tabs within the Networks view
type NetworksTabType int

const (
	NetworksListTab       NetworksTabType = iota // Tab 1: List of networks
	NetworksContainersTab                        // Tab 2: Containers in selected network
	NetworksAvailableTab                         // Tab 3: Containers available to attach
)

// AppState represents the global application state
type AppState struct {
	CurrentView       ViewType
	PreviousView      ViewType
	SelectedContainer *Container
}

// NewAppState creates a new application state with default values
func NewAppState() *AppState {
	return &AppState{
		CurrentView:  ViewContainers,
		PreviousView: ViewContainers,
	}
}
