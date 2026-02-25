package cmd

import (
	"fmt"
	"os"
	"path/filepath"

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

	fmt.Println() // blank line before progress

	packageManager := pm.Detect()
	log.Printf("Using %s", packageManager.Name())

	ins := &installer.Installer{
		PM:  packageManager,
		Log: log,
		OnStep: func(step, total int, label string) {
			fmt.Printf("  [%d/%d] %s...\n", step, total, label)
		},
	}

	result, err := ins.Install(sel, m)
	if err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	printSuccess(result)
	return nil
}

func printSuccess(r *installer.Result) {
	ok := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	val := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))

	fmt.Println()
	fmt.Println(ok.Render("✓ Installation complete") + dim.Render(fmt.Sprintf("  (%s)", r.Duration.Round(1e8))))
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
