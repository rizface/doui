package app

import (
	"github.com/rizface/doui/internal/config"
	"github.com/rizface/doui/internal/docker"
	"github.com/rizface/doui/internal/models"
)

// Message types for bubbletea

// DockerClientReadyMsg is sent when Docker client is initialized
type DockerClientReadyMsg struct {
	client *docker.Client
}

// GroupManagerReadyMsg is sent when GroupManager is initialized
type GroupManagerReadyMsg struct {
	manager *config.GroupManager
}

// GroupsLoadedMsg is sent when groups are loaded
type GroupsLoadedMsg struct {
	groups []models.Group
}

// ContainersLoadedMsg is sent when containers are loaded
type ContainersLoadedMsg struct {
	containers []models.Container
}

// ImagesLoadedMsg is sent when images are loaded
type ImagesLoadedMsg struct {
	images []models.Image
}

// Container operation messages
type ContainerStartedMsg struct {
	containerID string
	err         error
}

type ContainerStoppedMsg struct {
	containerID string
	err         error
}

type ContainerRestartedMsg struct {
	containerID string
	err         error
}

type ContainerRemovedMsg struct {
	containerID string
	err         error
}

// Image operation messages
type ImageRemovedMsg struct {
	imageID string
	err     error
}

type ImagesBulkRemovedMsg struct {
	count   int
	failed  int
	err     error
}

type ImagesPrunedMsg struct {
	count       int
	spaceFreed  int64
	err         error
}

// Group operation messages
type GroupStartedMsg struct {
	groupID string
	err     error
}

type GroupStoppedMsg struct {
	groupID string
	err     error
}

type GroupCreatedMsg struct {
	name string
}

// Container added to group
type ContainerAddedToGroupMsg struct {
	groupID     string
	containerID string
	err         error
}

// Container removed from group
type ContainerRemovedFromGroupMsg struct {
	groupID     string
	containerID string
	err         error
}

// Container ID replaced in groups (after container recreate)
type ContainerIDReplacedMsg struct {
	oldID string
	newID string
	err   error
}

// Container removed from all groups (after container delete)
type ContainerRemovedFromAllGroupsMsg struct {
	containerID string
	err         error
}

// UI messages
type RefreshTickMsg struct{}

type ErrorMsg struct {
	err error
}

type StatusMsg struct {
	message string
}

type ClearStatusMsg struct{}

// Volume operation messages
type VolumesLoadedMsg struct {
	volumes []models.Volume
}

type VolumeRemovedMsg struct {
	volumeName string
	err        error
}

// Compose operation messages
type ComposeProjectsLoadedMsg struct {
	projects []models.ComposeProject
}

type ComposeProjectStartedMsg struct {
	projectName string
	err         error
}

type ComposeProjectStoppedMsg struct {
	projectName string
	err         error
}

type ComposeProjectRestartedMsg struct {
	projectName string
	err         error
}

// Image pull messages
type ImagePullProgressMsg struct {
	imageName string
	status    string
	progress  string
	current   int64
	total     int64
	done      bool
	err       error
}

type ImagePullCompletedMsg struct {
	imageName string
	err       error
}

// Network operation messages
type NetworksLoadedMsg struct {
	networks []models.Network
}

type ContainerConnectedToNetworkMsg struct {
	networkID   string
	containerID string
	err         error
}

type ContainerDisconnectedFromNetworkMsg struct {
	networkID   string
	containerID string
	err         error
}

type NetworkCreatedMsg struct {
	name string
	err  error
}

type NetworkRemovedMsg struct {
	networkID string
	err       error
}

// Container configuration messages (for env var editing)
type ContainerConfigLoadedMsg struct {
	containerID string
	config      *models.ContainerFullConfig
	err         error
}

type ContainerRecreatedMsg struct {
	oldID         string
	newID         string
	containerName string
	err           error
}
