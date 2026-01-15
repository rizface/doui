package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/client"
)

// Client wraps the Docker SDK client
type Client struct {
	cli *client.Client
}

// NewClient creates a new Docker client with connectivity verification
func NewClient() (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	// Verify connectivity
	_, err = cli.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("docker daemon not reachable: %w", err)
	}

	return &Client{cli: cli}, nil
}

// Close closes the Docker client connection
func (c *Client) Close() error {
	if c.cli != nil {
		return c.cli.Close()
	}
	return nil
}

// GetRawClient returns the underlying Docker client for advanced usage
func (c *Client) GetRawClient() *client.Client {
	return c.cli
}
