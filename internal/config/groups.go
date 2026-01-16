package config

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rizface/doui/internal/models"
)

// GroupManager manages container groups with persistence
type GroupManager struct {
	config *models.GroupConfig
	mu     sync.RWMutex
}

// NewGroupManager creates a new group manager and loads config from disk
func NewGroupManager() (*GroupManager, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return &GroupManager{
		config: config,
	}, nil
}

// GetAllGroups returns all groups
func (m *GroupManager) GetAllGroups() []models.Group {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modifications
	groups := make([]models.Group, len(m.config.Groups))
	copy(groups, m.config.Groups)
	return groups
}

// GetGroup returns a group by ID
func (m *GroupManager) GetGroup(id string) *models.Group {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.config.FindGroup(id)
}

// CreateGroup creates a new group
func (m *GroupManager) CreateGroup(name, description string, containerIDs []string) (*models.Group, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	group := models.Group{
		ID:           uuid.New().String(),
		Name:         name,
		Description:  description,
		ContainerIDs: containerIDs,
		Created:      time.Now(),
		Modified:     time.Now(),
		Color:        selectColor(len(m.config.Groups)),
	}

	m.config.AddGroup(group)

	if err := m.save(); err != nil {
		return nil, err
	}

	return &group, nil
}

// UpdateGroup updates an existing group
func (m *GroupManager) UpdateGroup(group models.Group) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.config.UpdateGroup(group) {
		return fmt.Errorf("group not found: %s", group.ID)
	}

	return m.save()
}

// DeleteGroup deletes a group by ID
func (m *GroupManager) DeleteGroup(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.config.RemoveGroup(id) {
		return fmt.Errorf("group not found: %s", id)
	}

	return m.save()
}

// AddContainerToGroup adds a container to a group
func (m *GroupManager) AddContainerToGroup(groupID, containerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	group := m.config.FindGroup(groupID)
	if group == nil {
		return fmt.Errorf("group not found: %s", groupID)
	}

	// Check if container already in group
	for _, id := range group.ContainerIDs {
		if id == containerID {
			return nil // Already in group
		}
	}

	group.ContainerIDs = append(group.ContainerIDs, containerID)
	group.Modified = time.Now()

	if !m.config.UpdateGroup(*group) {
		return fmt.Errorf("failed to update group")
	}

	return m.save()
}

// RemoveContainerFromGroup removes a container from a group
func (m *GroupManager) RemoveContainerFromGroup(groupID, containerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	group := m.config.FindGroup(groupID)
	if group == nil {
		return fmt.Errorf("group not found: %s", groupID)
	}

	// Remove container from group
	newContainerIDs := make([]string, 0, len(group.ContainerIDs))
	for _, id := range group.ContainerIDs {
		if id != containerID {
			newContainerIDs = append(newContainerIDs, id)
		}
	}

	group.ContainerIDs = newContainerIDs
	group.Modified = time.Now()

	if !m.config.UpdateGroup(*group) {
		return fmt.Errorf("failed to update group")
	}

	return m.save()
}

// ReplaceContainerID replaces oldID with newID in all groups
// This is used when a container is recreated (e.g., after env var changes)
func (m *GroupManager) ReplaceContainerID(oldID, newID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	modified := false
	for i := range m.config.Groups {
		group := &m.config.Groups[i]
		for j, id := range group.ContainerIDs {
			if id == oldID {
				group.ContainerIDs[j] = newID
				group.Modified = time.Now()
				modified = true
				break // Container can only be in the group once
			}
		}
	}

	if modified {
		return m.save()
	}
	return nil
}

// RemoveContainerFromAllGroups removes a container ID from all groups
// This is used when a container is deleted
func (m *GroupManager) RemoveContainerFromAllGroups(containerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	modified := false
	for i := range m.config.Groups {
		group := &m.config.Groups[i]
		newContainerIDs := make([]string, 0, len(group.ContainerIDs))
		for _, id := range group.ContainerIDs {
			if id != containerID {
				newContainerIDs = append(newContainerIDs, id)
			} else {
				modified = true
			}
		}
		if len(newContainerIDs) != len(group.ContainerIDs) {
			group.ContainerIDs = newContainerIDs
			group.Modified = time.Now()
		}
	}

	if modified {
		return m.save()
	}
	return nil
}

// save persists the config to disk (caller must hold lock)
func (m *GroupManager) save() error {
	m.config.LastModified = time.Now()
	return SaveConfig(m.config)
}

// StartGroup starts all containers in a group
type ContainerOperation func(context.Context, string) error

func (m *GroupManager) ExecuteGroupOperation(ctx context.Context, groupID string, operation ContainerOperation) error {
	group := m.GetGroup(groupID)
	if group == nil {
		return fmt.Errorf("group not found: %s", groupID)
	}

	type result struct {
		containerID string
		err         error
	}

	results := make(chan result, len(group.ContainerIDs))
	var wg sync.WaitGroup

	// Execute operations in parallel
	for _, containerID := range group.ContainerIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			err := operation(ctx, id)
			results <- result{containerID: id, err: err}
		}(containerID)
	}

	wg.Wait()
	close(results)

	// Collect errors
	var errs []error
	for r := range results {
		if r.err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", r.containerID[:12], r.err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("group operation failed: %v", errs)
	}

	return nil
}

// selectColor selects a color for a new group based on index
func selectColor(index int) string {
	colors := []string{"blue", "green", "yellow", "magenta", "cyan", "red"}
	return colors[index%len(colors)]
}
