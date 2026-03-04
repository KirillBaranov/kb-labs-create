package cmd

import (
	"fmt"

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

	out := newOutput()

	out.Section("Installation Status")
	out.KeyValue("Platform", cfg.Platform)
	out.KeyValue("Project", cfg.CWD)
	out.KeyValue("PM", cfg.PM)
	out.KeyValue("Installed", cfg.InstalledAt.Format("2006-01-02 15:04"))
	out.KeyValue("Manifest", cfg.Manifest.Version)

	// core
	out.Section("Core packages")
	for _, p := range cfg.Manifest.Core {
		out.Bullet(p.Name, "")
	}

	// services
	if len(cfg.Manifest.Services) > 0 {
		out.Section("Services")
		for _, s := range cfg.Manifest.Services {
			out.Bullet(s.ID, s.Description)
		}
	}

	// plugins
	if len(cfg.Manifest.Plugins) > 0 {
		out.Section("Plugins")
		for _, p := range cfg.Manifest.Plugins {
			out.Bullet(p.ID, p.Description)
		}
	}

	fmt.Println()
	return nil
}
