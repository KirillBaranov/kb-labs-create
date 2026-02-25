package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/kb-labs/create/internal/config"
	"github.com/kb-labs/create/internal/installer"
	"github.com/kb-labs/create/internal/logger"
	"github.com/kb-labs/create/internal/manifest"
	"github.com/kb-labs/create/internal/pm"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an installed platform",
	Long: `Compares the current manifest against the installed snapshot,
shows what changed, and applies updates after confirmation.`,
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	platformDir, err := resolvePlatformDir(cmd)
	if err != nil {
		return err
	}

	m, err := manifest.LoadDefault()
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	log, err := logger.New(platformDir)
	if err != nil {
		return err
	}
	defer log.Close()

	ins := &installer.Installer{
		PM:  pm.Detect(),
		Log: log,
	}

	fmt.Println("Checking for updates...")
	diff, err := ins.Diff(platformDir, m)
	if err != nil {
		return err
	}

	if !diff.HasChanges() {
		fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("✓ Already up to date"))
		return nil
	}

	printDiff(diff)

	if !confirm("Apply updates? [Y/n] ") {
		fmt.Println("Cancelled.")
		return nil
	}

	result, err := ins.Update(platformDir, m)
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	ok := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	fmt.Printf("\n%s\n", ok.Render("✓ Update complete")+dim.Render(fmt.Sprintf("  (%s)", result.Duration.Round(1e8))))
	return nil
}

func printDiff(d *installer.UpdateDiff) {
	add := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	upd := lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	rem := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	fmt.Println()
	if len(d.Added) > 0 {
		for _, p := range d.Added {
			fmt.Printf("  %s  %s\n", add.Render("+"), p)
		}
	}
	if len(d.Updated) > 0 {
		for _, p := range d.Updated {
			fmt.Printf("  %s  %s\n", upd.Render("↑"), dim.Render(p))
		}
	}
	if len(d.Removed) > 0 {
		for _, p := range d.Removed {
			fmt.Printf("  %s  %s\n", rem.Render("-"), p)
		}
	}
	fmt.Println()
}

func confirm(prompt string) bool {
	fmt.Print(prompt)
	r := bufio.NewReader(os.Stdin)
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "" || line == "y" || line == "yes"
}

// resolvePlatformDir returns the platform dir from --platform flag or config in cwd.
func resolvePlatformDir(cmd *cobra.Command) (string, error) {
	if p, _ := cmd.Flags().GetString("platform"); p != "" {
		return p, nil
	}
	if p, _ := cmd.Root().PersistentFlags().GetString("platform"); p != "" {
		return p, nil
	}
	// Try reading config from current directory.
	cwd, _ := os.Getwd()
	cfg, err := config.Read(cwd)
	if err == nil {
		return cfg.Platform, nil
	}
	return "", fmt.Errorf("platform directory not specified — use --platform or run from the platform directory")
}
