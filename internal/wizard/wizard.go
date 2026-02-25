// Package wizard implements the interactive Bubble Tea TUI for kb-create.
// The wizard walks through three stages: directory inputs, component
// selection (services + plugins), and a final confirmation screen.
// When WizardOptions.Yes is true the TUI is skipped entirely and
// Run returns a Selection populated with manifest defaults.
package wizard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kb-labs/create/internal/installer"
	"github.com/kb-labs/create/internal/manifest"
)

// WizardOptions controls wizard behaviour.
type WizardOptions struct {
	// DefaultProjectCWD pre-fills the project directory input.
	DefaultProjectCWD string
	// DefaultPlatformDir pre-fills the platform directory input.
	DefaultPlatformDir string
	// Yes skips the TUI and returns defaults immediately.
	Yes bool
}

// Run shows the interactive wizard and returns the user's selection.
// If opts.Yes is true, returns defaults without launching TUI.
func Run(m *manifest.Manifest, opts WizardOptions) (*installer.Selection, error) {
	if opts.Yes {
		return defaultSelection(m, opts), nil
	}

	model := newModel(m, opts)
	p := tea.NewProgram(model, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return nil, err
	}
	result := final.(wizardModel)
	if result.cancelled {
		return nil, fmt.Errorf("installation cancelled")
	}
	return result.toSelection(), nil
}

// ── styles ────────────────────────────────────────────────────────────────────

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	sectionStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("8"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	normalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	focusStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	helpStyle     = dimStyle
)

// ── model stages ─────────────────────────────────────────────────────────────

type stage int

const (
	stageDirs    stage = iota // entering platform/project dirs
	stageOptions              // choosing services & plugins
	stageConfirm              // confirm / cancel
)

type checkItem struct {
	id      string
	pkg     string
	desc    string
	checked bool
}

type wizardModel struct {
	manifest      *manifest.Manifest
	errMsg        string
	services      []checkItem
	plugins       []checkItem
	platformInput textinput.Model
	cwdInput      textinput.Model
	stage         stage
	activeInput   int
	cursor        int
	cancelled     bool
	confirmed     bool
}

func newModel(m *manifest.Manifest, opts WizardOptions) wizardModel {
	platformDir := opts.DefaultPlatformDir
	if platformDir == "" {
		home, _ := os.UserHomeDir()
		platformDir = filepath.Join(home, "kb-platform")
	}
	cwd := opts.DefaultProjectCWD
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	pi := textinput.New()
	pi.Placeholder = "~/kb-platform"
	pi.SetValue(platformDir)
	pi.Focus()
	pi.Width = 50

	ci := textinput.New()
	ci.Placeholder = "~/projects/my-project"
	ci.SetValue(cwd)
	ci.Width = 50

	services := make([]checkItem, len(m.Services))
	for i, s := range m.Services {
		services[i] = checkItem{id: s.ID, pkg: s.Pkg, desc: s.Description, checked: s.Default}
	}
	plugins := make([]checkItem, len(m.Plugins))
	for i, p := range m.Plugins {
		plugins[i] = checkItem{id: p.ID, pkg: p.Pkg, desc: p.Description, checked: p.Default}
	}

	return wizardModel{
		manifest:      m,
		stage:         stageDirs,
		platformInput: pi,
		cwdInput:      ci,
		services:      services,
		plugins:       plugins,
	}
}

// ── tea.Model interface ───────────────────────────────────────────────────────

func (m wizardModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m wizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		return m.handleKey(key)
	}
	// forward to active input
	var cmd tea.Cmd
	if m.stage == stageDirs {
		if m.activeInput == 0 {
			m.platformInput, cmd = m.platformInput.Update(msg)
		} else {
			m.cwdInput, cmd = m.cwdInput.Update(msg)
		}
	}
	return m, cmd
}

func (m wizardModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.stage {
	case stageDirs:
		return m.handleDirsKey(msg)
	case stageOptions:
		return m.handleOptionsKey(msg)
	case stageConfirm:
		return m.handleConfirmKey(msg)
	}
	return m, nil
}

func (m wizardModel) handleDirsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.cancelled = true
		return m, tea.Quit
	case "tab", "down":
		m.activeInput = 1 - m.activeInput
		if m.activeInput == 0 {
			m.platformInput.Focus()
			m.cwdInput.Blur()
		} else {
			m.cwdInput.Focus()
			m.platformInput.Blur()
		}
		return m, textinput.Blink
	case "enter":
		if err := m.validateDirs(); err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		m.errMsg = ""
		m.stage = stageOptions
		m.cursor = 0
		return m, nil
	}
	var cmd tea.Cmd
	if m.activeInput == 0 {
		m.platformInput, cmd = m.platformInput.Update(msg)
	} else {
		m.cwdInput, cmd = m.cwdInput.Update(msg)
	}
	return m, cmd
}

func (m wizardModel) handleOptionsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	total := len(m.services) + len(m.plugins)
	switch msg.String() {
	case "ctrl+c", "esc":
		m.cancelled = true
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < total-1 {
			m.cursor++
		}
	case " ":
		m.toggleCursor()
	case "enter":
		m.stage = stageConfirm
	}
	return m, nil
}

func (m wizardModel) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc", "n", "N":
		m.cancelled = true
		return m, tea.Quit
	case "enter", "y", "Y":
		m.confirmed = true
		return m, tea.Quit
	}
	return m, nil
}

