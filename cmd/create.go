package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/kb-labs/create/internal/installer"
	"github.com/kb-labs/create/internal/logger"
	"github.com/kb-labs/create/internal/manifest"
	"github.com/kb-labs/create/internal/pm"
	"github.com/kb-labs/create/internal/wizard"
)

var (
	flagYes      bool
	flagPlatform string
)

func init() {
	rootCmd.Flags().BoolVarP(&flagYes, "yes", "y", false, "skip wizard and install with defaults")
	rootCmd.Flags().StringVar(&flagPlatform, "platform", "", "platform installation directory")
}

func runCreate(cmd *cobra.Command, args []string) error {
	// Resolve default project directory from arg or cwd.
	projectCWD := ""
	if len(args) > 0 {
		abs, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}
		projectCWD = abs
	}

	// Load manifest (embedded for now).
	m, err := manifest.LoadDefault()
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	// Show wizard or use defaults.
	sel, err := wizard.Run(m, wizard.WizardOptions{
		Yes:                flagYes,
		DefaultProjectCWD:  projectCWD,
		DefaultPlatformDir: flagPlatform,
	})
	if err != nil {
		return err // includes "cancelled"
	}

	// Create platform directory.
	if err := os.MkdirAll(sel.PlatformDir, 0o755); err != nil {
		return fmt.Errorf("create platform dir: %w", err)
	}

	// Set up logger (writes to stderr + log file).
	log, err := logger.New(sel.PlatformDir)
	if err != nil {
		return err
	}
	defer log.Close()

	fmt.Println()

	packageManager := pm.Detect()
	log.Printf("Using %s", packageManager.Name())

	sp := newSpinner()

	ins := &installer.Installer{
		PM:  packageManager,
		Log: log,
		OnStep: func(step, total int, label string) {
			sp.setLabel(fmt.Sprintf("[%d/%d] %s", step, total, label))
		},
		OnLine: func(line string) {
			sp.setDetail(line)
		},
	}

	sp.start()
	result, err := ins.Install(sel, m)
	sp.stop(err)

	if err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	printSuccess(result)
	return nil
}

// ── spinner ───────────────────────────────────────────────────────────────────

// spinner renders a rotating indicator with a label and a detail line
// that updates in-place while the install is running.
type spinner struct {
	mu     sync.Mutex
	label  string
	detail string
	done   chan struct{}
}

func newSpinner() *spinner { return &spinner{done: make(chan struct{})} }

func (s *spinner) setLabel(l string) {
	s.mu.Lock()
	s.label = l
	s.mu.Unlock()
}

func (s *spinner) setDetail(d string) {
	s.mu.Lock()
	// Truncate long npm lines so they fit on one terminal line.
	if len(d) > 72 {
		d = d[:69] + "..."
	}
	s.detail = d
	s.mu.Unlock()
}

// start launches the render loop in a goroutine.
func (s *spinner) start() {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	go func() {
		i := 0
		for {
			select {
			case <-s.done:
				return
			case <-time.After(80 * time.Millisecond):
				s.mu.Lock()
				label := s.label
				detail := s.detail
				s.mu.Unlock()

				frame := frames[i%len(frames)]
				i++

				// \r returns to column 0; \033[K clears to end of line.
				fmt.Printf("\r\033[K  %s %s\n\r\033[K    %s",
					frame,
					label,
					dim.Render(detail),
				)
				// Move cursor up one line so next tick overwrites both lines.
				fmt.Print("\033[1A")
			}
		}
	}()
}

// stop halts the spinner and prints a final status line.
func (s *spinner) stop(err error) {
	close(s.done)
	time.Sleep(90 * time.Millisecond) // let last frame finish

	s.mu.Lock()
	label := s.label
	s.mu.Unlock()

	// Clear both lines used by the spinner.
	fmt.Print("\r\033[K\033[1B\r\033[K\033[1A")

	if err == nil {
		ok := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
		fmt.Printf("  %s %s\n", ok.Render("✓"), label)
	} else {
		bad := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		fmt.Printf("  %s %s\n", bad.Render("✗"), label)
	}
}

// ── success banner ────────────────────────────────────────────────────────────

func printSuccess(r *installer.Result) {
	ok := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	val := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))

	fmt.Println()
	fmt.Println(ok.Render("✓ Installation complete") + dim.Render(fmt.Sprintf("  (%s)", r.Duration.Round(time.Second))))
	fmt.Println()
	fmt.Printf("  Platform:  %s\n", val.Render(r.PlatformDir))
	fmt.Printf("  Project:   %s\n", val.Render(r.ProjectCWD))
	fmt.Printf("  Config:    %s\n", val.Render(r.ConfigPath))
	fmt.Println()
	fmt.Printf("  %s\n", dim.Render("Next steps:"))
	fmt.Printf("    cd %s\n", r.ProjectCWD)
	fmt.Printf("    kb dev:start\n")
	fmt.Println()
}
