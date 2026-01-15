package models

import "time"

// Group represents a collection of containers
type Group struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	ContainerIDs []string  `json:"container_ids"`
	Created      time.Time `json:"created"`
	Modified     time.Time `json:"modified"`
	Color        string    `json:"color"`
}

// GroupConfig represents the persisted configuration
type GroupConfig struct {
	Version      string    `json:"version"`
	Groups       []Group   `json:"groups"`
	LastModified time.Time `json:"last_modified"`
}

// NewGroupConfig creates a new empty group configuration
func NewGroupConfig() *GroupConfig {
	return &GroupConfig{
		Version:      "1.0",
		Groups:       []Group{},
		LastModified: time.Now(),
	}
}

// FindGroup finds a group by ID, returns nil if not found
func (gc *GroupConfig) FindGroup(id string) *Group {
	for i := range gc.Groups {
		if gc.Groups[i].ID == id {
			return &gc.Groups[i]
		}
	}
	return nil
}

// AddGroup adds a new group to the configuration
func (gc *GroupConfig) AddGroup(group Group) {
	gc.Groups = append(gc.Groups, group)
	gc.LastModified = time.Now()
}

// RemoveGroup removes a group by ID
func (gc *GroupConfig) RemoveGroup(id string) bool {
	for i, group := range gc.Groups {
		if group.ID == id {
			gc.Groups = append(gc.Groups[:i], gc.Groups[i+1:]...)
			gc.LastModified = time.Now()
			return true
		}
	}
	return false
}

// UpdateGroup updates an existing group
func (gc *GroupConfig) UpdateGroup(updated Group) bool {
	for i := range gc.Groups {
		if gc.Groups[i].ID == updated.ID {
			updated.Modified = time.Now()
			gc.Groups[i] = updated
			gc.LastModified = time.Now()
			return true
		}
	}
	return false
}
