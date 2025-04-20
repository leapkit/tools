package importmap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func (m *manager) Unpin(ctx context.Context, packages ...string) error {
	for _, pkg := range packages {
		url, ok := m.modules.Map[pkg]
		if !ok {
			continue
		}

		// vendor/pkg@1.2.3.js -> 1.2.3
		pkgVersionReg := regexp.MustCompile(`.+@([\d\.]+).js$`)

		var version string
		if match := pkgVersionReg.FindStringSubmatch(url); len(match) == 2 {
			version += "@" + match[1]
		}

		fmt.Println("[info] unpinning", pkg+version)

		pkgPath := strings.TrimPrefix(m.modules.Map[pkg], "vendor/")
		parts := strings.Split(pkgPath, "/")

		os.RemoveAll(filepath.Join(m.containerFolder, "vendor", parts[0]))

		delete(m.modules.Map, pkg)
	}

	if err := m.write(); err != nil {
		return err
	}

	return nil
}
