package pm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// NpmManager implements PackageManager using npm.
type NpmManager struct{}

func (n *NpmManager) Name() string { return "npm" }

func (n *NpmManager) Install(dir string, pkgs []string, progress chan<- Progress) error {
	return n.run(dir, append([]string{"install", "--prefix", dir}, pkgs...), progress)
}

func (n *NpmManager) Update(dir string, pkgs []string, progress chan<- Progress) error {
	return n.run(dir, append([]string{"update", "--prefix", dir}, pkgs...), progress)
}

func (n *NpmManager) ListInstalled(dir string) ([]InstalledPackage, error) {
	nmDir := filepath.Join(dir, "node_modules")
	if _, err := os.Stat(nmDir); os.IsNotExist(err) {
		return nil, nil
	}

	cmd := exec.Command("npm", "list", "--prefix", dir, "--json", "--depth=0")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("npm list: %w", err)
	}

	var result struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	pkgs := make([]InstalledPackage, 0, len(result.Dependencies))
	for name, dep := range result.Dependencies {
		pkgs = append(pkgs, InstalledPackage{Name: name, Version: dep.Version})
	}
	return pkgs, nil
}

func (n *NpmManager) run(dir string, args []string, progress chan<- Progress) error {
	if err := ensurePackageJSON(dir); err != nil {
		return err
	}

	cmd := exec.Command("npm", args...)
	cmd.Dir = dir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("npm: %w", err)
	}

	// stream both stdout and stderr as progress lines
	done := make(chan struct{}, 2)
	pipe := func(r interface{ Read([]byte) (int, error) }) {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) != "" {
				progress <- Progress{Line: line}
			}
		}
		done <- struct{}{}
	}
	go pipe(stdout)
	go pipe(stderr)
	<-done
	<-done

	return cmd.Wait()
}

// ensurePackageJSON creates a minimal package.json if none exists.
func ensurePackageJSON(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	pkgPath := filepath.Join(dir, "package.json")
	if _, err := os.Stat(pkgPath); err == nil {
		return nil
	}
	content := `{"name":"kb-platform","version":"1.0.0","private":true}` + "\n"
	return os.WriteFile(pkgPath, []byte(content), 0o644)
}
