package wizard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"

	"github.com/kb-labs/create/internal/manifest"
)

// makeInput returns a textinput.Model pre-filled with value.
func makeInput(value string) textinput.Model {
	ti := textinput.New()
	ti.SetValue(value)
	return ti
}

// sampleManifest returns a small manifest for use in tests.
func sampleManifest() *manifest.Manifest {
	return &manifest.Manifest{
		Version: "1.0.0",
		Core:    []manifest.Package{{Name: "@kb-labs/cli-bin"}},
		Services: []manifest.Component{
			{ID: "rest", Pkg: "@kb-labs/rest-api", Default: true},
			{ID: "studio", Pkg: "@kb-labs/studio", Default: false},
		},
		Plugins: []manifest.Component{
			{ID: "mind", Pkg: "@kb-labs/mind", Default: true},
			{ID: "agents", Pkg: "@kb-labs/agents", Default: false},
		},
	}
}

// ── defaultSelection ─────────────────────────────────────────────────────────

// TestDefaultSelectionPicksDefaults verifies that only default-marked components
// appear in the selection when no overrides are provided.
func TestDefaultSelectionPicksDefaults(t *testing.T) {
	m := sampleManifest()
	sel := defaultSelection(m, WizardOptions{})

	if len(sel.Services) != 1 || sel.Services[0] != "rest" {
		t.Errorf("Services = %v, want [rest]", sel.Services)
	}
	if len(sel.Plugins) != 1 || sel.Plugins[0] != "mind" {
		t.Errorf("Plugins = %v, want [mind]", sel.Plugins)
	}
}

// TestDefaultSelectionPlatformDirOverride verifies that WizardOptions.DefaultPlatformDir
// is used when set.
func TestDefaultSelectionPlatformDirOverride(t *testing.T) {
	m := sampleManifest()
	sel := defaultSelection(m, WizardOptions{DefaultPlatformDir: "/custom/platform"})

	if sel.PlatformDir != "/custom/platform" {
		t.Errorf("PlatformDir = %q, want %q", sel.PlatformDir, "/custom/platform")
	}
}

// TestDefaultSelectionCWDOverride verifies that WizardOptions.DefaultProjectCWD
// is used when set.
func TestDefaultSelectionCWDOverride(t *testing.T) {
	m := sampleManifest()
	sel := defaultSelection(m, WizardOptions{DefaultProjectCWD: "/custom/project"})

	if sel.ProjectCWD != "/custom/project" {
		t.Errorf("ProjectCWD = %q, want %q", sel.ProjectCWD, "/custom/project")
	}
}

// TestDefaultSelectionFallsBackToHomeAndCWD verifies that when no overrides are
// given, PlatformDir is under home and ProjectCWD is the current directory.
func TestDefaultSelectionFallsBackToHomeAndCWD(t *testing.T) {
	m := sampleManifest()
	sel := defaultSelection(m, WizardOptions{})

	home, _ := os.UserHomeDir()
	if !strings.HasPrefix(sel.PlatformDir, home) {
		t.Errorf("PlatformDir %q does not start with home %q", sel.PlatformDir, home)
	}

	cwd, _ := os.Getwd()
	if sel.ProjectCWD != cwd {
		t.Errorf("ProjectCWD = %q, want %q", sel.ProjectCWD, cwd)
	}
}

// TestDefaultSelectionNoDefaults verifies an empty selection when no components
// are marked default.
func TestDefaultSelectionNoDefaults(t *testing.T) {
	m := &manifest.Manifest{
		Services: []manifest.Component{{ID: "rest", Default: false}},
		Plugins:  []manifest.Component{{ID: "mind", Default: false}},
	}
	sel := defaultSelection(m, WizardOptions{})
	if len(sel.Services) != 0 {
		t.Errorf("Services = %v, want []", sel.Services)
	}
	if len(sel.Plugins) != 0 {
		t.Errorf("Plugins = %v, want []", sel.Plugins)
	}
}

// ── expandHome ───────────────────────────────────────────────────────────────

