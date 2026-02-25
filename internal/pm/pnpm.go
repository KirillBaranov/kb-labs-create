package pm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// PnpmManager implements PackageManager using pnpm.
type PnpmManager struct{}

func (p *PnpmManager) Name() string { return "pnpm" }

func (p *PnpmManager) Install(dir string, pkgs []string, progress chan<- Progress) error {
	args := append([]string{"add", "--dir", dir}, pkgs...)
	return p.run(dir, args, progress)
}

func (p *PnpmManager) Update(dir string, pkgs []string, progress chan<- Progress) error {
	args := append([]string{"update", "--dir", dir}, pkgs...)
	return p.run(dir, args, progress)
}

func (p *PnpmManager) ListInstalled(dir string) ([]InstalledPackage, error) {
	cmd := exec.Command("pnpm", "list", "--dir", dir, "--json", "--depth=0")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("pnpm list: %w", err)
	}

	// pnpm list --json returns an array
	var results []struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(out, &results); err != nil {
		return nil, err
	}

	var pkgList []InstalledPackage
	if len(results) > 0 {
		for name, dep := range results[0].Dependencies {
			pkgList = append(pkgList, InstalledPackage{
				Name:    name,
				Version: dep.Version,
			})
		}
	}
	return pkgList, nil
}

func (p *PnpmManager) run(dir string, args []string, progress chan<- Progress) error {
	if err := ensurePackageJSON(dir); err != nil {
		return err
	}
	// ensure pnpm-workspace is NOT present (we want flat install, not workspace)
	wsPath := filepath.Join(dir, "pnpm-workspace.yaml")
	_ = wsPath // intentionally not creating it

	cmd := exec.Command("pnpm", args...)
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
		return fmt.Errorf("pnpm: %w", err)
	}

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
