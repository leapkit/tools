package importmap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (m *manager) Pin(ctx context.Context, packages ...string) error {
	result, err := m.bundler.Bundle(ctx, packages...)
	if err != nil {
		return err
	}

	pkgNames := []string{}
	for nameVersion, content := range result {
		//	get only the package name: "@scope/some-package@1.2.3" -> "@scope/some-package"
		pkgName := nameVersion
		if idx := strings.LastIndex(nameVersion, "@"); idx > 0 {
			pkgName = nameVersion[:idx]
		}

		if _, ok := m.modules.Map[pkgName]; ok {
			if err := m.Unpin(ctx, pkgName); err != nil {
				return err
			}
		}

		vendorPath := "vendor/" + nameVersion + "/bundle.min.js"
		destPath := filepath.Join(m.containerFolder, vendorPath)

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return fmt.Errorf("pin: write %s: %w", destPath, err)
		}

		if err := os.WriteFile(destPath, content, 0o644); err != nil {
			return fmt.Errorf("pin: write %s: %w", destPath, err)
		}

		m.modules.Map[pkgName] = vendorPath
		pkgNames = append(pkgNames, pkgName)
	}

	fmt.Printf("[info] pinned %s\n", strings.Join(pkgNames, ", "))

	return m.save()
}
