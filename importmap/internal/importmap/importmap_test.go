package importmap_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"go.leapkit.dev/tools/importmap/internal/importmap"
)

func TestImportMapPin(t *testing.T) {
	os.Chdir(t.TempDir())

	http.DefaultClient = &http.Client{
		Transport: &mockedRoundTripper{},
	}

	t.Cleanup(func() { http.DefaultClient = &http.Client{} })

	mockedGenerator := &mockGenerator{}
	mockAuditor := &mockAuditor{}

	t.Run("correct pinning a package", func(t *testing.T) {
		t.Cleanup(func() {
			os.Remove("importmap.json")
			os.RemoveAll("vendor")
		})

		ctx := context.Background()

		m := importmap.NewManager(".", mockedGenerator, mockAuditor)
		err := m.Pin(ctx, "pkg")
		if err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		expected := `{ "imports": { "pkg": "vendor/pkg@1.0.0/index.js" } }`
		current := strings.Join(strings.Fields(string(m.JSON())), " ")

		if current != expected {
			t.Errorf("Expected %q, got %q", expected, current)
		}

		if _, err := os.Stat("vendor/pkg@1.0.0/index.js"); err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}
	})

	t.Run("correct pinning multiple packages", func(t *testing.T) {
		t.Cleanup(func() {
			os.Remove("importmap.json")
			os.RemoveAll("vendor")
		})

		ctx := context.Background()

		m := importmap.NewManager(".", mockedGenerator, mockAuditor)
		err := m.Pin(ctx, "@pkg/one", "@pkg/two", "@pkg/three")
		if err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		// modules must be sorted alphabetically.
		expected := `
		{
			"imports": {
				"@pkg/one": "vendor/@pkg/one@1.0.0/index.js",
				"@pkg/three": "vendor/@pkg/three@1.0.0/index.js",
				"@pkg/two": "vendor/@pkg/two@1.0.0/index.js"
			}
		}`

		expected = strings.Join(strings.Fields(expected), " ")
		current := strings.Join(strings.Fields(string(m.JSON())), " ")

		if current != expected {
			t.Errorf("Expected %q, got %q", expected, current)
		}
	})

	t.Run("correct pin no packages must no create any importmap.json", func(t *testing.T) {
		t.Cleanup(func() {
			os.Remove("importmap.json")
			os.RemoveAll("vendor")
		})

		ctx := context.Background()

		m := importmap.NewManager(".", mockedGenerator, mockAuditor)
		err := m.Pin(ctx)
		if err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		if !bytes.Contains(m.JSON(), []byte("")) {
			t.Errorf("Expected empty JSON, got %q", string(m.JSON()))
		}
	})

	t.Run("correct pinning a package with a version", func(t *testing.T) {
		t.Cleanup(func() {
			os.Remove("importmap.json")
			os.RemoveAll("vendor")
		})

		ctx := context.Background()

		m := importmap.NewManager(".", mockedGenerator, mockAuditor)
		err := m.Pin(ctx, "pkg@1.2.0")
		if err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		expected := `{ "imports": { "pkg": "vendor/pkg@1.2.0/index.js" } }`
		current := strings.Join(strings.Fields(string(m.JSON())), " ")

		if current != expected {
			t.Errorf("Expected %q, got %q", expected, current)
		}
	})

	t.Run("correct pinning a package with a new version", func(t *testing.T) {
		t.Cleanup(func() {
			os.Remove("importmap.json")
			os.RemoveAll("vendor")
		})

		ctx := context.Background()

		m := importmap.NewManager(".", mockedGenerator, mockAuditor)
		err := m.Pin(ctx, "pkg")
		if err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		expected := `{ "imports": { "pkg": "vendor/pkg@1.0.0/index.js" } }`
		current := strings.Join(strings.Fields(string(m.JSON())), " ")

		if current != expected {
			t.Errorf("Expected %q, got %q", expected, current)
		}

		err = m.Pin(ctx, "pkg@1.2.0")
		if err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		expected = `{ "imports": { "pkg": "vendor/pkg@1.2.0/index.js" } }`
		current = strings.Join(strings.Fields(string(m.JSON())), " ")

		if current != expected {
			t.Errorf("Expected %q, got %q", expected, current)
		}
	})

	t.Run("incorrect pinning a nonexistent package should return an error", func(t *testing.T) {
		t.Cleanup(func() {
			os.Remove("importmap.json")
			os.RemoveAll("vendor")
		})

		ctx := context.Background()
		ctx = inCtx(ctx, "generate_no_exists", true)

		m := importmap.NewManager(".", mockedGenerator, mockAuditor)
		err := m.Pin(ctx, "pkg")
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		expected := "Error: Unable to resolve npm:pkg@ to a valid version"
		if err.Error() != expected {
			t.Errorf("Expected %q, got %q", expected, err.Error())
		}
	})

}

