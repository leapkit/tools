package importmap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

func (m *manager) Unpin(ctx context.Context, packages ...string) error {
	for _, pkg := range packages {
		url, ok := m.modules.Map[pkg]
		if !ok {
			fmt.Printf("[warn] package %s not pinned\n", pkg)
			continue
		}

		pkgDir, version := extractPackageDirAndVersion(url)
		if pkgDir == "" {
			fmt.Printf("[warn] invalid package path: %s\n", url)
			continue
		}

		fmt.Printf("[info] unpinning %s@%s\n", pkg, version)

		// 1. Remove from imports
		delete(m.modules.Map, pkg)

		// 2. Remove scope for this package
		// Scope key format: "vendor/@org/package@1.2.3/"
		scopeKey := pkgDir + "/"
		delete(m.modules.Scopes, scopeKey)

		// 3. Delete entire package directory
		fullPath := filepath.Join(m.containerFolder, pkgDir)
		if err := os.RemoveAll(fullPath); err != nil {
			fmt.Printf("[warn] failed to delete %s: %v\n", pkgDir, err)
		} else {
			fmt.Printf("[info] deleted %s\n", pkgDir)
		}
	}

	return m.write()
}

// extractPackageDirAndVersion extracts package directory and version from vendor path
// "vendor/@org/package@1.2.3/dist/index.js" → "vendor/@org/package@1.2.3", "1.2.3"
// "vendor/pkg@4.5.6/index.js" → "vendor/pkg@4.5.6", "4.5.6"
func extractPackageDirAndVersion(vendorPath string) (pkgDir, version string) {
	// Match pattern: vendor/[optional @org/]package@version/...
	// Capture everything up to and including @version
	re := regexp.MustCompile(`^(vendor/(?:@[^/]+/)?[^/@]+@([\d\.]+))`)
	matches := re.FindStringSubmatch(vendorPath)

	if len(matches) < 3 {
		return "", ""
	}

	return matches[1], matches[2]
}
