package importmap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func (m *manager) Unpin(ctx context.Context, packages ...string) error {
	for _, pkg := range packages {
		path, ok := m.modules.Map[pkg]
		if !ok {
			fmt.Printf("[warn] package %s not pinned\n", pkg)
			continue
		}

		// path is "vendor/name@version/bundle.min.js"
		// so the directory to remove is the parent: "vendor/name@version"
		pkgDir := filepath.Dir(path)

		fmt.Printf("[info] unpinning %s\n", pkg)

		delete(m.modules.Map, pkg)

		fullPath := filepath.Join(m.containerFolder, pkgDir)
		if err := os.RemoveAll(fullPath); err != nil {
			fmt.Printf("[warn] failed to delete %s: %v\n", pkgDir, err)
		}
	}

	return m.save()
}