func TestImportMapUnpin(t *testing.T) {
	os.Chdir(t.TempDir())

	http.DefaultClient = &http.Client{
		Transport: &mockedRoundTripper{},
	}

	t.Cleanup(func() {
		http.DefaultClient = &http.Client{}
		os.Remove("importmap.json")
		os.RemoveAll("vendor")
	})

	mockedGenerator := &mockGenerator{}
	mockAuditor := &mockAuditor{}

	ctx := context.Background()

	m := importmap.NewManager(".", mockedGenerator, mockAuditor)
	m.Pin(ctx, "pkg")

	if _, err := os.Stat("vendor/pkg@1.0.0"); err != nil {
		t.Errorf("Expected directory to exist, got error %v", err)
	}

	t.Run("correct unpinning a package", func(t *testing.T) {
		t.Cleanup(func() { m.Pin(ctx, "pkg") })

		err := m.Unpin(ctx, "pkg")
		if err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		expected := `{ "imports": {} }`
		current := strings.Join(strings.Fields(string(m.JSON())), " ")

		if current != expected {
			t.Errorf("Expected %q, got %q", expected, current)
		}

		if _, err := os.Stat("vendor/pkg@1.0.0"); err == nil {
			t.Errorf("Expected directory to be deleted")
		}
	})

	t.Run("correct unpinning nonexistent package", func(t *testing.T) {
		err := m.Unpin(ctx, "foo")
		if err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		expected := `{ "imports": { "pkg": "vendor/pkg@1.0.0/index.js" } }`
		current := strings.Join(strings.Fields(string(m.JSON())), " ")

		if current != expected {
			t.Errorf("Expected %q, got %q", expected, current)
		}

		if _, err := os.Stat("vendor/pkg@1.0.0"); err != nil {
			t.Errorf("Expected directory to exist, got error %v", err)
		}
	})
}

func TestImportMapUpdate(t *testing.T) {
	os.Chdir(t.TempDir())

	http.DefaultClient = &http.Client{
		Transport: &mockedRoundTripper{},
	}

	t.Cleanup(func() {
		http.DefaultClient = &http.Client{}
		os.Remove("importmap.json")
		os.RemoveAll("vendor")
	})

	mockedGenerator := &mockGenerator{}
	mockAuditor := &mockAuditor{}

	ctx := context.Background()
	m := importmap.NewManager(".", mockedGenerator, mockAuditor)

	m.Pin(ctx, "pkg")

	if _, err := os.Stat("vendor/pkg@1.0.0"); err != nil {
		t.Errorf("Expected directory to exist, got error %v", err)
	}

	t.Run("correct updating pinned packages", func(t *testing.T) {
		// mockAuditor returns version 2.2.3 as outdated, so Update should upgrade
		err := m.Update(ctx)
		if err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		expected := `
		{
			"imports": {
				"pkg": "vendor/pkg@2.2.3/index.js"
			}
		}`

		expected = strings.Join(strings.Fields(expected), " ")
		current := strings.Join(strings.Fields(string(m.JSON())), " ")

		if current != expected {
			t.Errorf("Expected %q, got %q", expected, current)
		}

		// New version directory should exist
		if _, err := os.Stat("vendor/pkg@2.2.3"); err != nil {
			t.Errorf("Expected updated directory to exist, got error %v", err)
		}
		
		// Old version should be deleted
		if _, err := os.Stat("vendor/pkg@1.0.0"); err == nil {
			t.Errorf("Expected old directory to be deleted")
		}
	})
}

