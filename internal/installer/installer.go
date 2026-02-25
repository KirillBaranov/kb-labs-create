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
	OnStep func(step, total int, label string) // called at each named stage
	OnLine func(line string)                   // called for each raw output line from pm
}

// Install installs the platform according to sel.
// All selected packages are passed to the package manager in a single
// invocation so it can resolve and deduplicate the dependency graph at once.
func (ins *Installer) Install(sel *Selection, m *manifest.Manifest) (*Result, error) {
	start := time.Now()

	// Collect every package that needs to be installed in one shot.
	allPkgs := m.CorePackageNames()
	allPkgs = append(allPkgs, ins.selectedPkgs(m.Services, sel.Services)...)
	allPkgs = append(allPkgs, ins.selectedPkgs(m.Plugins, sel.Plugins)...)

	ins.step(1, 2, fmt.Sprintf("Installing %d packages via %s", len(allPkgs), ins.PM.Name()))
	if err := ins.installGroup(sel.PlatformDir, allPkgs); err != nil {
		return nil, fmt.Errorf("install: %w", err)
	}

	ins.step(2, 2, "Writing config")
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

// installGroup installs pkgs into dir, draining progress lines to the log
// and forwarding each line to OnLine if set.
// It waits for the drain goroutine to finish before returning so no output
// is lost even when the channel is buffered.
func (ins *Installer) installGroup(dir string, pkgs []string) error {
	return ins.runGroup(dir, pkgs, ins.PM.Install)
}

// updateGroup updates pkgs in dir, draining progress lines to the log.
func (ins *Installer) updateGroup(dir string, pkgs []string) error {
	return ins.runGroup(dir, pkgs, ins.PM.Update)
}

// runGroup is the shared driver for installGroup / updateGroup.
func (ins *Installer) runGroup(dir string, pkgs []string, op func(string, []string, chan<- pm.Progress) error) error {
	ch := make(chan pm.Progress, 64)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for p := range ch {
			if p.Line == "" {
				continue
			}
			ins.Log.Printf("  %s", p.Line)
			if ins.OnLine != nil {
				ins.OnLine(p.Line)
			}
		}
	}()
	err := op(dir, pkgs, ch)
	close(ch)
	<-done // wait for drain goroutine to flush all buffered lines
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
