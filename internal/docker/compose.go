package docker

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/rizface/doui/internal/models"
)

// ListComposeProjects detects and returns all Docker Compose projects
func (c *Client) ListComposeProjects(ctx context.Context) ([]models.ComposeProject, error) {
	// List all containers with compose labels
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", "com.docker.compose.project")

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list compose containers: %w", err)
	}

	// Group containers by project name
	projectMap := make(map[string]*models.ComposeProject)

	for _, ctr := range containers {
		projectName := ctr.Labels["com.docker.compose.project"]
		serviceName := ctr.Labels["com.docker.compose.service"]
		configHash := ctr.Labels["com.docker.compose.config-hash"]
		workingDir := ctr.Labels["com.docker.compose.project.working_dir"]

		// Get or create project
		project, exists := projectMap[projectName]
		if !exists {
			project = &models.ComposeProject{
				Name:         projectName,
				Services:     []models.ComposeService{},
				ConfigHash:   configHash,
				WorkingDir:   workingDir,
				ContainerIDs: []string{},
			}
			projectMap[projectName] = project
		}

		// Add container ID to project
		project.ContainerIDs = append(project.ContainerIDs, ctr.ID)

		// Convert to models.Container
		name := ""
		if len(ctr.Names) > 0 {
			name = ctr.Names[0][1:] // Remove leading /
		}

		modelContainer := models.Container{
			ID:      ctr.ID,
			ShortID: ctr.ID[:12],
			Name:    name,
			Image:   ctr.Image,
			Status:  ctr.Status,
			State:   ctr.State,
			Created: time.Unix(ctr.Created, 0),
			Labels:  ctr.Labels,
		}

		// Find or create service
		var service *models.ComposeService
		for i := range project.Services {
			if project.Services[i].Name == serviceName {
				service = &project.Services[i]
				break
			}
		}

		if service == nil {
			project.Services = append(project.Services, models.ComposeService{
				Name:       serviceName,
				Containers: []models.Container{},
			})
			service = &project.Services[len(project.Services)-1]
		}

		// Add container to service
		service.Containers = append(service.Containers, modelContainer)
	}

	// Convert map to slice
	result := make([]models.ComposeProject, 0, len(projectMap))
	for _, project := range projectMap {
		result = append(result, *project)
	}

	// Sort projects alphabetically by name for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, nil
}

// StartComposeProject starts all containers in a compose project
func (c *Client) StartComposeProject(ctx context.Context, projectName string) error {
	// Find all containers for this project
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", fmt.Sprintf("com.docker.compose.project=%s", projectName))

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers for project %s: %w", projectName, err)
	}

	// Start all containers
	for _, ctr := range containers {
		if ctr.State != "running" {
			if err := c.cli.ContainerStart(ctx, ctr.ID, container.StartOptions{}); err != nil {
				return fmt.Errorf("failed to start container %s: %w", ctr.ID, err)
			}
		}
	}

	return nil
}

// StopComposeProject stops all containers in a compose project
func (c *Client) StopComposeProject(ctx context.Context, projectName string, timeout int) error {
	// Find all containers for this project
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", fmt.Sprintf("com.docker.compose.project=%s", projectName))

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers for project %s: %w", projectName, err)
	}

	stopTimeout := timeout
	// Stop all containers
	for _, ctr := range containers {
		if ctr.State == "running" {
			if err := c.cli.ContainerStop(ctx, ctr.ID, container.StopOptions{
				Timeout: &stopTimeout,
			}); err != nil {
				return fmt.Errorf("failed to stop container %s: %w", ctr.ID, err)
			}
		}
	}

	return nil
}

// RestartComposeProject restarts all containers in a compose project
func (c *Client) RestartComposeProject(ctx context.Context, projectName string, timeout int) error {
	// Find all containers for this project
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", fmt.Sprintf("com.docker.compose.project=%s", projectName))

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers for project %s: %w", projectName, err)
	}

	restartTimeout := timeout
	// Restart all containers
	for _, ctr := range containers {
		if err := c.cli.ContainerRestart(ctx, ctr.ID, container.StopOptions{
			Timeout: &restartTimeout,
		}); err != nil {
			return fmt.Errorf("failed to restart container %s: %w", ctr.ID, err)
		}
	}

	return nil
}