func TestImportMapPristine(t *testing.T) {
	os.Chdir(t.TempDir())

	http.DefaultClient = &http.Client{
		Transport: &mockedRoundTripper{},
	}

	t.Cleanup(func() {
		http.DefaultClient = &http.Client{}
		os.Remove("importmap.json")
		os.RemoveAll("vendor")
	})

	currentImportMap := `{ "imports": { "pkg": "vendor/pkg@1.0.0/index.js" } }`

	f, _ := os.Create("importmap.json")
	f.WriteString(currentImportMap)
	f.Close()

	if _, err := os.Stat("vendor/pkg@1.0.0"); !os.IsNotExist(err) {
		t.Errorf("Expected directory not to exist, got error %v", err)
	}

	mockedGenerator := &mockGenerator{}
	mockAuditor := &mockAuditor{}
	m := importmap.NewManager(".", mockedGenerator, mockAuditor)

	if err := m.Pristine(context.Background()); err != nil {
		t.Errorf("Expected nil, got error %v", err)
	}

	if _, err := os.Stat("vendor/pkg@1.0.0/index.js"); os.IsNotExist(err) {
		t.Error("Expected file to exist")
	}
}

func TestImportMapJSON(t *testing.T) {
	os.Chdir(t.TempDir())

	http.DefaultClient = &http.Client{
		Transport: &mockedRoundTripper{},
	}

	t.Cleanup(func() {
		http.DefaultClient = &http.Client{}
		os.Remove("importmap.json")
		os.RemoveAll("vendor")
	})

	mockedGenerator := &mockGenerator{}
	mockAuditor := &mockAuditor{}

	t.Run("correct JSON", func(t *testing.T) {
		m := importmap.NewManager(".", mockedGenerator, mockAuditor)

		m.Pin(context.Background(), "@pkg/one", "@pkg/two", "@pkg/three")

		expected := `{ "imports": { "@pkg/one": "vendor/@pkg/one@1.0.0/index.js", "@pkg/three": "vendor/@pkg/three@1.0.0/index.js", "@pkg/two": "vendor/@pkg/two@1.0.0/index.js" } }`

		if strings.Join(strings.Fields(string(m.JSON())), " ") != expected {
			t.Errorf("Expected %q, got %q", expected, m.JSON())
		}
	})

	t.Run("correct no modules", func(t *testing.T) {
		content := `{ "imports": { "pkg": "vendor/pkg@1.0.0.js`

		f, _ := os.Create("importmap.json")
		f.WriteString(content)
		f.Close()

		m := importmap.NewManager(".", mockedGenerator, mockAuditor)

		expected := `{ "imports": {} }`
		current := strings.Join(strings.Fields(string(m.JSON())), " ")

		if current != expected {
			t.Errorf("Expected %q, got %q", expected, current)
		}
	})
}

func TestImportMapPackages(t *testing.T) {
	os.Chdir(t.TempDir())

	http.DefaultClient = &http.Client{
		Transport: &mockedRoundTripper{},
	}

	t.Cleanup(func() {
		http.DefaultClient = &http.Client{}
		os.Remove("importmap.json")
		os.RemoveAll("vendor")
	})

	mockedGenerator := &mockGenerator{}
	mockAuditor := &mockAuditor{}

	t.Run("correct printing packages", func(t *testing.T) {
		m := importmap.NewManager(".", mockedGenerator, mockAuditor)

		m.Pin(context.Background(), "@pkg/one", "@pkg/two", "@pkg/three")

		r, w, _ := os.Pipe()

		current := os.Stdout
		os.Stdout = w
		t.Cleanup(func() {
			os.Stdout = current
		})

		m.Packages()

		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)

		if !strings.Contains(buf.String(), "@pkg/one   to: vendor/@pkg/one@1.0.0/index.js") {
			t.Errorf("Expected '@pkg/one   to: vendor/@pkg/one@1.0.0/index.js', got '%v'", buf.String())
		}

		if !strings.Contains(buf.String(), "@pkg/two   to: vendor/@pkg/two@1.0.0/index.js") {
			t.Errorf("Expected '@pkg/two   to: vendor/@pkg/two@1.0.0/index.js', got '%v'", buf.String())
		}

		if !strings.Contains(buf.String(), "@pkg/three to: vendor/@pkg/three@1.0.0/index.js") {
			t.Errorf("Expected '@pkg/three to: vendor/@pkg/three@1.0.0/index.js', got '%v'", buf.String())
		}
	})
}

