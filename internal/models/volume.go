package models

import "time"

// Volume represents a Docker volume
type Volume struct {
	Name       string
	Driver     string
	Mountpoint string
	Created    time.Time
	Labels     map[string]string
	Scope      string
	Options    map[string]string
	UsageData  *VolumeUsageData
}

// VolumeUsageData represents volume usage statistics
type VolumeUsageData struct {
	RefCount int   // Number of containers using this volume
	Size     int64 // Size in bytes (-1 if unavailable)
}

// GetShortName returns the volume name truncated if too long
func (v *Volume) GetShortName() string {
	if len(v.Name) > 40 {
		return v.Name[:37] + "..."
	}
	return v.Name
}

// IsInUse returns true if the volume is being used by containers
func (v *Volume) IsInUse() bool {
	return v.UsageData != nil && v.UsageData.RefCount > 0
}

// GetDriver returns the driver name or "local" as default
func (v *Volume) GetDriver() string {
	if v.Driver == "" {
		return "local"
	}
	return v.Driver
}
