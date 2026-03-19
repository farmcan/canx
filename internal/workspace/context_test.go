package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsReadmeAndAgents(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("readme"), 0o644); err != nil {
		t.Fatalf("WriteFile(README) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("agents"), 0o644); err != nil {
		t.Fatalf("WriteFile(AGENTS) error = %v", err)
	}

	ctx, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if ctx.Readme == "" || ctx.Agents == "" {
		t.Fatal("expected readme and agents content")
	}
}

func TestLoadCollectsMarkdownDocs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("readme"), 0o644); err != nil {
		t.Fatalf("WriteFile(README) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("agents"), 0o644); err != nil {
		t.Fatalf("WriteFile(AGENTS) error = %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatalf("Mkdir(docs) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "a.md"), []byte("doc-a"), 0o644); err != nil {
		t.Fatalf("WriteFile(doc) error = %v", err)
	}

	ctx, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(ctx.Docs) != 1 {
		t.Fatalf("Load() docs = %d, want 1", len(ctx.Docs))
	}
}

func TestLoadAllowsMissingAgentsFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("readme"), 0o644); err != nil {
		t.Fatalf("WriteFile(README) error = %v", err)
	}

	ctx, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if ctx.Readme == "" {
		t.Fatal("expected readme content")
	}
}

func TestLoadReadsPatternsFileWhenPresent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("readme"), 0o644); err != nil {
		t.Fatalf("WriteFile(README) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".canx"), 0o755); err != nil {
		t.Fatalf("MkdirAll(.canx) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".canx", "patterns.md"), []byte("- failing test: make test"), 0o644); err != nil {
		t.Fatalf("WriteFile(patterns) error = %v", err)
	}

	ctx, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if ctx.Patterns == "" {
		t.Fatal("expected patterns content")
	}
}
