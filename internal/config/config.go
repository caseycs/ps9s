package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GetProfilesFromEnv reads PS9S_AWS_PROFILES environment variable
// and returns a list of AWS profile names.
// If PS9S_AWS_PROFILES is not set, returns the current AWS profile (from AWS_PROFILE)
// or "default" if no profile is specified.
func GetProfilesFromEnv() ([]string, error) {
	envValue := os.Getenv("PS9S_AWS_PROFILES")

	// If PS9S_AWS_PROFILES is not set, use current AWS profile
	if envValue == "" {
		currentProfile := os.Getenv("AWS_PROFILE")
		if currentProfile == "" {
			currentProfile = "default"
		}
		return []string{currentProfile}, nil
	}

	// Split by comma and trim whitespace
	rawProfiles := strings.Split(envValue, ",")
	profiles := make([]string, 0, len(rawProfiles))

	for _, p := range rawProfiles {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			profiles = append(profiles, trimmed)
		}
	}

	if len(profiles) == 0 {
		return nil, fmt.Errorf("no valid profiles found in PS9S_AWS_PROFILES")
	}

	return profiles, nil
}

// GetConfigDir returns the ps9s configuration directory
// Uses XDG_CONFIG_HOME/.ps9s or ~/.ps9s as fallback
func GetConfigDir() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome != "" {
		return filepath.Join(configHome, ".ps9s"), nil
	}

	// Fallback to ~/.ps9s
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(homeDir, ".ps9s"), nil
}

// RegionMapping represents the mapping of profiles to their last selected regions
type RegionMapping struct {
	ProfileRegions map[string]string `json:"profile_regions"`
}

// LoadRegionMapping loads the region mapping from config file
// Returns an empty mapping if file doesn't exist
func LoadRegionMapping() (*RegionMapping, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}

	configFile := filepath.Join(configDir, "regions.json")

	// If file doesn't exist, return empty mapping
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return &RegionMapping{
			ProfileRegions: make(map[string]string),
		}, nil
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var mapping RegionMapping
	if err := json.Unmarshal(data, &mapping); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if mapping.ProfileRegions == nil {
		mapping.ProfileRegions = make(map[string]string)
	}

	return &mapping, nil
}

// SaveRegionMapping saves the region mapping to config file
func SaveRegionMapping(mapping *RegionMapping) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configFile := filepath.Join(configDir, "regions.json")

	data, err := json.MarshalIndent(mapping, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
