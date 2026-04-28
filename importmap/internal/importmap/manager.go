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

// GeneratorResult contains the imports, scopes, and staticDeps returned by a package generator.
type GeneratorResult interface {
	GetImports() map[string]string
	GetScopes() map[string]map[string]string
	GetStaticDeps() []string
}

// Generator defines an interface for generating a map of dependencies from a list of packages.
// It returns a result that can provide imports and scopes for the given packages.
type Generator interface {
	Generate(context.Context, ...string) (any, error)
}

// Auditor provides methods for analyzing dependencies for known vulnerabilities
// and checking if they are outdated.
type Auditor interface {
	Audit(context.Context, map[string]string) (map[string][]map[string]string, error)
	Outdated(context.Context, map[string]string) (map[string]string, error)
}

type modules struct {
	Map    map[string]string            `json:"imports"`
	Scopes map[string]map[string]string `json:"scopes,omitempty"`
}

func (m *modules) Bytes() []byte {
	b, _ := json.MarshalIndent(m, "", "  ")
	return b
}

type manager struct {
	containerFolder string
	modules         modules

	generator Generator
	auditor   Auditor
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

func NewManager(containerFolder string, generator Generator, auditor Auditor) *manager {
	m := &manager{
		containerFolder: containerFolder,
		modules: modules{
			Map:    make(map[string]string),
			Scopes: make(map[string]map[string]string),
		},
		auditor:   auditor,
		generator: generator,
	}

	data, _ := os.ReadFile(filepath.Join(m.containerFolder, "importmap.json"))
	json.Unmarshal(data, &m.modules)

	return m
}
