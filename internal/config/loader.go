package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rizface/doui/internal/models"
)

// LoadConfig loads the configuration from disk
func LoadConfig() (*models.GroupConfig, error) {
	configPath, err := GetConfigFilePath()
	if err != nil {
		return nil, err
	}

	// If config file doesn't exist, return new empty config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return models.NewGroupConfig(), nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	var config models.GroupConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveConfig saves the configuration to disk using atomic write
func SaveConfig(config *models.GroupConfig) error {
	configPath, err := GetConfigFilePath()
	if err != nil {
		return err
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to temporary file first
	tmpFile := configPath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp config file: %w", err)
	}

	// Backup current config if it exists
	if _, err := os.Stat(configPath); err == nil {
		backupPath := configPath + ".bak"
		if err := os.Rename(configPath, backupPath); err != nil {
			// Non-fatal, just log and continue
			_ = err
		}
	}

	// Atomic rename
	if err := os.Rename(tmpFile, configPath); err != nil {
		return fmt.Errorf("failed to rename temp config file: %w", err)
	}

	return nil
}
