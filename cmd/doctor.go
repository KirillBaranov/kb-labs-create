package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type doctorCheck struct {
	Name    string
	OK      bool
	Details string
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run environment diagnostics",
	Long:  "Checks local prerequisites and connectivity used by kb-create.",
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	checks := []doctorCheck{
		checkPath(),
		checkBinary("node", "--version"),
		checkBinary("git", "--version"),
		checkBinary("docker", "--version"),
		checkNetwork(),
	}

	okCount := 0
	for _, c := range checks {
		if c.OK {
			okCount++
			fmt.Printf("✓ %-12s %s\n", c.Name, c.Details)
		} else {
			fmt.Printf("✗ %-12s %s\n", c.Name, c.Details)
		}
	}

	fmt.Println()
	fmt.Printf("Doctor summary: %d/%d checks passed\n", okCount, len(checks))
	if okCount != len(checks) {
		return fmt.Errorf("some checks failed")
	}
	return nil
}

func checkPath() doctorCheck {
	path := os.Getenv("PATH")
	target := os.ExpandEnv("$HOME/.local/bin")
	withSep := ":" + path + ":"
	needle := ":" + target + ":"
	if strings.Contains(withSep, needle) {
		return doctorCheck{Name: "PATH", OK: true, Details: target + " is present"}
	}
	return doctorCheck{
		Name:    "PATH",
		OK:      false,
		Details: target + " is missing (add: export PATH=\"$HOME/.local/bin:$PATH\")",
	}
}

func checkBinary(name, arg string) doctorCheck {
	_, err := exec.LookPath(name)
	if err != nil {
		return doctorCheck{Name: name, OK: false, Details: "not found in PATH"}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// #nosec G204 -- command names/args are fixed diagnostics probes.
	out, err := exec.CommandContext(ctx, name, arg).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return doctorCheck{Name: name, OK: false, Details: "found but failed: " + msg}
	}
	version := firstLine(strings.TrimSpace(string(out)))
	if version == "" {
		version = "ok"
	}
	return doctorCheck{Name: name, OK: true, Details: version}
}

func checkNetwork() doctorCheck {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, "https://github.com", http.NoBody)
	if err != nil {
		return doctorCheck{Name: "network", OK: false, Details: err.Error()}
	}

	// #nosec G704 -- request target is a fixed trusted endpoint (github.com).
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return doctorCheck{Name: "network", OK: false, Details: "cannot reach github.com"}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusInternalServerError {
		return doctorCheck{Name: "network", OK: false, Details: fmt.Sprintf("github.com returned %d", resp.StatusCode)}
	}
	return doctorCheck{Name: "network", OK: true, Details: fmt.Sprintf("github.com reachable (%d)", resp.StatusCode)}
}

func firstLine(s string) string {
	if s == "" {
		return ""
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
