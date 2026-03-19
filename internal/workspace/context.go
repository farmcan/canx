package workspace

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

type Context struct {
	Root   string
	Readme string
	Agents string
	Patterns string
	Docs   []Document
}

type Document struct {
	Path    string
	Content string
}

func Load(root string) (Context, error) {
	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		return Context{}, err
	}

	agents, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil && !os.IsNotExist(err) {
		return Context{}, err
	}

	ctx := Context{
		Root:   root,
		Readme: string(readme),
		Agents: string(agents),
	}
	patterns, err := os.ReadFile(filepath.Join(root, ".canx", "patterns.md"))
	if err != nil && !os.IsNotExist(err) {
		return Context{}, err
	}
	ctx.Patterns = string(patterns)

	docsRoot := filepath.Join(root, "docs")
	_ = filepath.WalkDir(docsRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}

		relPath, relErr := filepath.Rel(root, path)
		if relErr != nil {
			relPath = path
		}

		ctx.Docs = append(ctx.Docs, Document{
			Path:    relPath,
			Content: string(content),
		})
		return nil
	})

	sort.Slice(ctx.Docs, func(i, j int) bool {
		return ctx.Docs[i].Path < ctx.Docs[j].Path
	})

	return ctx, nil
}
