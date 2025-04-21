package importmap

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

func (m *manager) Pin(ctx context.Context, pkg ...string) error {
	if len(pkg) == 0 {
		return nil
	}

	modules, err := m.generator.Generate(ctx, pkg...)
	if err != nil {
		return err
	}

	var mu sync.Mutex
	errCh := make(chan error, len(modules))
	for name, url := range modules {
		if m.modules.Map[name] != "" {
			if err := m.Unpin(ctx, name); err != nil {
				return err
			}
		}

		go func() {
			path, err := m.download(ctx, name, url)
			if err != nil {
				errCh <- err
				return
			}

			mu.Lock()
			m.modules.Map[name] = path
			mu.Unlock()

			errCh <- nil
		}()
	}

	for range modules {
		if err := <-errCh; err != nil {
			return err
		}
	}

	if err := m.write(); err != nil {
		return err
	}

	return nil
}

func (m *manager) download(ctx context.Context, pkg, url string) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	b, _ := io.ReadAll(res.Body)

	pkgReg := regexp.MustCompile(`npm:(.+)@([\d\.]+)`)

	var version string
	if match := pkgReg.FindStringSubmatch(url); len(match) == 3 {
		version += "@" + match[2]
	}

	vendorPath := filepath.Join(m.containerFolder, "vendor")
	filename := filepath.Join(vendorPath, pkg+version+".js")

	fmt.Printf("[info] downloading %s\n", filename)

	if err := os.MkdirAll(filepath.Dir(filename), os.ModePerm); err != nil {
		return "", err
	}

	f, err := os.Create(filename)
	if err != nil {
		return "", err
	}

	defer f.Close()

	f.Write(b)

	prefix := strings.TrimPrefix(m.containerFolder, "./") + "/"

	return strings.TrimPrefix(filename, prefix), nil
}

func (m *manager) write() error {
	outputPath := filepath.Join(m.containerFolder, "importmap.json")
	if err := os.MkdirAll(filepath.Dir(outputPath), os.ModePerm); err != nil {
		return err
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}

	f.Write(m.modules.Bytes())

	return f.Close()
}
