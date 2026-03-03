// Package scaffold generates the initial .kb/kb.config.jsonc for new projects.
// The file uses JSONC (JSON with Comments) so users get inline documentation
// for every section — same pattern as tsconfig.json.
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Options controls which sections are included in the generated config.
type Options struct {
	PlatformDir string
	Services    []string // selected service IDs (e.g. "rest", "workflow")
	Plugins     []string // selected plugin IDs  (e.g. "mind", "agents")
}

// WriteProjectConfig generates .kb/kb.config.jsonc inside projectDir.
func WriteProjectConfig(projectDir string, opts Options) error {
	dir := filepath.Join(projectDir, ".kb")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create .kb dir: %w", err)
	}

	content := generate(opts)
	path := filepath.Join(dir, "kb.config.jsonc")
	return os.WriteFile(path, []byte(content), 0o644)
}

func generate(opts Options) string {
	svcSet := toSet(opts.Services)
	plugSet := toSet(opts.Plugins)

	var b strings.Builder

	b.WriteString(`{
  // ─── KB Labs Project Configuration ────────────────────────────────────
  //
  // This file configures the KB Labs platform for your project.
  // Format: JSONC (JSON with Comments) — same as tsconfig.json.
  //
  // Docs:  https://kb-labs.dev/docs/configuration
  // CLI:   kb config --help

`)

	// ── platform section ──────────────────────────────────────────────────
	b.WriteString(`  // ─── Platform ──────────────────────────────────────────────────────────
  // Connection to the platform installation (node_modules, adapters, etc.)
  "platform": {
    // Path to the platform installation directory.
    "dir": `)
	b.WriteString(quote(opts.PlatformDir))
	b.WriteString(`,

    // Adapter bindings — which packages handle storage, LLM, logging, etc.
    // Each key maps to an adapter interface; the value is the npm package
    // that implements it. You can swap adapters without changing app code.
    "adapters": {
      // LLM provider(s). Array = fallback chain, string = single provider.
      // Available: @kb-labs/adapters-openai, @kb-labs/adapters-vibeproxy
      "llm": "@kb-labs/adapters-openai",

      // Embedding model for vector search (Mind RAG).
      "embeddings": "@kb-labs/adapters-openai/embeddings",

      // File storage backend.
      "storage": "@kb-labs/adapters-fs",

      // Structured logger.
      // Available: @kb-labs/adapters-pino, @kb-labs/adapters-console
      "logger": "@kb-labs/adapters-pino"
    },

    // Plugin execution mode: "in-process" (fast, shared memory) or
    // "subprocess" (isolated, separate Node.js process per plugin).
    "execution": {
      "mode": "in-process"
    }
  },

`)

	// ── services section ──────────────────────────────────────────────────
	b.WriteString(`  // ─── Services ─────────────────────────────────────────────────────────
  // Background daemons. Enable/disable based on what you installed.
  "services": {
`)
	writeToggle(&b, "rest", "REST API daemon on port 5050.", svcSet)
	writeToggle(&b, "workflow", "Workflow engine on port 7778.", svcSet)
	writeToggle(&b, "studio", "Web UI on port 3000.", svcSet)
	b.WriteString(`  },

`)

	// ── plugins section ───────────────────────────────────────────────────
	b.WriteString(`  // ─── Plugins ──────────────────────────────────────────────────────────
  // Optional functionality. Each plugin can have its own nested config.
  "plugins": {
`)
	writePluginBlock(&b, "mind", "AI-powered code search (RAG).", plugSet, `
      // Vector store for embeddings.
      // "local" = on-disk HNSW index, "qdrant" = external Qdrant server.
      "vectorStore": "local"`)
	writePluginBlock(&b, "agents", "Autonomous agent execution.", plugSet, `
      // Max steps per agent run (prevents infinite loops).
      "maxSteps": 25`)
	writePluginBlock(&b, "ai-review", "AI code review.", plugSet, `
      // Review mode: "heuristic" (fast), "llm" (smart), "full" (both).
      "mode": "full"`)
	writePluginBlock(&b, "commit", "AI-powered commit message generation.", plugSet, `
      // Auto-stage changed files before generating commit.
      "autoStage": false`)
	b.WriteString(`  }
}
`)

	return b.String()
}

// writeToggle writes an enabled/disabled entry with a comment.
func writeToggle(b *strings.Builder, id, comment string, enabled map[string]bool) {
	val := "false"
	if enabled[id] {
		val = "true"
	}
	fmt.Fprintf(b, "    // %s\n    %s: %s,\n", comment, quote(id), val)
}

// writePluginBlock writes a plugin config object with a comment and optional
// inner settings. Disabled plugins are written commented-out style (enabled: false).
func writePluginBlock(b *strings.Builder, id, comment string, enabled map[string]bool, inner string) {
	on := enabled[id]
	fmt.Fprintf(b, "    // %s\n", comment)
	fmt.Fprintf(b, "    %s: {\n", quote(id))
	if on {
		fmt.Fprintf(b, "      \"enabled\": true,")
	} else {
		fmt.Fprintf(b, "      \"enabled\": false,")
	}
	b.WriteString(inner)
	b.WriteString("\n    },\n")
}

func quote(s string) string {
	return `"` + s + `"`
}

func toSet(ids []string) map[string]bool {
	m := make(map[string]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m
}