// TestExpandHomeTilde verifies that a ~/... path is expanded to the real home.
func TestExpandHomeTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("UserHomeDir unavailable:", err)
	}

	got := expandHome("~/projects/foo")
	want := filepath.Join(home, "projects", "foo")
	if got != want {
		t.Errorf("expandHome(~/projects/foo) = %q, want %q", got, want)
	}
}

// TestExpandHomeAbsolute verifies that an absolute path is returned unchanged.
func TestExpandHomeAbsolute(t *testing.T) {
	path := "/usr/local/bin"
	if got := expandHome(path); got != path {
		t.Errorf("expandHome(%q) = %q, want %q", path, got, path)
	}
}

// TestExpandHomeNoTilde verifies that a relative path without ~ is unchanged.
func TestExpandHomeNoTilde(t *testing.T) {
	path := "relative/path"
	if got := expandHome(path); got != path {
		t.Errorf("expandHome(%q) = %q, want %q", path, got, path)
	}
}

// TestExpandHomeTildeOnly verifies ~/  with nothing after it expands to home.
func TestExpandHomeTildeOnly(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("UserHomeDir unavailable:", err)
	}
	// "~/" has an empty trailing part — filepath.Join handles this correctly.
	got := expandHome("~/")
	if got != home {
		t.Errorf("expandHome(~/) = %q, want %q", got, home)
	}
}

// ── toSelection ──────────────────────────────────────────────────────────────

// TestToSelectionCheckedItems verifies that checked services/plugins are
// included in the resulting Selection.
func TestToSelectionCheckedItems(t *testing.T) {
	m := wizardModel{
		platformInput: makeInput("/platform"),
		cwdInput:      makeInput("/project"),
		services: []checkItem{
			{id: "rest", checked: true},
			{id: "studio", checked: false},
		},
		plugins: []checkItem{
			{id: "mind", checked: true},
			{id: "agents", checked: false},
		},
	}

	sel := m.toSelection()

	if sel.PlatformDir != "/platform" {
		t.Errorf("PlatformDir = %q, want %q", sel.PlatformDir, "/platform")
	}
	if sel.ProjectCWD != "/project" {
		t.Errorf("ProjectCWD = %q, want %q", sel.ProjectCWD, "/project")
	}
	if len(sel.Services) != 1 || sel.Services[0] != "rest" {
		t.Errorf("Services = %v, want [rest]", sel.Services)
	}
	if len(sel.Plugins) != 1 || sel.Plugins[0] != "mind" {
		t.Errorf("Plugins = %v, want [mind]", sel.Plugins)
	}
}

// TestToSelectionNoneChecked verifies that an empty selection is produced
// when no items are checked.
func TestToSelectionNoneChecked(t *testing.T) {
	m := wizardModel{
		platformInput: makeInput("/p"),
		cwdInput:      makeInput("/c"),
		services:      []checkItem{{id: "rest", checked: false}},
		plugins:       []checkItem{{id: "mind", checked: false}},
	}

	sel := m.toSelection()
	if len(sel.Services) != 0 {
		t.Errorf("Services = %v, want []", sel.Services)
	}
	if len(sel.Plugins) != 0 {
		t.Errorf("Plugins = %v, want []", sel.Plugins)
	}
}

// TestToSelectionAllChecked verifies that all items appear when all are checked.
func TestToSelectionAllChecked(t *testing.T) {
	m := wizardModel{
		platformInput: makeInput("/p"),
		cwdInput:      makeInput("/c"),
		services: []checkItem{
			{id: "rest", checked: true},
			{id: "studio", checked: true},
		},
		plugins: []checkItem{
			{id: "mind", checked: true},
			{id: "agents", checked: true},
		},
	}

	sel := m.toSelection()
	if len(sel.Services) != 2 {
		t.Errorf("Services len = %d, want 2", len(sel.Services))
	}
	if len(sel.Plugins) != 2 {
		t.Errorf("Plugins len = %d, want 2", len(sel.Plugins))
	}
}
