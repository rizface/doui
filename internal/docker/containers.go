package docker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/rizface/doui/internal/models"
)

// ListContainers returns all containers (running and stopped)
func (c *Client) ListContainers(ctx context.Context, all bool) ([]models.Container, error) {
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{
		All: all,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	result := make([]models.Container, 0, len(containers))
	for _, ctr := range containers {
		// Extract container name (remove leading /)
		name := ""
		if len(ctr.Names) > 0 {
			name = strings.TrimPrefix(ctr.Names[0], "/")
		}

		// Convert ports
		ports := make([]models.PortMapping, 0, len(ctr.Ports))
		for _, port := range ctr.Ports {
			ports = append(ports, models.PortMapping{
				PrivatePort: int(port.PrivatePort),
				PublicPort:  int(port.PublicPort),
				Type:        port.Type,
				IP:          port.IP,
			})
		}

		// Extract networks
		networks := make([]string, 0, len(ctr.NetworkSettings.Networks))
		for name := range ctr.NetworkSettings.Networks {
			networks = append(networks, name)
		}

		// Extract mounts
		mounts := make([]models.MountPoint, 0, len(ctr.Mounts))
		for _, m := range ctr.Mounts {
			mounts = append(mounts, models.MountPoint{
				Type:        string(m.Type),
				Name:        m.Name,
				Source:      m.Source,
				Destination: m.Destination,
				ReadOnly:    !m.RW,
			})
		}

		result = append(result, models.Container{
			ID:         ctr.ID,
			ShortID:    ctr.ID[:12],
			Name:       name,
			Image:      ctr.Image,
			Status:     ctr.Status,
			State:      ctr.State,
			Created:    time.Unix(ctr.Created, 0),
			Ports:      ports,
			Networks:   networks,
			Mounts:     mounts,
			Labels:     ctr.Labels,
			SizeRw:     ctr.SizeRw,
			SizeRootFs: ctr.SizeRootFs,
		})
	}

	return result, nil
}

// StartContainer starts a container by ID
func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	err := c.cli.ContainerStart(ctx, containerID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container %s: %w", containerID, err)
	}
	return nil
}

// StopContainer stops a container by ID with a timeout
func (c *Client) StopContainer(ctx context.Context, containerID string, timeout int) error {
	stopTimeout := timeout
	err := c.cli.ContainerStop(ctx, containerID, container.StopOptions{
		Timeout: &stopTimeout,
	})
	if err != nil {
		return fmt.Errorf("failed to stop container %s: %w", containerID, err)
	}
	return nil
}

// RestartContainer restarts a container by ID with a timeout
func (c *Client) RestartContainer(ctx context.Context, containerID string, timeout int) error {
	restartTimeout := timeout
	err := c.cli.ContainerRestart(ctx, containerID, container.StopOptions{
		Timeout: &restartTimeout,
	})
	if err != nil {
		return fmt.Errorf("failed to restart container %s: %w", containerID, err)
	}
	return nil
}

// RemoveContainer removes a container by ID
func (c *Client) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	err := c.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force: force,
	})
	if err != nil {
		return fmt.Errorf("failed to remove container %s: %w", containerID, err)
	}
	return nil
}

// GetContainer gets detailed information about a container
func (c *Client) GetContainer(ctx context.Context, containerID string) (*models.Container, error) {
	inspect, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}

	// Extract ports
	ports := make([]models.PortMapping, 0)
	for port, bindings := range inspect.NetworkSettings.Ports {
		privatePort := port.Int()
		portType := port.Proto()

		if len(bindings) > 0 {
			for _, binding := range bindings {
				publicPort := 0
				if binding.HostPort != "" {
					fmt.Sscanf(binding.HostPort, "%d", &publicPort)
				}
				ports = append(ports, models.PortMapping{
					PrivatePort: privatePort,
					PublicPort:  publicPort,
					Type:        portType,
					IP:          binding.HostIP,
				})
			}
		} else {
			ports = append(ports, models.PortMapping{
				PrivatePort: privatePort,
				PublicPort:  0,
				Type:        portType,
				IP:          "",
			})
		}
	}

	// Extract networks
	networks := make([]string, 0, len(inspect.NetworkSettings.Networks))
	for name := range inspect.NetworkSettings.Networks {
		networks = append(networks, name)
	}

	// Parse created time
	created, err := time.Parse(time.RFC3339Nano, inspect.Created)
	if err != nil {
		created = time.Now()
	}

	return &models.Container{
		ID:       inspect.ID,
		ShortID:  inspect.ID[:12],
		Name:     strings.TrimPrefix(inspect.Name, "/"),
		Image:    inspect.Config.Image,
		Status:   inspect.State.Status,
		State:    inspect.State.Status,
		Created:  created,
		Ports:    ports,
		Networks: networks,
		Labels:   inspect.Config.Labels,
	}, nil
}

