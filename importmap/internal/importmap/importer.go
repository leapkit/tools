package importmap

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	"go.leapkit.dev/tools/importmap/internal/jspm"
	"go.leapkit.dev/tools/importmap/internal/npm"
)

var (
	importer = jspm.Client(
		jspm.WithEnv(env),
		jspm.WithProvider(provider),
	)
)

type modules struct {
	Map map[string]string `json:"imports"`
}

func (m *modules) Bytes() []byte {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		fmt.Println("[error] error encoding modules")
	}

	return b
}

type importManager struct {
	modules modules
}

func (m *importManager) Pin(ctx context.Context, pkg ...string) error {
	modules, err := importer.Generate(ctx, pkg...)
	if err != nil {
		return err
	}

	errCh := make(chan error, len(modules))
	mu := sync.Mutex{}
	for name, url := range modules {
		if m.modules.Map[name] != "" {
			if err := m.Unpin(ctx, name); err != nil {
				return err
			}
		}

		go func() {
			path, err := m.download(ctx, name, url)
			defer func() {
				errCh <- err
			}()

			if err != nil {
				return
			}

			mu.Lock()
			m.modules.Map[name] = path
			mu.Unlock()
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

// download fetches a module from a URL and saves it to the container folder.
// This function returns the output path of the downloaded module.
func (m *importManager) download(ctx context.Context, pkg, url string) (string, error) {
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

	vendorPath := filepath.Join(containerFolder, "vendor")
	filename := filepath.Join(vendorPath, pkg+version+".js")

	fmt.Printf("[info] downloading %s\n", pkg+version)

	if err := os.MkdirAll(filepath.Dir(filename), os.ModePerm); err != nil {
		return "", err
	}

	f, err := os.Create(filename)
	if err != nil {
		return "", err
	}

	f.Write(b)

	prefix := strings.TrimPrefix(containerFolder, "./") + "/"

	return strings.TrimPrefix(filename, prefix), f.Close()
}

func (m *importManager) write() error {
	outputPath := filepath.Join(containerFolder, "importmap.json")
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}

	f.Write(m.modules.Bytes())

	return f.Close()

}

func (m *importManager) Unpin(ctx context.Context, pkg ...string) error {
	for _, p := range pkg {
		url, ok := m.modules.Map[p]
		if !ok {
			continue
		}

		pkgReg := regexp.MustCompile(`\/(.+)@([\d\.]+).js`)

		var version string
		if match := pkgReg.FindStringSubmatch(url); len(match) == 3 {
			version += "@" + match[2]
		}

		fmt.Printf("[info] unpinning %s\n", p+version)

		pkgPath := strings.TrimPrefix(m.modules.Map[p], "vendor/")
		parts := strings.Split(pkgPath, "/")

		os.RemoveAll(filepath.Join(containerFolder, "vendor", parts[0]))

		delete(m.modules.Map, p)
	}

	if err := m.write(); err != nil {
		return err
	}

	return nil
}

func (m *importManager) Update(ctx context.Context) error {
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

	var update []string
	for pkg, version := range npm.OutdatedPackages(ctx, pkgVersion) {
		update = append(update, pkg+"@"+version)
	}

	return m.Pin(ctx, update...)
}

func (m *importManager) OutdatedPackages(ctx context.Context) {
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

	outdated := npm.OutdatedPackages(ctx, pkgVersion)

	var maxTrailSpace int
	var packages []string
	for k := range outdated {
		packages = append(packages, k)
		maxTrailSpace = max(maxTrailSpace, len(k))
	}

	slices.Sort(packages)

	for _, k := range packages {
		trailingSpaces := strings.Repeat(" ", max(maxTrailSpace-len(k), 0))
		fmt.Printf("  %s%s pinned: %s, latest: %s\n", k, trailingSpaces, pkgVersion[k], outdated[k])
	}
}

func (m *importManager) Packages() {
	var maxTrailSpace int
	var packages []string
	for k := range m.modules.Map {
		packages = append(packages, k)
		maxTrailSpace = max(maxTrailSpace, len(k))
	}

	slices.Sort(packages)

	for _, k := range packages {
		trailingSpaces := strings.Repeat(" ", max(maxTrailSpace-len(k), 0))
		fmt.Printf("  %s%s to: %s\n", k, trailingSpaces, m.modules.Map[k])
	}
}

func (m *importManager) Refresh(ctx context.Context) error {
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

func newManager() *importManager {
	m := &importManager{
		modules: modules{
			Map: make(map[string]string),
		},
	}

	data, _ := os.ReadFile(filepath.Join(containerFolder, "importmap.json"))
	json.Unmarshal(data, &m.modules)

	return m
}
