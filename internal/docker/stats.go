package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/rizface/doui/internal/models"
)

// StreamStats streams container statistics
func (c *Client) StreamStats(ctx context.Context, containerID string) (<-chan *models.ContainerStats, <-chan error) {
	statsChan := make(chan *models.ContainerStats, 10)
	errorChan := make(chan error, 1)

	go func() {
		defer close(statsChan)
		defer close(errorChan)

		stats, err := c.cli.ContainerStats(ctx, containerID, true) // stream=true
		if err != nil {
			errorChan <- fmt.Errorf("failed to get container stats: %w", err)
			return
		}
		defer stats.Body.Close()

		decoder := json.NewDecoder(stats.Body)
		var prevCPU, prevSystem uint64

		for {
			var v types.StatsJSON
			if err := decoder.Decode(&v); err != nil {
				if ctx.Err() != nil {
					// Context cancelled, normal exit
					return
				}
				errorChan <- fmt.Errorf("error decoding stats: %w", err)
				return
			}

			// Calculate CPU percentage
			cpuPercent := calculateCPUPercent(prevCPU, prevSystem, &v)
			prevCPU = v.CPUStats.CPUUsage.TotalUsage
			prevSystem = v.CPUStats.SystemUsage

			// Calculate memory percentage
			var memPercent float64
			if v.MemoryStats.Limit > 0 {
				memPercent = float64(v.MemoryStats.Usage) / float64(v.MemoryStats.Limit) * 100.0
			}

			// Calculate network I/O
			var networkRx, networkTx uint64
			for _, netStats := range v.Networks {
				networkRx += netStats.RxBytes
				networkTx += netStats.TxBytes
			}

			// Calculate block I/O
			var blockRead, blockWrite uint64
			for _, bioEntry := range v.BlkioStats.IoServiceBytesRecursive {
				if bioEntry.Op == "read" || bioEntry.Op == "Read" {
					blockRead += bioEntry.Value
				} else if bioEntry.Op == "write" || bioEntry.Op == "Write" {
					blockWrite += bioEntry.Value
				}
			}

			select {
			case statsChan <- &models.ContainerStats{
				ContainerID:   containerID,
				CPUPercent:    cpuPercent,
				MemoryUsage:   v.MemoryStats.Usage,
				MemoryLimit:   v.MemoryStats.Limit,
				MemoryPercent: memPercent,
				NetworkRx:     networkRx,
				NetworkTx:     networkTx,
				BlockRead:     blockRead,
				BlockWrite:    blockWrite,
				PIDs:          v.PidsStats.Current,
				Timestamp:     time.Now(),
			}:
			case <-ctx.Done():
				return
			}
		}
	}()

	return statsChan, errorChan
}

// calculateCPUPercent calculates CPU usage percentage
// Docker requires two samples to calculate CPU percentage
func calculateCPUPercent(previousCPU, previousSystem uint64, stats *types.StatsJSON) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - previousCPU)
	systemDelta := float64(stats.CPUStats.SystemUsage - previousSystem)

	if systemDelta > 0 && cpuDelta > 0 {
		cpuCount := float64(stats.CPUStats.OnlineCPUs)
		if cpuCount == 0 {
			cpuCount = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
		}
		if cpuCount == 0 {
			cpuCount = 1
		}
		return (cpuDelta / systemDelta) * cpuCount * 100.0
	}
	return 0.0
}

// GetStats fetches a single stats snapshot (non-streaming)
func (c *Client) GetStats(ctx context.Context, containerID string) (*models.ContainerStats, error) {
	stats, err := c.cli.ContainerStats(ctx, containerID, false) // stream=false
	if err != nil {
		return nil, fmt.Errorf("failed to get container stats: %w", err)
	}
	defer stats.Body.Close()

	var v types.StatsJSON
	if err := json.NewDecoder(stats.Body).Decode(&v); err != nil {
		return nil, fmt.Errorf("error decoding stats: %w", err)
	}

	// Calculate memory percentage
	var memPercent float64
	if v.MemoryStats.Limit > 0 {
		memPercent = float64(v.MemoryStats.Usage) / float64(v.MemoryStats.Limit) * 100.0
	}

	// Calculate network I/O
	var networkRx, networkTx uint64
	for _, netStats := range v.Networks {
		networkRx += netStats.RxBytes
		networkTx += netStats.TxBytes
	}

	// Calculate block I/O
	var blockRead, blockWrite uint64
	for _, bioEntry := range v.BlkioStats.IoServiceBytesRecursive {
		if bioEntry.Op == "read" || bioEntry.Op == "Read" {
			blockRead += bioEntry.Value
		} else if bioEntry.Op == "write" || bioEntry.Op == "Write" {
			blockWrite += bioEntry.Value
		}
	}

	return &models.ContainerStats{
		ContainerID:   containerID,
		CPUPercent:    0, // Single sample, can't calculate
		MemoryUsage:   v.MemoryStats.Usage,
		MemoryLimit:   v.MemoryStats.Limit,
		MemoryPercent: memPercent,
		NetworkRx:     networkRx,
		NetworkTx:     networkTx,
		BlockRead:     blockRead,
		BlockWrite:    blockWrite,
		PIDs:          v.PidsStats.Current,
		Timestamp:     time.Now(),
	}, nil
}
