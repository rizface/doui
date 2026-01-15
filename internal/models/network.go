package models

import (
	"time"
)

// Network represents a Docker network with UI-relevant fields
type Network struct {
	ID         string
	Name       string
	Driver     string // bridge, host, overlay, macvlan, etc.
	Scope      string // local, swarm, global
	Internal   bool
	Attachable bool
	Created    time.Time
	Containers []string          // Container IDs attached to this network
	Labels     map[string]string
	IPAM       NetworkIPAM
}

// NetworkIPAM represents IPAM configuration for a network
type NetworkIPAM struct {
	Driver  string
	Subnet  string
	Gateway string
}

// GetShortID returns the first 12 characters of the network ID
func (n *Network) GetShortID() string {
	if len(n.ID) >= 12 {
		return n.ID[:12]
	}
	return n.ID
}

// GetContainerCount returns the number of containers attached to this network
func (n *Network) GetContainerCount() int {
	return len(n.Containers)
}

// IsSystemNetwork returns true if this is a default Docker system network
func (n *Network) IsSystemNetwork() bool {
	return n.Name == "bridge" || n.Name == "host" || n.Name == "none"
}
