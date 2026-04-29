package importmap

import (
	"context"
	"regexp"
	"strings"
)

func (m *manager) Pristine(ctx context.Context) error {
	pkgReg := regexp.MustCompile(`^vendor/((?:@[^/]+/)?[^/@]+@[\d\.]+)`)

	var pkg []string
	unique := make(map[string]bool)

	for _, url := range m.modules.Map {
		if !strings.HasPrefix(url, "vendor/") {
			continue
		}

		match := pkgReg.FindStringSubmatch(url)
		if len(match) != 2 {
			continue
		}

		pkgName := match[1]
		if !unique[pkgName] {
			pkg = append(pkg, pkgName)
			unique[pkgName] = true
		}
	}

	return m.Pin(ctx, pkg...)
}