func TestImportMapAudit(t *testing.T) {
	os.Chdir(t.TempDir())

	http.DefaultClient = &http.Client{
		Transport: &mockedRoundTripper{},
	}

	t.Cleanup(func() {
		http.DefaultClient = &http.Client{}
		os.Remove("importmap.json")
		os.RemoveAll("vendor")
	})

	mockedGenerator := &mockGenerator{}
	mockAuditor := &mockAuditor{}

	t.Run("correct auditing packages", func(t *testing.T) {
		m := importmap.NewManager(".", mockedGenerator, mockAuditor)

		ctx := context.Background()

		m.Pin(ctx, "@pkg/one")

		r, w, _ := os.Pipe()

		current := os.Stdout
		os.Stdout = w
		t.Cleanup(func() {
			os.Stdout = current
		})

		if err := m.Audit(ctx); err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)

		// [info] Report results:

		// @pkg/one@1.0.0
		//   Severity            "low"
		//   Description         "test description"
		//   Vulnerable versions "> 1.0.0"

		if !strings.Contains(buf.String(), "[info] Report results:") {
			t.Errorf("Expected '[info] Report results:', got '%v'", buf.String())
		}

		if !strings.Contains(buf.String(), "@pkg/one@1.0.0") {
			t.Errorf("Expected '@pkg/one@1.0.0', got '%v'", buf.String())
		}

		if !strings.Contains(buf.String(), `Severity            "low"`) {
			t.Errorf(`Expected 'Severity            "low"', got '%v'`, buf.String())
		}

		if !strings.Contains(buf.String(), `Description         "test description"`) {
			t.Errorf(`Expected 'Description         "test description"', got '%v'`, buf.String())
		}

		if !strings.Contains(buf.String(), `Vulnerable versions "> 1.0.0"`) {
			t.Errorf(`Expected 'Vulnerable versions "> 1.0.0"', got '%v'`, buf.String())
		}
	})

	t.Run("correct auditing no packages", func(t *testing.T) {
		os.Remove("importmap.json")
		os.RemoveAll("vendor")

		m := importmap.NewManager(".", mockedGenerator, mockAuditor)

		r, w, _ := os.Pipe()

		current := os.Stdout
		os.Stdout = w
		t.Cleanup(func() {
			os.Stdout = current
		})

		ctx := context.Background()
		ctx = inCtx(ctx, "audit_zero", true)
		if err := m.Audit(ctx); err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)

		if !strings.Contains(buf.String(), "[info] no packages to check for vulnerabilities.") {
			t.Errorf("Expected '[info] no packages to check for vulnerabilities.', got '%v'", buf.String())
		}
	})

	t.Run("correct auditing packages no vulnerabilities", func(t *testing.T) {
		os.Remove("importmap.json")
		os.RemoveAll("vendor")

		m := importmap.NewManager(".", mockedGenerator, mockAuditor)

		m.Pin(context.Background(), "@pkg/one")

		r, w, _ := os.Pipe()

		current := os.Stdout
		os.Stdout = w
		t.Cleanup(func() {
			os.Stdout = current
		})

		ctx := context.Background()
		ctx = inCtx(ctx, "audit_zero", true)

		if err := m.Audit(ctx); err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)

		if !strings.Contains(buf.String(), "[info] No vulnerable packages found.") {
			t.Errorf("Expected '[info] Report results:', got '%v'", buf.String())
		}
	})

	t.Run("incorrect auditing packages error", func(t *testing.T) {
		m := importmap.NewManager(".", mockedGenerator, mockAuditor)

		m.Pin(context.Background(), "@pkg/one")

		ctx := context.Background()
		ctx = inCtx(ctx, "audit_error", true)
		if err := m.Audit(ctx); err == nil {
			t.Errorf("Expected error, got nil")
		}
	})
}

