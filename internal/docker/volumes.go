package docker

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/rizface/doui/internal/models"
)

// ListVolumes returns all Docker volumes
func (c *Client) ListVolumes(ctx context.Context) ([]models.Volume, error) {
	volumesResponse, err := c.cli.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}

	result := make([]models.Volume, 0, len(volumesResponse.Volumes))
	for _, vol := range volumesResponse.Volumes {
		// Parse created time
		created := time.Now()
		if vol.CreatedAt != "" {
			if parsedTime, err := time.Parse(time.RFC3339, vol.CreatedAt); err == nil {
				created = parsedTime
			}
		}

		// Get usage data if available
		var usageData *models.VolumeUsageData
		if vol.UsageData != nil {
			usageData = &models.VolumeUsageData{
				RefCount: int(vol.UsageData.RefCount),
				Size:     vol.UsageData.Size,
			}
		}

		result = append(result, models.Volume{
			Name:       vol.Name,
			Driver:     vol.Driver,
			Mountpoint: vol.Mountpoint,
			Created:    created,
			Labels:     vol.Labels,
			Scope:      vol.Scope,
			Options:    vol.Options,
			UsageData:  usageData,
		})
	}

	// Sort volumes alphabetically by name for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, nil
}

// RemoveVolume removes a volume by name
func (c *Client) RemoveVolume(ctx context.Context, volumeName string, force bool) error {
	err := c.cli.VolumeRemove(ctx, volumeName, force)
	if err != nil {
		return fmt.Errorf("failed to remove volume %s: %w", volumeName, err)
	}
	return nil
}

// PruneUnusedVolumes removes all unused volumes
func (c *Client) PruneUnusedVolumes(ctx context.Context) (uint64, error) {
	report, err := c.cli.VolumesPrune(ctx, filters.Args{})
	if err != nil {
		return 0, fmt.Errorf("failed to prune volumes: %w", err)
	}
	return report.SpaceReclaimed, nil
}
