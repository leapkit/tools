package importmap

import (
	"context"
	"regexp"
	"strings"
)

func (m *manager) Pristine(ctx context.Context) error {
	pkgReg := regexp.MustCompile(`\/(.+@[\d\.]+).js$`)

	var pkg []string
	for _, url := range m.modules.Map {
		if !strings.HasPrefix(url, "vendor/") {
			continue
		}

		match := pkgReg.FindStringSubmatch(url)

		if len(match) != 2 {
			continue
		}

		pkg = append(pkg, match[1])
	}

	return m.Pin(ctx, pkg...)
}