func TestImportMapOutdated(t *testing.T) {
	os.Chdir(t.TempDir())

	http.DefaultClient = &http.Client{
		Transport: &mockedRoundTripper{},
	}

	t.Cleanup(func() {
		http.DefaultClient = &http.Client{}
		os.Remove("importmap.json")
		os.RemoveAll("vendor")
	})

	mockedGenerator := &mockGenerator{}
	mockAuditor := &mockAuditor{}

	t.Run("correct outdated packages", func(t *testing.T) {
		t.Cleanup(func() {
			http.DefaultClient = &http.Client{}
			os.Remove("importmap.json")
			os.RemoveAll("vendor")
		})

		m := importmap.NewManager(".", mockedGenerator, mockAuditor)

		m.Pin(context.Background(), "@pkg/one")

		r, w, _ := os.Pipe()

		current := os.Stdout
		os.Stdout = w
		t.Cleanup(func() {
			os.Stdout = current
		})

		ctx := context.Background()

		if err := m.OutdatedPackages(ctx); err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)

		// @pkg/one@1.0.0 pinned: 1.0.0, latest: 2.2.3
		if !strings.Contains(buf.String(), "@pkg/one pinned: 1.0.0, latest: 2.2.3") {
			t.Errorf("Expected '@pkg/one pinned: 1.0.0, latest: 2.2.3', got '%v'", buf.String())
		}
	})

	t.Run("correct zero outdated packages", func(t *testing.T) {
		t.Cleanup(func() {
			http.DefaultClient = &http.Client{}
			os.Remove("importmap.json")
			os.RemoveAll("vendor")
		})

		m := importmap.NewManager(".", mockedGenerator, mockAuditor)

		m.Pin(context.Background(), "@pkg/one")

		r, w, _ := os.Pipe()

		current := os.Stdout
		os.Stdout = w
		t.Cleanup(func() {
			os.Stdout = current
		})

		ctx := context.Background()
		ctx = inCtx(ctx, "outdated_zero", true)

		if err := m.OutdatedPackages(ctx); err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)

		if !strings.Contains(buf.String(), "[info] All packages are up to date.") {
			t.Errorf("Expected '[info] All packages are up to date.', got '%v'", buf.String())
		}
	})

	t.Run("incorrect outdated packages error", func(t *testing.T) {
		t.Cleanup(func() {
			http.DefaultClient = &http.Client{}
			os.Remove("importmap.json")
			os.RemoveAll("vendor")
		})

		m := importmap.NewManager(".", mockedGenerator, mockAuditor)

		m.Pin(context.Background(), "@pkg/one")

		ctx := context.Background()
		ctx = inCtx(ctx, "outdated_error", true)

		if err := m.OutdatedPackages(ctx); err == nil {
			t.Errorf("Expected error, got nil")
		}
	})
}

