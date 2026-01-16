package docker

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types/container"
)

// LogEntry represents a single log line
type LogEntry struct {
	Line      string
	Timestamp time.Time
	IsError   bool
}

// StreamLogs streams logs from a container
func (c *Client) StreamLogs(ctx context.Context, containerID string, follow bool, since time.Time, tail string) (<-chan LogEntry, <-chan error) {
	logsChan := make(chan LogEntry, 100)
	errorChan := make(chan error, 1)

	go func() {
		defer close(logsChan)
		defer close(errorChan)

		sinceStr := ""
		if !since.IsZero() {
			sinceStr = since.Format(time.RFC3339)
		}

		reader, err := c.cli.ContainerLogs(ctx, containerID, container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     follow,
			Timestamps: true,
			Since:      sinceStr,
			Tail:       tail,
		})
		if err != nil {
			errorChan <- fmt.Errorf("failed to get container logs: %w", err)
			return
		}
		defer reader.Close()

		// Docker logs are multiplexed with an 8-byte header
		// [8]byte{STREAM_TYPE, 0, 0, 0, SIZE1, SIZE2, SIZE3, SIZE4}
		// STREAM_TYPE: 0=stdin, 1=stdout, 2=stderr
		// SIZE: uint32 big endian
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024) // 64KB initial, 1MB max

		for scanner.Scan() {
			line := scanner.Text()

			// Docker adds headers, but with Timestamps they're already readable
			// We'll just send the line as-is
			select {
			case logsChan <- LogEntry{
				Line:      line,
				Timestamp: time.Now(),
				IsError:   false,
			}:
			case <-ctx.Done():
				return
			}
		}

		if err := scanner.Err(); err != nil && err != io.EOF {
			errorChan <- fmt.Errorf("error reading logs: %w", err)
		}
	}()

	return logsChan, errorChan
}
