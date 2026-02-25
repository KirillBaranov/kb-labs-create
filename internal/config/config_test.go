package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kb-labs/create/internal/manifest"
)

func sampleManifest() manifest.Manifest {
	return manifest.Manifest{
		Version: "1.0.0",
		Core:    []manifest.Package{{Name: "@kb-labs/cli-bin"}},
		Services: []manifest.Component{
			{ID: "rest", Pkg: "@kb-labs/rest-api", Description: "REST API", Default: true},
		},
		Plugins: []manifest.Component{
			{ID: "mind", Pkg: "@kb-labs/mind", Description: "RAG", Default: true},
		},
	}
}

// TestNewConfig verifies that NewConfig populates all required fields.
func TestNewConfig(t *testing.T) {
	m := sampleManifest()
	cfg := NewConfig("/tmp/platform", "/tmp/project", "pnpm", &m)

	if cfg.Version != configVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, configVersion)
	}
	if cfg.PM != "pnpm" {
		t.Errorf("PM = %q, want %q", cfg.PM, "pnpm")
	}
	if cfg.InstalledAt.IsZero() {
		t.Error("InstalledAt is zero")
	}
	if cfg.Manifest.Version != "1.0.0" {
		t.Errorf("Manifest.Version = %q, want %q", cfg.Manifest.Version, "1.0.0")
	}
}

// TestWriteThenRead verifies round-trip write → read produces identical config.
func TestWriteThenRead(t *testing.T) {
	dir := t.TempDir()
	m := sampleManifest()
	want := NewConfig(dir, "/some/project", "npm", &m)
	// Fix timestamp for deterministic comparison.
	want.InstalledAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	if err := Write(dir, want); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	got, err := Read(dir)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if got.Version != want.Version {
		t.Errorf("Version: got %d, want %d", got.Version, want.Version)
	}
	if got.PM != want.PM {
		t.Errorf("PM: got %q, want %q", got.PM, want.PM)
	}
	if !got.InstalledAt.Equal(want.InstalledAt) {
		t.Errorf("InstalledAt: got %v, want %v", got.InstalledAt, want.InstalledAt)
	}
	if got.Manifest.Version != want.Manifest.Version {
		t.Errorf("Manifest.Version: got %q, want %q", got.Manifest.Version, want.Manifest.Version)
	}
	if len(got.Manifest.Core) != len(want.Manifest.Core) {
		t.Errorf("Core len: got %d, want %d", len(got.Manifest.Core), len(want.Manifest.Core))
	}
}

// TestReadMissing verifies that reading a non-existent config returns an error.
func TestReadMissing(t *testing.T) {
	dir := t.TempDir()
	_, err := Read(dir)
	if err == nil {
		t.Error("Read() on missing config should return error, got nil")
	}
}

// TestConfigPath verifies the expected config file path.
func TestConfigPath(t *testing.T) {
	got := ConfigPath("/platform")
	want := filepath.Join("/platform", configDir, configFile)
	if got != want {
		t.Errorf("ConfigPath = %q, want %q", got, want)
	}
}

// TestWriteCreatesDirectory verifies that Write creates .kb/ if it does not exist.
func TestWriteCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	// Remove any pre-existing .kb directory.
	os.RemoveAll(filepath.Join(dir, ".kb"))

	m := sampleManifest()
	cfg := NewConfig(dir, dir, "npm", &m)
	if err := Write(dir, cfg); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if _, err := os.Stat(ConfigPath(dir)); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}
