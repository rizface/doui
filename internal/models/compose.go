package models

// ComposeProject represents a Docker Compose project
type ComposeProject struct {
	Name         string
	Services     []ComposeService
	ConfigHash   string
	WorkingDir   string
	ContainerIDs []string // All container IDs in this project
}

// ComposeService represents a service within a compose project
type ComposeService struct {
	Name       string
	Containers []Container // Containers for this service (can be scaled)
}

// GetContainerCount returns the total number of containers in the project
func (p *ComposeProject) GetContainerCount() int {
	return len(p.ContainerIDs)
}

// GetServiceCount returns the number of services in the project
func (p *ComposeProject) GetServiceCount() int {
	return len(p.Services)
}

// GetRunningCount returns the number of running containers
func (p *ComposeProject) GetRunningCount() int {
	count := 0
	for _, service := range p.Services {
		for _, container := range service.Containers {
			if container.IsRunning() {
				count++
			}
		}
	}
	return count
}

// AllRunning returns true if all containers in the project are running
func (p *ComposeProject) AllRunning() bool {
	if len(p.ContainerIDs) == 0 {
		return false
	}
	return p.GetRunningCount() == len(p.ContainerIDs)
}
