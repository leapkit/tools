package importmap

import (
	"context"
	"regexp"
	"strings"
)

func (m *manager) Update(ctx context.Context) error {
	pkgReg := regexp.MustCompile(`\/.+@([\d\.]+).js$`)
	pkgVersion := map[string]string{}
	for pkg, url := range m.modules.Map {
		if !strings.HasPrefix(url, "vendor/") {
			continue
		}

		match := pkgReg.FindStringSubmatch(url)
		if len(match) != 2 {
			continue
		}

		pkgVersion[pkg] = match[1]
	}

	outdatedPackages, err := m.auditor.Outdated(ctx, pkgVersion)
	if err != nil {
		return err
	}

	var update []string
	for pkg, version := range outdatedPackages {
		update = append(update, pkg+"@"+version)
	}

	return m.Pin(ctx, update...)
}
