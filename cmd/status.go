package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/kb-labs/create/internal/config"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show installation status",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	platformDir, err := resolvePlatformDir(cmd)
	if err != nil {
		return err
	}

	cfg, err := config.Read(platformDir)
	if err != nil {
		return err
	}

	label := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("8"))
	val := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	ok := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))

	fmt.Println()
	fmt.Printf("  %s %s\n\n", label.Render("Platform:"), val.Render(cfg.Platform))
	fmt.Printf("  %s %s\n", label.Render("Project:  "), val.Render(cfg.CWD))
	fmt.Printf("  %s %s\n", label.Render("PM:       "), cfg.PM)
	fmt.Printf("  %s %s\n", label.Render("Installed:"), cfg.InstalledAt.Format("2006-01-02 15:04"))
	fmt.Printf("  %s %s\n\n", label.Render("Manifest: "), cfg.Manifest.Version)

	// core
	fmt.Printf("  %s\n", label.Render("Core packages:"))
	for _, p := range cfg.Manifest.Core {
		fmt.Printf("    %s %s\n", ok.Render("●"), p.Name)
	}

	// services
	if len(cfg.Manifest.Services) > 0 {
		fmt.Printf("\n  %s\n", label.Render("Services:"))
		for _, s := range cfg.Manifest.Services {
			mark := ok.Render("●")
			fmt.Printf("    %s %-15s  %s\n", mark, s.ID, dimStr(s.Description))
		}
	}

	// plugins
	if len(cfg.Manifest.Plugins) > 0 {
		fmt.Printf("\n  %s\n", label.Render("Plugins:"))
		for _, p := range cfg.Manifest.Plugins {
			mark := ok.Render("●")
			fmt.Printf("    %s %-15s  %s\n", mark, p.ID, dimStr(p.Description))
		}
	}

	fmt.Println()
	return nil
}

func dimStr(s string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(s)
}

