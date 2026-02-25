// Package installer orchestrates the KB Labs platform installation and update
// lifecycle. It delegates package operations to a pm.PackageManager and
// persists the resulting configuration via the config package.
package installer

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kb-labs/create/internal/config"
	"github.com/kb-labs/create/internal/logger"
	"github.com/kb-labs/create/internal/manifest"
	"github.com/kb-labs/create/internal/pm"
)

// Selection holds what the user chose to install.
type Selection struct {
	PlatformDir string
	ProjectCWD  string
	Services    []string // component IDs
	Plugins     []string // component IDs
}

// Result is returned after a successful Install.
type Result struct {
	PlatformDir string
	ProjectCWD  string
	ConfigPath  string
	Duration    time.Duration
}

// UpdateDiff describes changes between the installed manifest and the current one.
type UpdateDiff struct {
	Updated []string // packages with version changes
	Added   []string // new packages
	Removed []string // removed packages
}

// HasChanges returns true if there is anything to update.
func (d *UpdateDiff) HasChanges() bool {
	return len(d.Updated)+len(d.Added)+len(d.Removed) > 0
}

// UpdateResult is returned after a successful Update.
type UpdateResult struct {
	Diff     *UpdateDiff
	Duration time.Duration
}

// Installer orchestrates platform installation and updates.
type Installer struct {
	PM     pm.PackageManager
	Log    *logger.Logger
	OnStep func(step, total int, label string) // optional progress callback
}

// Install installs the platform according to sel.
func (ins *Installer) Install(sel *Selection, m *manifest.Manifest) (*Result, error) {
	start := time.Now()

	ins.step(1, 4, "Installing core packages")
	if err := ins.installGroup(sel.PlatformDir, m.CorePackageNames()); err != nil {
		return nil, fmt.Errorf("core: %w", err)
	}

	ins.step(2, 4, "Installing services")
	svcPkgs := ins.selectedPkgs(m.Services, sel.Services)
	if len(svcPkgs) > 0 {
		if err := ins.installGroup(sel.PlatformDir, svcPkgs); err != nil {
			return nil, fmt.Errorf("services: %w", err)
		}
	}

	ins.step(3, 4, "Installing plugins")
	pluginPkgs := ins.selectedPkgs(m.Plugins, sel.Plugins)
	if len(pluginPkgs) > 0 {
		if err := ins.installGroup(sel.PlatformDir, pluginPkgs); err != nil {
			return nil, fmt.Errorf("plugins: %w", err)
		}
	}

	ins.step(4, 4, "Writing config")
	cfg := config.NewConfig(sel.PlatformDir, sel.ProjectCWD, ins.PM.Name(), m)
	if err := config.Write(sel.PlatformDir, cfg); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	// Create project .kb dir so the platform can write artifacts there.
	if err := os.MkdirAll(sel.ProjectCWD+"/.kb", 0o755); err != nil {
		return nil, fmt.Errorf("project dir: %w", err)
	}

	return &Result{
		PlatformDir: sel.PlatformDir,
		ProjectCWD:  sel.ProjectCWD,
		ConfigPath:  config.ConfigPath(sel.PlatformDir),
		Duration:    time.Since(start),
	}, nil
}

// Diff computes what would change if Update were applied now.
func (ins *Installer) Diff(platformDir string, current *manifest.Manifest) (*UpdateDiff, error) {
	cfg, err := config.Read(platformDir)
	if err != nil {
		return nil, err
	}

	installed := pkgSet(cfg.Manifest)
	currentSet := pkgSet(*current)

	diff := &UpdateDiff{}
	for pkg := range currentSet {
		if _, ok := installed[pkg]; !ok {
			diff.Added = append(diff.Added, pkg)
		} else {
			// version comparison not available without npm query; mark as updated
			diff.Updated = append(diff.Updated, pkg)
		}
	}
	for pkg := range installed {
		if _, ok := currentSet[pkg]; !ok {
			diff.Removed = append(diff.Removed, pkg)
		}
	}
	return diff, nil
}

// Update applies the diff: installs new packages, updates existing ones.
func (ins *Installer) Update(platformDir string, current *manifest.Manifest) (*UpdateResult, error) {
	start := time.Now()

	diff, err := ins.Diff(platformDir, current)
	if err != nil {
		return nil, err
	}

	if len(diff.Added) > 0 {
		ins.Log.Printf("Installing new packages: %s", strings.Join(diff.Added, " "))
		if err := ins.installGroup(platformDir, diff.Added); err != nil {
			return nil, fmt.Errorf("add new packages: %w", err)
		}
	}

	allPkgs := current.CorePackageNames()
	for _, c := range append(current.Services, current.Plugins...) {
		allPkgs = append(allPkgs, c.Pkg)
	}
	if err := ins.updateGroup(platformDir, allPkgs); err != nil {
		return nil, fmt.Errorf("update packages: %w", err)
	}

	// Refresh config snapshot.
	cfg, err := config.Read(platformDir)
	if err != nil {
		return nil, err
	}
	cfg.Manifest = *current
	if err := config.Write(platformDir, cfg); err != nil {
		return nil, err
	}

	return &UpdateResult{Diff: diff, Duration: time.Since(start)}, nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func (ins *Installer) step(n, total int, label string) {
	ins.Log.Printf("[%d/%d] %s", n, total, label)
	if ins.OnStep != nil {
		ins.OnStep(n, total, label)
	}
}

// installGroup installs pkgs into dir, draining progress lines to the log.
// It waits for the log-drain goroutine to finish before returning so no
// output is lost even when the channel is buffered.
func (ins *Installer) installGroup(dir string, pkgs []string) error {
	ch := make(chan pm.Progress, 64)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for p := range ch {
			if p.Line != "" {
				ins.Log.Printf("  %s", p.Line)
			}
		}
	}()
	err := ins.PM.Install(dir, pkgs, ch)
	close(ch)
	<-done // wait for drain goroutine to flush all buffered lines
	return err
}

// updateGroup updates pkgs in dir, draining progress lines to the log.
func (ins *Installer) updateGroup(dir string, pkgs []string) error {
	ch := make(chan pm.Progress, 64)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for p := range ch {
			if p.Line != "" {
				ins.Log.Printf("  %s", p.Line)
			}
		}
	}()
	err := ins.PM.Update(dir, pkgs, ch)
	close(ch)
	<-done
	return err
}

func (ins *Installer) selectedPkgs(components []manifest.Component, ids []string) []string {
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	var out []string
	for _, c := range components {
		if set[c.ID] {
			out = append(out, c.Pkg)
		}
	}
	return out
}

func pkgSet(m manifest.Manifest) map[string]bool {
	s := make(map[string]bool)
	for _, p := range m.Core {
		s[p.Name] = true
	}
	for _, c := range append(m.Services, m.Plugins...) {
		s[c.Pkg] = true
	}
	return s
}
