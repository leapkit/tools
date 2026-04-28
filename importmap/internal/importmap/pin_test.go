package importmap_test

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.leapkit.dev/tools/importmap/internal/importmap"
)

// mockScopedGenerator returns package with scopes (new structure)
type mockScopedGenerator struct{}

func (g *mockScopedGenerator) Generate(ctx context.Context, packages ...string) (any, error) {
	imports := map[string]string{
		"@org/package-a": "https://ga.jspm.io/npm:@org/package-a@1.0.0/dist/index.js",
	}

	scopes := map[string]map[string]string{
		"https://ga.jspm.io/": {
			"package-b":         "https://ga.jspm.io/npm:package-b@2.0.0/dist/index.js",
			"@org/package-c/utils": "https://ga.jspm.io/npm:@org/package-c@3.0.0/dist/utils/index.js",
		},
	}

	// staticDeps contains ALL files to download (main + dependencies + internal files)
	staticDeps := []string{
		"https://ga.jspm.io/npm:@org/package-a@1.0.0/dist/index.js",
		"https://ga.jspm.io/npm:@org/package-a@1.0.0/_/abc123.js",        // internal file
		"https://ga.jspm.io/npm:@org/package-a@1.0.0/helpers/index.js", // extra export path
		"https://ga.jspm.io/npm:package-b@2.0.0/dist/index.js",
		"https://ga.jspm.io/npm:@org/package-c@3.0.0/dist/utils/index.js",
	}

	return &mockResult{
		imports:    imports,
		scopes:     scopes,
		staticDeps: staticDeps,
	}, nil
}

// mockHTTPTransport intercepts HTTP requests for testing
type mockHTTPTransport struct{}

func (m *mockHTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Return mock file content for any JSPM download
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("// mock file content")),
	}, nil
}

type mockResult struct {
	imports    map[string]string
	scopes     map[string]map[string]string
	staticDeps []string
}

func (r *mockResult) GetImports() map[string]string {
	return r.imports
}

func (r *mockResult) GetScopes() map[string]map[string]string {
	return r.scopes
}

func (r *mockResult) GetStaticDeps() []string {
	return r.staticDeps
}

func TestPackageScopedStructure(t *testing.T) {
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)

	// Mock HTTP client to avoid real downloads
	http.DefaultClient = &http.Client{
		Transport: &mockHTTPTransport{},
	}
	t.Cleanup(func() { http.DefaultClient = &http.Client{} })

	m := importmap.NewManager(".", &mockScopedGenerator{}, &mockAuditor{})

	ctx := context.Background()
	err := m.Pin(ctx, "@org/package-a")
	if err != nil {
		t.Fatalf("Pin failed: %v", err)
	}

	// Verify main file exists with nested structure
	mainFile := filepath.Join(tmpDir, "vendor/@org/package-a@1.0.0/dist/index.js")
	if _, err := os.Stat(mainFile); err != nil {
		t.Errorf("Main file not found: %s", mainFile)
	}

	// Verify scope files exist in scopes/ subdirectory
	scopeFile1 := filepath.Join(tmpDir, "vendor/@org/package-a@1.0.0/scopes/package-b@2.0.0/dist/index.js")
	if _, err := os.Stat(scopeFile1); err != nil {
		t.Errorf("Scope file not found: %s", scopeFile1)
	}

	scopeFile2 := filepath.Join(tmpDir, "vendor/@org/package-a@1.0.0/scopes/@org/package-c@3.0.0/dist/utils/index.js")
	if _, err := os.Stat(scopeFile2); err != nil {
		t.Errorf("Scope file with nested path not found: %s", scopeFile2)
	}

	// Verify internal files (_/) exist in package directory
	internalFile := filepath.Join(tmpDir, "vendor/@org/package-a@1.0.0/_/abc123.js")
	if _, err := os.Stat(internalFile); err != nil {
		t.Errorf("Internal file not found: %s", internalFile)
	}

	// Verify extra export paths exist in package directory
	helperFile := filepath.Join(tmpDir, "vendor/@org/package-a@1.0.0/helpers/index.js")
	if _, err := os.Stat(helperFile); err != nil {
		t.Errorf("Helper file not found: %s", helperFile)
	}

	// Verify importmap.json structure
	data, err := os.ReadFile(filepath.Join(tmpDir, "importmap.json"))
	if err != nil {
		t.Fatalf("Failed to read importmap.json: %v", err)
	}

	jsonStr := string(data)

	// Check imports
	if !strings.Contains(jsonStr, `"@org/package-a": "vendor/@org/package-a@1.0.0/dist/index.js"`) {
		t.Errorf("Expected import entry not found in:\n%s", jsonStr)
	}

	// Check scopes
	if !strings.Contains(jsonStr, `"vendor/@org/package-a@1.0.0/"`) {
		t.Errorf("Expected scope key not found in:\n%s", jsonStr)
	}

	if !strings.Contains(jsonStr, `"package-b": "vendor/@org/package-a@1.0.0/scopes/package-b@2.0.0/dist/index.js"`) {
		t.Errorf("Expected scope value not found in:\n%s", jsonStr)
	}

	// Verify internal files (_/) are NOT in importmap (not in scopes)
	if strings.Contains(jsonStr, `_/abc123.js`) {
		t.Errorf("Internal file should not be in importmap:\n%s", jsonStr)
	}
}

func TestUnpinDeletesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)

	// Mock HTTP client to avoid real downloads
	http.DefaultClient = &http.Client{
		Transport: &mockHTTPTransport{},
	}
	t.Cleanup(func() { http.DefaultClient = &http.Client{} })

	m := importmap.NewManager(".", &mockScopedGenerator{}, &mockAuditor{})

	ctx := context.Background()

	// Pin
	err := m.Pin(ctx, "@org/package-a")
	if err != nil {
		t.Fatalf("Pin failed: %v", err)
	}

	pkgDir := filepath.Join(tmpDir, "vendor/@org/package-a@1.0.0")
	if _, err := os.Stat(pkgDir); err != nil {
		t.Fatalf("Package directory not created: %s", pkgDir)
	}

	// Unpin
	err = m.Unpin(ctx, "@org/package-a")
	if err != nil {
		t.Fatalf("Unpin failed: %v", err)
	}

	// Verify directory deleted
	if _, err := os.Stat(pkgDir); !os.IsNotExist(err) {
		t.Errorf("Package directory should be deleted: %s", pkgDir)
	}

	// Verify removed from importmap
	data, err := os.ReadFile(filepath.Join(tmpDir, "importmap.json"))
	if err != nil {
		t.Fatalf("Failed to read importmap.json: %v", err)
	}

	jsonStr := string(data)
	if strings.Contains(jsonStr, "@org/package-a") {
		t.Errorf("Package should be removed from importmap:\n%s", jsonStr)
	}
}

func TestPristineExtractsPackages(t *testing.T) {
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)

	// Mock HTTP client to avoid real downloads
	http.DefaultClient = &http.Client{
		Transport: &mockHTTPTransport{},
	}
	t.Cleanup(func() { http.DefaultClient = &http.Client{} })

	m := importmap.NewManager(".", &mockScopedGenerator{}, &mockAuditor{})

	ctx := context.Background()

	// Pin a package to create importmap
	err := m.Pin(ctx, "@org/package-a")
	if err != nil {
		t.Fatalf("Pin failed: %v", err)
	}

	// Remove vendor directory (simulate missing files)
	os.RemoveAll(filepath.Join(tmpDir, "vendor"))

	// Run Pristine - should re-download files based on importmap.json
	err = m.Pristine(ctx)
	if err != nil {
		t.Fatalf("Pristine failed: %v", err)
	}

	// Verify files re-downloaded
	mainFile := filepath.Join(tmpDir, "vendor/@org/package-a@1.0.0/dist/index.js")
	if _, err := os.Stat(mainFile); err != nil {
		t.Errorf("Pristine should re-download main file: %s", mainFile)
	}

	scopeFile := filepath.Join(tmpDir, "vendor/@org/package-a@1.0.0/scopes/package-b@2.0.0/dist/index.js")
	if _, err := os.Stat(scopeFile); err != nil {
		t.Errorf("Pristine should re-download scope file: %s", scopeFile)
	}
}