func TestImportMapScopes(t *testing.T) {
	os.Chdir(t.TempDir())

	http.DefaultClient = &http.Client{
		Transport: &mockedRoundTripper{},
	}

	t.Cleanup(func() {
		http.DefaultClient = &http.Client{}
		os.Remove("importmap.json")
		os.RemoveAll("vendor")
	})

	// Create a mock generator that returns scopes
	mockGenWithScopes := &mockGeneratorWithScopes{}
	mockAuditor := &mockAuditor{}

	ctx := context.Background()

	t.Run("correct pinning a package with scopes", func(t *testing.T) {
		t.Cleanup(func() {
			os.Remove("importmap.json")
			os.RemoveAll("vendor")
		})

		m := importmap.NewManager(".", mockGenWithScopes, mockAuditor)
		err := m.Pin(ctx, "@org/package-main")
		if err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		jsonBytes := m.JSON()
		jsonStr := string(jsonBytes)

		if !strings.Contains(jsonStr, "@org/package-main") {
			t.Error("Expected @org/package-main in JSON")
		}

		if !strings.Contains(jsonStr, "scopes") {
			t.Error("Expected scopes field in JSON")
		}

		if !strings.Contains(jsonStr, "vendor/") {
			t.Error("Expected vendor/ paths in scopes")
		}

		if !strings.Contains(jsonStr, "@org/package-dep/utils") {
			t.Error("Expected @org/package-dep/utils in scopes")
		}
	})

	t.Run("unpinning a package removes it from imports", func(t *testing.T) {
		t.Cleanup(func() {
			os.Remove("importmap.json")
			os.RemoveAll("vendor")
		})

		m := importmap.NewManager(".", mockGenWithScopes, mockAuditor)
		m.Pin(ctx, "@org/package-main")

		// Verify scope exists before unpin
		jsonBefore := string(m.JSON())
		if !strings.Contains(jsonBefore, "scopes") {
			t.Fatal("Expected scopes before unpin")
		}

		err := m.Unpin(ctx, "@org/package-main")
		if err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		jsonAfter := string(m.JSON())
		if strings.Contains(jsonAfter, "@org/package-main") {
			t.Error("Expected @org/package-main to be removed from imports")
		}
	})

	t.Run("JSON omits empty scopes", func(t *testing.T) {
		t.Cleanup(func() {
			os.Remove("importmap.json")
			os.RemoveAll("vendor")
		})

		m := importmap.NewManager(".", &mockGenerator{}, mockAuditor)
		m.Pin(ctx, "pkg")

		jsonBytes := m.JSON()
		jsonStr := string(jsonBytes)

		// Should not contain "scopes" field if empty
		if strings.Contains(jsonStr, "scopes") {
			t.Errorf("Expected JSON to omit empty scopes field, got: %s", jsonStr)
		}
	})

	t.Run("JSON includes scopes when present", func(t *testing.T) {
		t.Cleanup(func() {
			os.Remove("importmap.json")
			os.RemoveAll("vendor")
		})

		m := importmap.NewManager(".", mockGenWithScopes, mockAuditor)
		m.Pin(ctx, "@org/package-main")

		jsonBytes := m.JSON()
		jsonStr := string(jsonBytes)

		// Should contain "scopes" field
		if !strings.Contains(jsonStr, "scopes") {
			t.Errorf("Expected JSON to include scopes field, got: %s", jsonStr)
		}
	})

	t.Run("downloads staticDeps including special _/ files", func(t *testing.T) {
		t.Cleanup(func() {
			os.Remove("importmap.json")
			os.RemoveAll("vendor")
		})

		mockGenWithStaticDeps := &mockGeneratorWithStaticDeps{}
		m := importmap.NewManager(".", mockGenWithStaticDeps, mockAuditor)

		err := m.Pin(ctx, "@org/package-ext")
		if err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		// Verify the _/ file was downloaded to scopes (dependency of main package)
		if _, err := os.Stat("vendor/@org/package-ext@2.5.0/scopes/@org/package-base@1.5.0/_/xyz789.js"); err != nil {
			t.Errorf("Expected _/ file in scopes, got error: %v", err)
		}

		// Verify the runtime file was downloaded to scopes
		if _, err := os.Stat("vendor/@org/package-ext@2.5.0/scopes/@org/package-base@1.5.0/runtime/index.js"); err != nil {
			t.Errorf("Expected runtime file in scopes, got error: %v", err)
		}

		// Verify the main package file was downloaded to package root
		if _, err := os.Stat("vendor/@org/package-ext@2.5.0/dist/index.js"); err != nil {
			t.Errorf("Expected main package file to exist, got error: %v", err)
		}

		// Verify JSON structure - internal files are NOT in importmap, only main package
		jsonBytes := m.JSON()
		jsonStr := string(jsonBytes)

		// Should only have the main package in imports, not the internal _/ files
		if !strings.Contains(jsonStr, "@org/package-ext") {
			t.Error("Expected main package in imports")
		}
		
		if !strings.Contains(jsonStr, "vendor/@org/package-ext@2.5.0/dist/index.js") {
			t.Errorf("Expected main package path in imports, got: %s", jsonStr)
		}
	})
}

// mockGeneratorWithStaticDeps returns imports with staticDeps including _/ files
type mockGeneratorWithStaticDeps struct{}

func (g *mockGeneratorWithStaticDeps) Generate(ctx context.Context, packages ...string) (interface{}, error) {
	imports := map[string]string{
		"@org/package-ext": "https://ga.jspm.io/npm:@org/package-ext@2.5.0/dist/index.js",
	}

	staticDeps := []string{
		"https://ga.jspm.io/npm:@org/package-base@1.5.0/_/xyz789.js",
		"https://ga.jspm.io/npm:@org/package-base@1.5.0/runtime/index.js",
		"https://ga.jspm.io/npm:@org/package-ext@2.5.0/dist/index.js",
	}

	return &mockGeneratorResult{
		imports:    imports,
		scopes:     make(map[string]map[string]string),
		staticDeps: staticDeps,
	}, nil
}

// mockGeneratorWithScopes returns both imports and scopes
type mockGeneratorWithScopes struct{}

