package models

import "time"

// Image represents a Docker image
type Image struct {
	ID           string
	ShortID      string
	RepoTags     []string
	RepoDigests  []string
	Created      time.Time
	Size         int64
	VirtualSize  int64
	Labels       map[string]string
	Containers   int // Number of containers using this image
}

// GetShortID returns the first 12 characters of the image ID
func (i *Image) GetShortID() string {
	if len(i.ID) >= 12 {
		return i.ID[:12]
	}
	return i.ID
}

// GetPrimaryTag returns the first tag or "<none>" if no tags exist
func (i *Image) GetPrimaryTag() string {
	if len(i.RepoTags) > 0 {
		return i.RepoTags[0]
	}
	return "<none>"
}

// GetRepository returns the repository part of the primary tag
func (i *Image) GetRepository() string {
	tag := i.GetPrimaryTag()
	if tag == "<none>" {
		return "<none>"
	}

	// Split by ':' to get repository
	for idx, ch := range tag {
		if ch == ':' {
			return tag[:idx]
		}
	}
	return tag
}

// GetTag returns the tag part of the primary tag
func (i *Image) GetTag() string {
	tag := i.GetPrimaryTag()
	if tag == "<none>" {
		return "<none>"
	}

	// Split by ':' to get tag
	for idx, ch := range tag {
		if ch == ':' {
			return tag[idx+1:]
		}
	}
	return "latest"
}

// IsDangling returns true if the image has no tags (untagged image)
func (i *Image) IsDangling() bool {
	if len(i.RepoTags) == 0 {
		return true
	}
	// Check if all tags are <none>:<none>
	for _, tag := range i.RepoTags {
		if tag != "<none>:<none>" {
			return false
		}
	}
	return true
}

// IsUnused returns true if the image is not used by any container
func (i *Image) IsUnused() bool {
	return i.Containers == 0
}
