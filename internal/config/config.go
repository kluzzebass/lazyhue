// Package config handles application configuration and credential storage.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	configDirName  = "lazyhue"
	configFileName = "config.json"
)

// Config holds application configuration.
type Config struct {
	PollInterval  string `json:"poll_interval"`
	DefaultBridge string `json:"default_bridge,omitempty"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		PollInterval: "2s",
	}
}

// configDir returns the path to the lazyhue config directory.
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", configDirName), nil
}

// ensureConfigDir creates the config directory if it doesn't exist.
func ensureConfigDir() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

// Load reads the config file, returning defaults if it doesn't exist.
func Load() (*Config, error) {
	dir, err := configDir()
	if err != nil {
		return DefaultConfig(), nil
	}

	path := filepath.Join(dir, configFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save writes the config to disk.
func (c *Config) Save() error {
	dir, err := ensureConfigDir()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(dir, configFileName)
	return os.WriteFile(path, data, 0600)
}

