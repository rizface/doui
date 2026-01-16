package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
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

	// Sort images: tagged first (alphabetically), then dangling (by created date, newest first)
	sort.Slice(result, func(i, j int) bool {
		iDangling := result[i].IsDangling()
		jDangling := result[j].IsDangling()

		// Tagged images come before dangling
		if iDangling != jDangling {
			return !iDangling
		}

		// Both tagged: sort by tag name
		if !iDangling {
			return result[i].GetPrimaryTag() < result[j].GetPrimaryTag()
		}

		// Both dangling: sort by created date (newest first)
		return result[i].Created.After(result[j].Created)
	})

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

// PullProgress represents progress of an image pull operation
type PullProgress struct {
	Status   string
	Progress string // Progress bar string from Docker
	Current  int64
	Total    int64
	Done     bool
	Error    error
}

// pullEvent represents a single event from Docker's image pull stream
type pullEvent struct {
	Status         string `json:"status"`
	Progress       string `json:"progress"`
	ProgressDetail struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"progressDetail"`
	ID    string `json:"id"`
	Error string `json:"error"`
}

// PullImageWithProgress pulls an image and streams progress updates
func (c *Client) PullImageWithProgress(ctx context.Context, imageName string) <-chan PullProgress {
	progressChan := make(chan PullProgress)

	go func() {
		defer close(progressChan)

		out, err := c.cli.ImagePull(ctx, imageName, image.PullOptions{})
		if err != nil {
			progressChan <- PullProgress{Error: fmt.Errorf("failed to pull image %s: %w", imageName, err), Done: true}
			return
		}
		defer out.Close()

		// Track progress per layer
		layerProgress := make(map[string]pullEvent)
		scanner := bufio.NewScanner(out)

		for scanner.Scan() {
			var event pullEvent
			if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
				continue
			}

			if event.Error != "" {
				progressChan <- PullProgress{Error: fmt.Errorf("%s", event.Error), Done: true}
				return
			}

			// Track layer progress
			if event.ID != "" {
				layerProgress[event.ID] = event
			}

			// Calculate total progress across all layers
			var totalCurrent, totalTotal int64
			for _, layer := range layerProgress {
				totalCurrent += layer.ProgressDetail.Current
				totalTotal += layer.ProgressDetail.Total
			}

			progress := PullProgress{
				Status:  event.Status,
				Current: totalCurrent,
				Total:   totalTotal,
			}

			// Build progress string
			if event.Progress != "" {
				progress.Progress = event.Progress
			}

			progressChan <- progress
		}

		if err := scanner.Err(); err != nil {
			progressChan <- PullProgress{Error: fmt.Errorf("failed to read pull output: %w", err), Done: true}
			return
		}

		progressChan <- PullProgress{Status: "Pull complete", Done: true}
	}()

	return progressChan
}

// PruneImages removes all dangling images
func (c *Client) PruneImages(ctx context.Context) (int, int64, error) {
	report, err := c.cli.ImagesPrune(ctx, filters.NewArgs())
	if err != nil {
		return 0, 0, fmt.Errorf("failed to prune images: %w", err)
	}

	return len(report.ImagesDeleted), int64(report.SpaceReclaimed), nil
}
