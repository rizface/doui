package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetConfigDir returns the configuration directory path
// Priority: DOUI_CONFIG_PATH > $HOME/.config/doui > $HOME/.doui
func GetConfigDir() (string, error) {
	// Check environment variable first
	if configPath := os.Getenv("DOUI_CONFIG_PATH"); configPath != "" {
		return configPath, nil
	}

	// Get home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Try XDG config directory first
	configDir := filepath.Join(home, ".config", "doui")
	if _, err := os.Stat(configDir); err == nil {
		return configDir, nil
	}

	// Check if legacy directory exists
	legacyDir := filepath.Join(home, ".doui")
	if _, err := os.Stat(legacyDir); err == nil {
		return legacyDir, nil
	}

	// Default to XDG config directory
	return configDir, nil
}

// EnsureConfigDir creates the configuration directory if it doesn't exist
func EnsureConfigDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return configDir, nil
}

// GetConfigFilePath returns the full path to the config file
func GetConfigFilePath() (string, error) {
	configDir, err := EnsureConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "config.json"), nil
}
