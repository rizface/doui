package models

import (
	"fmt"
	"strings"
	"time"
)

// Container represents a Docker container with UI-relevant fields
type Container struct {
	ID         string
	ShortID    string // First 12 chars
	Name       string
	Image      string
	Status     string
	State      string // running, paused, exited, etc.
	Created    time.Time
	Ports      []PortMapping
	Networks   []string
	Mounts     []MountPoint // Volume/bind mounts
	Labels     map[string]string
	SizeRw     int64
	SizeRootFs int64
}

// MountPoint represents a container mount (volume or bind)
type MountPoint struct {
	Type        string // "volume", "bind", "tmpfs"
	Name        string // Volume name (empty for bind mounts)
	Source      string // Source path on host
	Destination string // Mount path in container
	ReadOnly    bool
}

// PortMapping represents a container port mapping
type PortMapping struct {
	PrivatePort int
	PublicPort  int
	Type        string // tcp, udp
	IP          string
}

// ContainerStats represents runtime statistics
type ContainerStats struct {
	ContainerID   string
	CPUPercent    float64
	MemoryUsage   uint64
	MemoryLimit   uint64
	MemoryPercent float64
	NetworkRx     uint64
	NetworkTx     uint64
	BlockRead     uint64
	BlockWrite    uint64
	PIDs          uint64
	Timestamp     time.Time
}

// ShortID returns the first 12 characters of the container ID
func (c *Container) GetShortID() string {
	if len(c.ID) >= 12 {
		return c.ID[:12]
	}
	return c.ID
}

// IsRunning returns true if the container is currently running
func (c *Container) IsRunning() bool {
	return c.State == "running"
}

// GetPortsString returns a formatted string of port mappings
func (c *Container) GetPortsString() string {
	if len(c.Ports) == 0 {
		return ""
	}

	result := ""
	for i, port := range c.Ports {
		if i > 0 {
			result += ", "
		}
		if port.PublicPort > 0 {
			result += fmt.Sprintf("%d:%d/%s", port.PublicPort, port.PrivatePort, port.Type)
		} else {
			result += fmt.Sprintf("%d/%s", port.PrivatePort, port.Type)
		}
	}
	return result
}

// ContainerFullConfig holds all configuration needed to recreate a container
type ContainerFullConfig struct {
	// Basic Info
	Name  string
	Image string

	// Environment
	Env []string // ["KEY=value", ...]

	// Host Config
	Binds         []string // Volume binds ["/host/path:/container/path:ro", ...]
	PortBindings  map[string][]HostPortBinding
	RestartPolicy ContainerRestartPolicy
	NetworkMode   string
	Privileged    bool
	CapAdd        []string
	CapDrop       []string

	// Network Config
	Networks map[string]NetworkEndpointConfig

	// Other
	Cmd        []string
	Entrypoint []string
	WorkingDir string
	User       string
	Labels     map[string]string
}

// HostPortBinding represents a port binding to the host
type HostPortBinding struct {
	HostIP   string
	HostPort string
}

// ContainerRestartPolicy represents the restart policy for a container
type ContainerRestartPolicy struct {
	Name              string
	MaximumRetryCount int
}

// NetworkEndpointConfig represents network endpoint configuration
type NetworkEndpointConfig struct {
	IPAddress string
	Aliases   []string
	NetworkID string
}

// EnvVar represents a parsed environment variable for display/editing
type EnvVar struct {
	Key   string
	Value string
}

// ParseEnvVars converts []string env (KEY=value format) to []EnvVar
func ParseEnvVars(env []string) []EnvVar {
	result := make([]EnvVar, 0, len(env))
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			result = append(result, EnvVar{Key: parts[0], Value: parts[1]})
		} else if len(parts) == 1 && parts[0] != "" {
			// Handle env vars with no value
			result = append(result, EnvVar{Key: parts[0], Value: ""})
		}
	}
	return result
}

// EnvVarsToStrings converts []EnvVar back to []string (KEY=value format)
func EnvVarsToStrings(vars []EnvVar) []string {
	result := make([]string, len(vars))
	for i, v := range vars {
		result[i] = v.Key + "=" + v.Value
	}
	return result
}