func (m *wizardModel) toggleCursor() {
	if m.cursor < len(m.services) {
		m.services[m.cursor].checked = !m.services[m.cursor].checked
	} else {
		i := m.cursor - len(m.services)
		m.plugins[i].checked = !m.plugins[i].checked
	}
}

func (m wizardModel) validateDirs() error {
	if strings.TrimSpace(m.platformInput.Value()) == "" {
		return fmt.Errorf("platform directory is required")
	}
	if strings.TrimSpace(m.cwdInput.Value()) == "" {
		return fmt.Errorf("project directory is required")
	}
	return nil
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m wizardModel) View() string {
	switch m.stage {
	case stageDirs:
		return m.viewDirs()
	case stageOptions:
		return m.viewOptions()
	case stageConfirm:
		return m.viewConfirm()
	}
	return ""
}

func (m wizardModel) viewDirs() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("  kb-create") + "  platform installer\n\n")

	b.WriteString("  " + sectionStyle.Render("Platform directory") + "\n")
	b.WriteString("  " + m.platformInput.View() + "\n")
	b.WriteString(dimStyle.Render("  Where the platform (node_modules) will be installed\n\n"))

	b.WriteString("  " + sectionStyle.Render("Project directory") + "\n")
	b.WriteString("  " + m.cwdInput.View() + "\n")
	b.WriteString(dimStyle.Render("  Your project — all CLI calls and artifacts go here\n\n"))

	if m.errMsg != "" {
		b.WriteString("  " + errorStyle.Render("✖ "+m.errMsg) + "\n\n")
	}

	b.WriteString(helpStyle.Render("  tab switch · enter next · esc quit"))
	return b.String()
}

func (m wizardModel) viewOptions() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("  kb-create") + "  select components\n\n")

	// core (always on)
	b.WriteString("  " + sectionStyle.Render("─── Core (always installed) ───") + "\n")
	names := make([]string, len(m.manifest.Core))
	for i, p := range m.manifest.Core {
		names[i] = p.Name
	}
	b.WriteString("  " + dimStyle.Render("  ● "+strings.Join(names, "  ")) + "\n\n")

	// services
	b.WriteString("  " + sectionStyle.Render("─── Services ───") + "\n")
	for i, s := range m.services {
		b.WriteString(m.renderItem(i, s))
	}
	b.WriteString("\n")

	// plugins
	b.WriteString("  " + sectionStyle.Render("─── Plugins ───") + "\n")
	for i, p := range m.plugins {
		b.WriteString(m.renderItem(len(m.services)+i, p))
	}
	b.WriteString("\n")

	b.WriteString(helpStyle.Render("  ↑↓ move · space toggle · enter install · esc quit"))
	return b.String()
}

func (m wizardModel) renderItem(idx int, item checkItem) string {
	cursor := "  "
	if idx == m.cursor {
		cursor = focusStyle.Render(" ▶")
	}
	check := "○"
	style := normalStyle
	if item.checked {
		check = selectedStyle.Render("◉")
		style = selectedStyle
	}
	return fmt.Sprintf("%s %s  %-15s  %s\n",
		cursor, check,
		style.Render(item.id),
		dimStyle.Render(item.desc),
	)
}

func (m wizardModel) viewConfirm() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("  kb-create") + "  ready to install\n\n")
	b.WriteString(fmt.Sprintf("  Platform:  %s\n", focusStyle.Render(m.platformInput.Value())))
	b.WriteString(fmt.Sprintf("  Project:   %s\n\n", focusStyle.Render(m.cwdInput.Value())))

	var selected []string
	for _, s := range m.services {
		if s.checked {
			selected = append(selected, s.id)
		}
	}
	for _, p := range m.plugins {
		if p.checked {
			selected = append(selected, p.id)
		}
	}
	if len(selected) > 0 {
		b.WriteString("  Components: " + strings.Join(selected, ", ") + "\n\n")
	}

	b.WriteString(helpStyle.Render("  Press enter to install · n to cancel"))
	return b.String()
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (m wizardModel) toSelection() *installer.Selection {
	var services, plugins []string
	for _, s := range m.services {
		if s.checked {
			services = append(services, s.id)
		}
	}
	for _, p := range m.plugins {
		if p.checked {
			plugins = append(plugins, p.id)
		}
	}
	return &installer.Selection{
		PlatformDir: expandHome(m.platformInput.Value()),
		ProjectCWD:  expandHome(m.cwdInput.Value()),
		Services:    services,
		Plugins:     plugins,
	}
}

func defaultSelection(m *manifest.Manifest, opts WizardOptions) *installer.Selection {
	home, _ := os.UserHomeDir()
	platformDir := opts.DefaultPlatformDir
	if platformDir == "" {
		platformDir = filepath.Join(home, "kb-platform")
	}
	cwd := opts.DefaultProjectCWD
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	var services, plugins []string
	for _, s := range m.Services {
		if s.Default {
			services = append(services, s.ID)
		}
	}
	for _, p := range m.Plugins {
		if p.Default {
			plugins = append(plugins, p.ID)
		}
	}
	return &installer.Selection{
		PlatformDir: expandHome(platformDir),
		ProjectCWD:  expandHome(cwd),
		Services:    services,
		Plugins:     plugins,
	}
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
