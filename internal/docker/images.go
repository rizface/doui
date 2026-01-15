package docker

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/rizface/doui/internal/models"
)

// ListImages returns all images
func (c *Client) ListImages(ctx context.Context) ([]models.Image, error) {
	images, err := c.cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	// Count containers for each image
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		// Non-fatal, just log and continue without container counts
		containers = nil
	}

	imageContainerCount := make(map[string]int)
	for _, ctr := range containers {
		imageContainerCount[ctr.ImageID]++
	}

	result := make([]models.Image, 0, len(images))
	for _, img := range images {
		shortID := img.ID
		if len(shortID) > 12 {
			shortID = shortID[7:19] // Remove sha256: prefix and get 12 chars
		}

		result = append(result, models.Image{
			ID:          img.ID,
			ShortID:     shortID,
			RepoTags:    img.RepoTags,
			RepoDigests: img.RepoDigests,
			Created:     time.Unix(img.Created, 0),
			Size:        img.Size,
			VirtualSize: img.VirtualSize,
			Labels:      img.Labels,
			Containers:  imageContainerCount[img.ID],
		})
	}

	return result, nil
}

// RemoveImage removes an image by ID
func (c *Client) RemoveImage(ctx context.Context, imageID string, force bool) error {
	_, err := c.cli.ImageRemove(ctx, imageID, image.RemoveOptions{
		Force: force,
	})
	if err != nil {
		return fmt.Errorf("failed to remove image %s: %w", imageID, err)
	}
	return nil
}

// GetImage gets detailed information about an image
func (c *Client) GetImage(ctx context.Context, imageID string) (*models.Image, error) {
	inspect, _, err := c.cli.ImageInspectWithRaw(ctx, imageID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect image %s: %w", imageID, err)
	}

	// Parse created time
	created, err := time.Parse(time.RFC3339Nano, inspect.Created)
	if err != nil {
		created = time.Now()
	}

	shortID := inspect.ID
	if len(shortID) > 12 {
		shortID = shortID[7:19] // Remove sha256: prefix and get 12 chars
	}

	return &models.Image{
		ID:          inspect.ID,
		ShortID:     shortID,
		RepoTags:    inspect.RepoTags,
		RepoDigests: inspect.RepoDigests,
		Created:     created,
		Size:        inspect.Size,
		VirtualSize: inspect.VirtualSize,
		Labels:      inspect.Config.Labels,
	}, nil
}

// PullImage pulls an image from a registry
func (c *Client) PullImage(ctx context.Context, imageName string) error {
	out, err := c.cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer out.Close()

	// Read all the output to ensure pull completes
	_, err = io.Copy(io.Discard, out)
	if err != nil {
		return fmt.Errorf("failed to read pull output: %w", err)
	}

	return nil
}