func (g *mockGeneratorWithScopes) Generate(ctx context.Context, packages ...string) (interface{}, error) {
	imports := map[string]string{
		"@org/package-main": "https://ga.jspm.io/npm:@org/package-main@1.2.0/dist/index.js",
	}

	scopes := map[string]map[string]string{
		"https://ga.jspm.io/": {
			"@org/package-dep/utils": "https://ga.jspm.io/npm:@org/package-dep@2.3.0/dist/utils/index.js",
			"package-helper":         "https://ga.jspm.io/npm:package-helper@3.1.0/dist/index.js",
		},
	}

	return &mockGeneratorResult{
		imports:    imports,
		scopes:     scopes,
		staticDeps: []string{}, // Empty by default
	}, nil
}

func TestProcess(t *testing.T) {
	os.Chdir(t.TempDir())

	http.DefaultClient = &http.Client{
		Transport: &mockedRoundTripper{},
	}

	t.Cleanup(func() {
		http.DefaultClient = &http.Client{}
		os.Remove("importmap.json")
		os.RemoveAll("vendor")
	})

	t.Run("correct process with no arguments", func(t *testing.T) {
		r, w, _ := os.Pipe()

		current := os.Stdout
		os.Stdout = w
		t.Cleanup(func() {
			os.Stdout = current
		})

		os.Args = []string{"importmap"}
		ctx := context.Background()
		if err := importmap.Process(ctx); err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)

		if !strings.Contains(buf.String(), "Usage: importmap [flags] <command> [arguments]") {
			t.Errorf("Expected 'Usage: importmap [flags] <command> [arguments]', got '%v'", buf.String())
		}

		if !strings.Contains(buf.String(), "Available commands:") {
			t.Errorf("Expected 'Available commands:', got '%v'", buf.String())
		}
	})

	t.Run("correct process pin with no package", func(t *testing.T) {
		r, w, _ := os.Pipe()

		current := os.Stdout
		os.Stdout = w
		t.Cleanup(func() {
			os.Stdout = current
		})

		os.Args = []string{"importmap", "pin"}

		ctx := context.Background()
		if err := importmap.Process(ctx); err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)

		if !strings.Contains(buf.String(), "[info] importmap pin <package> [package...]") {
			t.Errorf("Expected '[info] importmap pin <package> [package...]', got '%v'", buf.String())
		}
	})

	t.Run("correct process with pin command", func(t *testing.T) {
		r, w, _ := os.Pipe()

		current := os.Stdout
		os.Stdout = w
		t.Cleanup(func() {
			os.Stdout = current
		})

		os.Args = []string{"importmap", "pin", "pkg"}

		ctx := context.Background()
		if err := importmap.Process(ctx); err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)

		output := buf.String()
		if !strings.Contains(output, "[info] pinned pkg") {
			t.Errorf("Expected '[info] pinned pkg', got '%v'", output)
		}

		if !strings.Contains(output, "[info] Packages pinned successfully") {
			t.Errorf("Expected '[info] Packages pinned successfully', got '%v'", output)
		}
	})

	t.Run("incorrect process pinning package error", func(t *testing.T) {
		ctx := context.Background()
		ctx = inCtx(ctx, "download_error", true)

		os.Args = []string{"importmap", "pin", "pkg"}

		if err := importmap.Process(ctx); err == nil {
			t.Errorf("Expected error, got nil")
		}
	})

	t.Run("correct process unpinning a package", func(t *testing.T) {
		t.Cleanup(func() {
			os.Remove("importmap.json")
			os.RemoveAll("vendor")
		})

		r, w, _ := os.Pipe()

		current := os.Stdout
		os.Stdout = w
		t.Cleanup(func() {
			os.Stdout = current
		})

		content := `{ "imports": { "pkg": "vendor/pkg@1.0.0/index.js"	} }`

		f, _ := os.Create("importmap.json")
		f.WriteString(content)
		f.Close()

		os.Args = []string{"importmap", "--importmap.folder=.", "audit"}

		ctx := context.Background()
		if err := importmap.Process(ctx); err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}
		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)

		if !strings.Contains(buf.String(), "[info] audit report:") {
			t.Errorf("Expected '[info] audit report:', got '%v'", buf.String())
		}

		if !strings.Contains(buf.String(), "pkg") {
			t.Errorf("Expected 'pkg', got '%v'", buf.String())
		}

		if !strings.Contains(buf.String(), "Severity            \"low\"") {
			t.Errorf("Expected 'Severity            \"low\"', got '%v'", buf.String())
		}

		if !strings.Contains(buf.String(), "Description         \"test description\"") {
			t.Errorf("Expected 'Description         \"test description\"', got '%v'", buf.String())
		}

		if !strings.Contains(buf.String(), "Vulnerable versions \"> 1.0.0\"") {
			t.Errorf("Expected 'Vulnerable versions \"> 1.0.0\"', got '%v'", buf.String())
		}
	})

	t.Run("unknown command", func(t *testing.T) {
		os.Args = []string{"importmap", "--importmap.folder=.", "unknown_command"}

		ctx := context.Background()
		if err := importmap.Process(ctx); err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}
	})
}