// InspectContainerFull returns the full container configuration needed for recreation
func (c *Client) InspectContainerFull(ctx context.Context, containerID string) (*models.ContainerFullConfig, error) {
	inspect, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	// Build the full config from inspection data
	config := &models.ContainerFullConfig{
		Name:       strings.TrimPrefix(inspect.Name, "/"),
		Image:      inspect.Config.Image,
		Env:        inspect.Config.Env,
		Cmd:        inspect.Config.Cmd,
		Entrypoint: inspect.Config.Entrypoint,
		WorkingDir: inspect.Config.WorkingDir,
		User:       inspect.Config.User,
		Labels:     inspect.Config.Labels,
	}

	// Host config
	if inspect.HostConfig != nil {
		config.Binds = inspect.HostConfig.Binds
		config.NetworkMode = string(inspect.HostConfig.NetworkMode)
		config.Privileged = inspect.HostConfig.Privileged
		config.CapAdd = inspect.HostConfig.CapAdd
		config.CapDrop = inspect.HostConfig.CapDrop
		config.RestartPolicy = models.ContainerRestartPolicy{
			Name:              string(inspect.HostConfig.RestartPolicy.Name),
			MaximumRetryCount: inspect.HostConfig.RestartPolicy.MaximumRetryCount,
		}

		// Convert port bindings
		config.PortBindings = make(map[string][]models.HostPortBinding)
		for port, bindings := range inspect.HostConfig.PortBindings {
			portKey := string(port)
			config.PortBindings[portKey] = make([]models.HostPortBinding, len(bindings))
			for i, b := range bindings {
				config.PortBindings[portKey][i] = models.HostPortBinding{
					HostIP:   b.HostIP,
					HostPort: b.HostPort,
				}
			}
		}
	}

	// Network config
	config.Networks = make(map[string]models.NetworkEndpointConfig)
	for netName, netConfig := range inspect.NetworkSettings.Networks {
		config.Networks[netName] = models.NetworkEndpointConfig{
			IPAddress: netConfig.IPAddress,
			Aliases:   netConfig.Aliases,
			NetworkID: netConfig.NetworkID,
		}
	}

	return config, nil
}

// RecreateContainer stops, removes, creates, and starts a container with new config
func (c *Client) RecreateContainer(ctx context.Context, containerID string, newConfig *models.ContainerFullConfig) (string, error) {
	// 1. Stop the container (if running) - ignore errors as container might already be stopped
	timeout := 10
	_ = c.cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})

	// 2. Remove the container
	if err := c.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}); err != nil {
		return "", fmt.Errorf("failed to remove old container: %w", err)
	}

	// 3. Build Docker SDK config from our model
	dockerConfig := &container.Config{
		Image:      newConfig.Image,
		Env:        newConfig.Env,
		Cmd:        newConfig.Cmd,
		Entrypoint: newConfig.Entrypoint,
		WorkingDir: newConfig.WorkingDir,
		User:       newConfig.User,
		Labels:     newConfig.Labels,
	}

	// Build exposed ports from port bindings
	dockerConfig.ExposedPorts = make(nat.PortSet)
	for port := range newConfig.PortBindings {
		dockerConfig.ExposedPorts[nat.Port(port)] = struct{}{}
	}

	// 4. Build host config
	hostConfig := &container.HostConfig{
		Binds:       newConfig.Binds,
		NetworkMode: container.NetworkMode(newConfig.NetworkMode),
		Privileged:  newConfig.Privileged,
		CapAdd:      newConfig.CapAdd,
		CapDrop:     newConfig.CapDrop,
		RestartPolicy: container.RestartPolicy{
			Name:              container.RestartPolicyMode(newConfig.RestartPolicy.Name),
			MaximumRetryCount: newConfig.RestartPolicy.MaximumRetryCount,
		},
	}

	// Convert port bindings
	hostConfig.PortBindings = make(nat.PortMap)
	for port, bindings := range newConfig.PortBindings {
		natPort := nat.Port(port)
		hostConfig.PortBindings[natPort] = make([]nat.PortBinding, len(bindings))
		for i, b := range bindings {
			hostConfig.PortBindings[natPort][i] = nat.PortBinding{
				HostIP:   b.HostIP,
				HostPort: b.HostPort,
			}
		}
	}

	// 5. Build network config (only for primary network at creation time)
	var networkConfig *network.NetworkingConfig
	var firstNetworkName string
	if len(newConfig.Networks) > 0 {
		networkConfig = &network.NetworkingConfig{
			EndpointsConfig: make(map[string]*network.EndpointSettings),
		}
		// Add first network at creation time
		for netName, netConfig := range newConfig.Networks {
			networkConfig.EndpointsConfig[netName] = &network.EndpointSettings{
				Aliases: netConfig.Aliases,
			}
			firstNetworkName = netName
			break // Only one network at creation time
		}
	}

	// 6. Create new container
	resp, err := c.cli.ContainerCreate(ctx, dockerConfig, hostConfig, networkConfig, nil, newConfig.Name)
	if err != nil {
		return "", fmt.Errorf("failed to create new container: %w", err)
	}

	// 7. Connect to additional networks
	for netName, netConfig := range newConfig.Networks {
		// Skip the first network (already connected at creation)
		if netName == firstNetworkName {
			continue
		}

		err := c.cli.NetworkConnect(ctx, netConfig.NetworkID, resp.ID, &network.EndpointSettings{
			Aliases: netConfig.Aliases,
		})
		if err != nil {
			// Log but don't fail - network might not exist anymore
			continue
		}
	}

	// 8. Start the container
	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return resp.ID, fmt.Errorf("container created but failed to start: %w", err)
	}

	return resp.ID, nil
}
