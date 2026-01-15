package docker

import (
	"context"
	"fmt"
	"sort"

	"github.com/docker/docker/api/types/network"
	"github.com/rizface/doui/internal/models"
)

// ListNetworks returns all Docker networks
func (c *Client) ListNetworks(ctx context.Context) ([]models.Network, error) {
	networks, err := c.cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	result := make([]models.Network, 0, len(networks))
	for _, net := range networks {
		// Extract container IDs
		containerIDs := make([]string, 0, len(net.Containers))
		for containerID := range net.Containers {
			containerIDs = append(containerIDs, containerID)
		}

		// Extract IPAM config
		ipam := models.NetworkIPAM{
			Driver: net.IPAM.Driver,
		}
		if len(net.IPAM.Config) > 0 {
			ipam.Subnet = net.IPAM.Config[0].Subnet
			ipam.Gateway = net.IPAM.Config[0].Gateway
		}

		result = append(result, models.Network{
			ID:         net.ID,
			Name:       net.Name,
			Driver:     net.Driver,
			Scope:      net.Scope,
			Internal:   net.Internal,
			Attachable: net.Attachable,
			Created:    net.Created,
			Containers: containerIDs,
			Labels:     net.Labels,
			IPAM:       ipam,
		})
	}

	// Sort networks alphabetically by name for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, nil
}

// GetNetwork returns detailed information about a specific network
func (c *Client) GetNetwork(ctx context.Context, networkID string) (*models.Network, error) {
	net, err := c.cli.NetworkInspect(ctx, networkID, network.InspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to inspect network %s: %w", networkID, err)
	}

	// Extract container IDs
	containerIDs := make([]string, 0, len(net.Containers))
	for containerID := range net.Containers {
		containerIDs = append(containerIDs, containerID)
	}

	// Extract IPAM config
	ipam := models.NetworkIPAM{
		Driver: net.IPAM.Driver,
	}
	if len(net.IPAM.Config) > 0 {
		ipam.Subnet = net.IPAM.Config[0].Subnet
		ipam.Gateway = net.IPAM.Config[0].Gateway
	}

	return &models.Network{
		ID:         net.ID,
		Name:       net.Name,
		Driver:     net.Driver,
		Scope:      net.Scope,
		Internal:   net.Internal,
		Attachable: net.Attachable,
		Created:    net.Created,
		Containers: containerIDs,
		Labels:     net.Labels,
		IPAM:       ipam,
	}, nil
}

// ConnectContainer connects a container to a network
func (c *Client) ConnectContainer(ctx context.Context, networkID, containerID string) error {
	err := c.cli.NetworkConnect(ctx, networkID, containerID, nil)
	if err != nil {
		return fmt.Errorf("failed to connect container %s to network %s: %w", containerID, networkID, err)
	}
	return nil
}

// DisconnectContainer disconnects a container from a network
func (c *Client) DisconnectContainer(ctx context.Context, networkID, containerID string, force bool) error {
	err := c.cli.NetworkDisconnect(ctx, networkID, containerID, force)
	if err != nil {
		return fmt.Errorf("failed to disconnect container %s from network %s: %w", containerID, networkID, err)
	}
	return nil
}

// CreateNetwork creates a new Docker network
func (c *Client) CreateNetwork(ctx context.Context, name, driver string) error {
	_, err := c.cli.NetworkCreate(ctx, name, network.CreateOptions{
		Driver: driver,
	})
	if err != nil {
		return fmt.Errorf("failed to create network %s: %w", name, err)
	}
	return nil
}

// RemoveNetwork removes a Docker network by ID
func (c *Client) RemoveNetwork(ctx context.Context, networkID string) error {
	err := c.cli.NetworkRemove(ctx, networkID)
	if err != nil {
		return fmt.Errorf("failed to remove network %s: %w", networkID, err)
	}
	return nil
}
