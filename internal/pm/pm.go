// Package pm abstracts node package manager operations behind a common interface.
// Use Detect() to obtain the appropriate manager for the current environment.
package pm

import (
	"os/exec"
)

// Progress reports installation progress for a single step.
type Progress struct {
	Error   error
	Package string
	Line    string // raw output line for logging
	Done    bool
}

// InstalledPackage describes a package found in node_modules.
type InstalledPackage struct {
	Name    string
	Version string
}

// PackageManager abstracts npm/pnpm/bun install operations.
// All methods run synchronously and stream progress via the channel.
// The channel is closed when the operation completes.
type PackageManager interface {
	// Name returns "npm" or "pnpm".
	Name() string
	// Install installs the given packages into dir/node_modules.
	Install(dir string, pkgs []string, progress chan<- Progress) error
	// Update updates already-installed packages to their latest versions.
	Update(dir string, pkgs []string, progress chan<- Progress) error
	// ListInstalled returns packages installed in dir.
	ListInstalled(dir string) ([]InstalledPackage, error)
}

// Detect returns pnpm if available, otherwise npm.
func Detect() PackageManager {
	if _, err := exec.LookPath("pnpm"); err == nil {
		return &PnpmManager{}
	}
	return &NpmManager{}
}
