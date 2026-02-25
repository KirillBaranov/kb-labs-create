# kb-create

> **One-command installer for the KB Labs platform.** Download, configure and launch the full KB Labs stack in seconds — no manual setup required.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://golang.org/)
[![KB Labs Platform](https://img.shields.io/badge/KB_Labs-Platform-blue.svg)](https://github.com/kb-labs)
[![Release](https://img.shields.io/github/v/release/kb-labs/create)](https://github.com/kb-labs/create/releases)

## Overview

`kb-create` is a standalone Go binary that installs and manages the KB Labs platform on your machine. It is completely independent — no Node.js, no existing KB Labs installation required to run it.

**Key features:**
- ✅ **Interactive TUI wizard** — pick services and plugins with checkboxes
- ✅ **Silent mode** — `--yes` for CI/scripted environments
- ✅ **Isolated platform directory** — platform lives separately from your project
- ✅ **CWD binding** — all CLI calls and artifacts are scoped to your project folder
- ✅ **Update with diff** — see exactly what changes before applying
- ✅ **Install logs** — every run is logged, follow with `--follow`
- ✅ **pnpm-first** — uses pnpm if available, falls back to npm

## Quick Start

### Install kb-create

```bash
curl https://raw.githubusercontent.com/kb-labs/create/main/install.sh | sh
```

This downloads the correct binary for your OS/arch and places it in `~/.local/bin/kb-create`.

### Create a project

```bash
kb-create my-project
```

The wizard guides you through:
1. Platform directory (where node_modules live)
2. Project directory (your actual work folder)
3. Services to install (REST API, Workflow, Studio)
4. Plugins to install (mind, agents, ai-review, commit)

### Silent install with defaults

```bash
kb-create my-project --yes
```

Installs core + default services/plugins without any prompts.

## How It Works

```
kb-create my-project
        │
        ▼
   Interactive wizard
   ─────────────────────────────────────────────────
   Platform dir:  ~/kb-platform
   Project cwd:   ~/projects/my-project

   ◉ REST API       REST daemon (port 5050)
   ◉ Workflow       Workflow engine (port 7778)
   ○ Studio         Web UI (port 3000)

   ◉ mind           AI code search (RAG)
   ○ agents         Autonomous agents
   ─────────────────────────────────────────────────
        │
        ▼
   npm/pnpm install @kb-labs/* packages
   into ~/kb-platform/node_modules/
        │
        ▼
   Write ~/kb-platform/.kb/kb.config.json
   { "platform": "~/kb-platform", "cwd": "~/projects/my-project" }
        │
        ▼
   ✅ Done — run: kb dev:start
```

### Platform vs Project separation

The platform (node_modules) lives in one place; your project files live elsewhere. The KB Labs CLI reads `.kb/kb.config.json` and `chdir`s into `cwd` before executing any command — so all artifacts, logs and outputs land in your project folder.

```
~/kb-platform/          ← platform installation
  node_modules/
  package.json
  .kb/
    kb.config.json      ← cwd binding lives here
    logs/               ← install logs

~/projects/my-project/  ← your project
  .kb/                  ← runtime artifacts (created by platform)
```

## Commands

### `kb-create [project-dir]`

Default command. Launches the interactive wizard (or silent install with `--yes`).

```bash
kb-create my-project
kb-create my-project --yes
kb-create my-project --platform ~/custom/platform/path
```

| Flag | Description |
|------|-------------|
| `-y, --yes` | Skip wizard, install with defaults |
| `--platform <dir>` | Override default platform directory |

### `kb-create update`

Compares the current manifest against the installed snapshot. Shows a diff, asks for confirmation, then applies updates.

```bash
kb-create update
kb-create update --platform ~/kb-platform
```

**Example output:**
```
Checking for updates...

  + @kb-labs/new-plugin         (new)
  ↑ @kb-labs/cli-bin            (update available)
  - @kb-labs/old-package        (removed from manifest)

Apply updates? [Y/n]
```

### `kb-create status`

Shows what is currently installed and the platform configuration.

```bash
kb-create status
kb-create status --platform ~/kb-platform
```

**Example output:**
```
  Platform:   ~/kb-platform
  Project:    ~/projects/my-project
  PM:         pnpm
  Installed:  2026-02-25 10:00
  Manifest:   1.0.0

  Core packages:
    ● @kb-labs/cli-bin
    ● @kb-labs/sdk

  Services:
    ● rest       REST API daemon (port 5050)
    ● workflow   Workflow engine (port 7778)

  Plugins:
    ● mind       AI-powered code search (RAG)
```

### `kb-create logs`

Prints the most recent install log.

```bash
kb-create logs                       # print last log
kb-create logs --follow              # stream in real time (like tail -f)
kb-create logs --platform ~/kb-platform
```

## Installation

### curl | sh (recommended)

```bash
curl https://raw.githubusercontent.com/kb-labs/create/main/install.sh | sh
```

Installs to `~/.local/bin/kb-create`. No `sudo` needed.

### Manual download

Download the correct binary from [GitHub Releases](https://github.com/kb-labs/create/releases/latest):

| Platform | Binary |
|----------|--------|
| macOS Apple Silicon | `kb-create-darwin-arm64` |
| macOS Intel | `kb-create-darwin-amd64` |
| Linux x86_64 | `kb-create-linux-amd64` |
| Linux ARM64 | `kb-create-linux-arm64` |

```bash
# Example for macOS Apple Silicon
curl -fsSL https://github.com/kb-labs/create/releases/latest/download/kb-create-darwin-arm64 \
  -o ~/.local/bin/kb-create
chmod +x ~/.local/bin/kb-create
```

### Build from source

```bash
git clone https://github.com/kb-labs/create
cd create
go build -o kb-create .
```

Requires Go 1.21+.

## Manifest

The list of installable packages is defined in [`internal/manifest/manifest.json`](internal/manifest/manifest.json) and embedded into the binary at build time. To update the package list, edit that file and rebuild.

**Structure:**

```json
{
  "version": "1.0.0",
  "registryUrl": "https://registry.npmjs.org",
  "core": [
    { "name": "@kb-labs/cli-bin" }
  ],
  "services": [
    { "id": "rest", "pkg": "@kb-labs/rest-api", "description": "...", "default": true }
  ],
  "plugins": [
    { "id": "mind", "pkg": "@kb-labs/mind", "description": "...", "default": true }
  ]
}
```

**Extensibility:** `manifest.Loader` supports a fallback chain — Remote URL → Local override file → Embedded JSON. When a remote registry endpoint is available, set `LoadOptions.RemoteURL` to always fetch the latest manifest without rebuilding the binary.

## Architecture

```
kb-labs-create/
├── main.go                        ← entrypoint, injects build-time version
├── manifest.json                  ← canonical package list (see internal/manifest/)
├── cmd/
│   ├── root.go                    ← cobra root, --version, Execute()
│   ├── create.go                  ← default command: wizard → install
│   ├── update.go                  ← diff → confirm → npm update
│   ├── status.go                  ← read config, pretty-print
│   └── logs.go                    ← cat / tail -f install log
└── internal/
    ├── manifest/
    │   ├── types.go               ← Manifest, Package, Component structs
    │   └── loader.go              ← Load() with fallback chain + //go:embed
    ├── pm/
    │   ├── pm.go                  ← PackageManager interface + Detect()
    │   ├── npm.go                 ← NpmManager
    │   └── pnpm.go                ← PnpmManager
    ├── wizard/
    │   └── wizard.go              ← Bubble Tea TUI (3-stage: dirs → options → confirm)
    ├── installer/
    │   └── installer.go           ← Install(), Diff(), Update()
    ├── config/
    │   └── config.go              ← Read/Write versioned PlatformConfig
    └── logger/
        └── logger.go              ← io.MultiWriter(stderr + file)
```

### Extension points

| Point | How to extend |
|-------|--------------|
| **New packages/services/plugins** | Edit `internal/manifest/manifest.json`, rebuild |
| **Remote manifest** | Set `manifest.LoadOptions.RemoteURL` — fallback to embedded if unreachable |
| **New package manager** | Implement `pm.PackageManager` interface, add to `pm.Detect()` |
| **Config migrations** | Increment `configVersion`, add case in `config.Read()` |
| **Wizard steps** | Add a new `stage` const and handler in `wizard.go` |

## FAQ

### Q: Do I need Node.js installed?

**A:** Yes — `kb-create` itself is a Go binary with no Node.js dependency, but it installs `@kb-labs/*` npm packages, so Node.js and npm (or pnpm) must be available on the system.

### Q: Where should I install the platform?

**A:** Anywhere you like — `~/kb-platform` is the default. The platform directory is independent from your project. You can have one platform installation shared across multiple projects (each with its own `cwd` binding), or a dedicated installation per project.

### Q: Can I run kb-create in CI?

**A:** Yes:
```bash
kb-create /workspace/my-project --yes --platform /opt/kb-platform
```

### Q: How do I update the platform later?

**A:**
```bash
kb-create update --platform ~/kb-platform
```

### Q: What if pnpm is not installed?

**A:** `kb-create` automatically falls back to npm. To use pnpm, install it first:
```bash
npm install -g pnpm
```

### Q: Can I customise what gets installed?

**A:** Yes — in wizard mode, use space to toggle any service or plugin. In silent mode, all items marked `"default": true` in the manifest are installed. For fine-grained control, edit the manifest and rebuild.

### Q: The binary shows version `dev` — is that normal?

**A:** Only when built with `go build .` directly. Official releases from GitHub have proper version strings injected by goreleaser via `-ldflags`. Check with `kb-create --version`.

## Development

```bash
# Clone
git clone https://github.com/kb-labs/create
cd create

# Install dependencies
go mod download

# Build
go build -o kb-create .

# Run tests
go test ./...

# Vet
go vet ./...

# Build for all platforms (requires goreleaser)
goreleaser build --snapshot --clean
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

## Support & Resources

- **Issues**: [Report bugs →](https://github.com/kb-labs/create/issues)
- **Discussions**: [Ask questions →](https://github.com/kb-labs/discussions)
- **KB Labs Platform**: [Main repository →](https://github.com/kb-labs)

## License

MIT — see [LICENSE](LICENSE) for details.
