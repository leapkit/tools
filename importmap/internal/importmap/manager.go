package importmap

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Auditor provides methods for analyzing dependencies for known vulnerabilities
// and checking if they are outdated.
type Auditor interface {
	Audit(context.Context, map[string]string) (map[string][]map[string]string, error)
	Outdated(context.Context, map[string]string) (map[string]string, error)
}

type modules struct {
	Map map[string]string `json:"imports"`
}

func (m *modules) Bytes() []byte {
	b, _ := json.MarshalIndent(m, "", "  ")
	return b
}

type manager struct {
	containerFolder string
	modules         modules

	bundler Bundler
	auditor Auditor
}

func (m *manager) Packages() {
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

func (m *manager) JSON() []byte {
	return m.modules.Bytes()
}

func NewManager(containerFolder string, bundler Bundler, auditor Auditor) *manager {
	m := &manager{
		containerFolder: containerFolder,
		modules: modules{
			Map: make(map[string]string),
		},
		bundler: bundler,
		auditor: auditor,
	}

	data, _ := os.ReadFile(filepath.Join(m.containerFolder, "importmap.json"))
	json.Unmarshal(data, &m.modules)

	return m
}
