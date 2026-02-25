// Package config manages the platform configuration file written to
// <platformDir>/.kb/kb.config.json. The schema is versioned to support
// forward-compatible migrations.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kb-labs/create/internal/manifest"
)

const (
	configVersion = 1
	configDir     = ".kb"
	configFile    = "kb.config.json"
)

// PlatformConfig is the persistent state written to <platform>/.kb/kb.config.json.
// Version field enables future migrations.
type PlatformConfig struct {
	InstalledAt time.Time         `json:"installedAt"`
	Platform    string            `json:"platform"`
	CWD         string            `json:"cwd"`
	PM          string            `json:"pm"`
	Manifest    manifest.Manifest `json:"manifest"`
	Version     int               `json:"version"`
}

// ConfigPath returns the path to the config file for the given platform directory.
func ConfigPath(platformDir string) string {
	return filepath.Join(platformDir, configDir, configFile)
}

// Write persists config to <platformDir>/.kb/kb.config.json.
func Write(platformDir string, cfg *PlatformConfig) error {
	dir := filepath.Join(platformDir, configDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	path := filepath.Join(dir, configFile)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// Read loads and parses the config from <platformDir>/.kb/kb.config.json.
func Read(platformDir string) (*PlatformConfig, error) {
	path := ConfigPath(platformDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no config found at %s — is the platform installed?", path)
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg PlatformConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Future: handle cfg.Version < configVersion migrations here.

	return &cfg, nil
}

// NewConfig creates a fresh PlatformConfig ready to be written.
func NewConfig(platformDir, cwd, pmName string, m *manifest.Manifest) *PlatformConfig {
	abs, _ := filepath.Abs(platformDir)
	absCWD, _ := filepath.Abs(cwd)
	return &PlatformConfig{
		Version:     configVersion,
		Platform:    abs,
		CWD:         absCWD,
		PM:          pmName,
		InstalledAt: time.Now().UTC(),
		Manifest:    *m,
	}
}