func inCtx(ctx context.Context, key, value any) context.Context {
	return context.WithValue(ctx, key, value)
}

type mockAuditor struct{}

func (a *mockAuditor) Audit(ctx context.Context, packages map[string]string) (map[string][]map[string]string, error) {
	if ctx.Value("audit_error") != nil {
		return nil, fmt.Errorf("test error")
	}

	if ctx.Value("audit_zero") != nil {
		return nil, nil
	}

	vulnerabilities := make(map[string][]map[string]string)
	for k := range packages {
		vulnerabilities[k] = append(vulnerabilities[k], map[string]string{
			"severity":            "low",
			"description":         "test description",
			"vulnerable_versions": "> 1.0.0",
		})
	}

	return vulnerabilities, nil
}

func (a *mockAuditor) Outdated(ctx context.Context, packages map[string]string) (map[string]string, error) {
	if ctx.Value("outdated_error") != nil {
		return nil, fmt.Errorf("test error")
	}

	if ctx.Value("outdated_zero") != nil {
		return nil, nil
	}

	outdated := make(map[string]string)
	for k := range packages {
		outdated[k] = "2.2.3"
	}

	return outdated, nil
}

type mockGeneratorResult struct {
	imports    map[string]string
	scopes     map[string]map[string]string
	staticDeps []string
}

func (r *mockGeneratorResult) GetImports() map[string]string {
	return r.imports
}

func (r *mockGeneratorResult) GetScopes() map[string]map[string]string {
	return r.scopes
}

func (r *mockGeneratorResult) GetStaticDeps() []string {
	return r.staticDeps
}

type mockGenerator struct{}

func (g *mockGenerator) Generate(ctx context.Context, packages ...string) (interface{}, error) {
	if ctx.Value("generate_no_exists") != nil {
		pkg := "pkg"
		if len(packages) > 0 {
			pkg = packages[0]
		}

		return nil, fmt.Errorf("Error: Unable to resolve npm:%s@ to a valid version", pkg)
	}

	if ctx.Value("generate_error") != nil {
		return nil, fmt.Errorf("test error")
	}

	pkgVersionRegex := regexp.MustCompile(`(.+)\@([\d\.]+)$`)

	m := map[string]string{}
	staticDeps := []string{}
	
	for _, pkg := range packages {
		p := pkg

		if !pkgVersionRegex.MatchString(p) {
			p += "@1.0.0"
		}

		matches := pkgVersionRegex.FindStringSubmatch(p)
		
		packageURL := fmt.Sprintf("https://ga.jspm.io/npm:%s/index.js", p)
		m[matches[1]] = packageURL
		
		// Add the package URL to staticDeps (this is what JSPM returns)
		staticDeps = append(staticDeps, packageURL)
	}

	return &mockGeneratorResult{
		imports:    m,
		scopes:     make(map[string]map[string]string),
		staticDeps: staticDeps,
	}, nil
}

type mockedRoundTripper struct{}

func (m *mockedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Context().Value("download_error") != nil {
		return nil, fmt.Errorf("download test error")
	}

	if strings.HasPrefix(req.URL.String(), "https://api.jspm.io/generate") {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{ "map":{ "imports":{ "pkg": "https://ga.jspm.io/npm:pkg@1.0.0/index.js" } } }`)),
		}, nil
	}

	if strings.HasPrefix(req.URL.String(), "https://registry.npmjs.org") {
		if strings.Contains(req.URL.String(), "-/npm/v1/security/advisories/bulk") {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"pkg": [{ "title": "test description", "severity": "low", "vulnerable_versions": "> 1.0.0"}]}`)),
			}, nil
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{ "versions":{ "1.0.0":{},"2.2.3":{} } }`)),
		}, nil
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("file content")),
	}, nil
}
