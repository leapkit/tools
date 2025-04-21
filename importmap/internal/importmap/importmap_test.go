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

		expected := `{ "imports": { "pkg": "vendor/pkg@1.0.0.js" } }`
		current := strings.Join(strings.Fields(string(m.JSON())), " ")

		if current != expected {
			t.Errorf("Expected %q, got %q", expected, current)
		}

		if _, err := os.Stat("vendor/pkg@1.0.0.js"); err != nil {
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
				"@pkg/one": "vendor/@pkg/one@1.0.0.js",
				"@pkg/three": "vendor/@pkg/three@1.0.0.js",
				"@pkg/two": "vendor/@pkg/two@1.0.0.js"
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

		expected := `{ "imports": { "pkg": "vendor/pkg@1.2.0.js" } }`
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

		expected := `{ "imports": { "pkg": "vendor/pkg@1.0.0.js" } }`
		current := strings.Join(strings.Fields(string(m.JSON())), " ")

		if current != expected {
			t.Errorf("Expected %q, got %q", expected, current)
		}

		err = m.Pin(ctx, "pkg@1.2.0")
		if err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		expected = `{ "imports": { "pkg": "vendor/pkg@1.2.0.js" } }`
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

	t.Run("incorrect download error should return an error", func(t *testing.T) {
		t.Cleanup(func() {
			os.Remove("importmap.json")
			os.RemoveAll("vendor")
		})

		ctx := context.Background()
		ctx = inCtx(ctx, "download_error", true)

		m := importmap.NewManager(".", mockedGenerator, mockAuditor)
		err := m.Pin(ctx, "pkg")
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		expectedErr := "download test error"
		if !strings.Contains(err.Error(), expectedErr) {
			t.Errorf("Expected %q, got %q", expectedErr, err.Error())
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

	if _, err := os.Stat("vendor/pkg@1.0.0.js"); err != nil {
		t.Errorf("Expected nil, got error %v", err)
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

		if _, err := os.Stat("vendor/pkg@1.0.0.js"); !os.IsNotExist(err) {
			t.Errorf("Expected no exits file, got %v", err)
		}
	})

	t.Run("correct unpinning nonexistent package", func(t *testing.T) {
		err := m.Unpin(ctx, "foo")
		if err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		expected := `{ "imports": { "pkg": "vendor/pkg@1.0.0.js" } }`
		current := strings.Join(strings.Fields(string(m.JSON())), " ")

		if current != expected {
			t.Errorf("Expected %q, got %q", expected, current)
		}

		if _, err := os.Stat("vendor/pkg@1.0.0.js"); err != nil {
			t.Errorf("Expected nil, got error %v", err)
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

	if _, err := os.Stat("vendor/pkg@1.0.0.js"); err != nil {
		t.Errorf("Expected nil, got error %v", err)
	}

	t.Run("correct updating pinned packages", func(t *testing.T) {
		err := m.Update(ctx)
		if err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		expected := `
		{
			"imports": {
				"pkg": "vendor/pkg@2.2.3.js"
			}
		}`

		expected = strings.Join(strings.Fields(expected), " ")
		current := strings.Join(strings.Fields(string(m.JSON())), " ")

		if current != expected {
			t.Errorf("Expected %q, got %q", expected, current)
		}

		if _, err := os.Stat("vendor/pkg@2.2.3.js"); err != nil {
			t.Errorf("Expected nil, got error %v", err)
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

	currentImportmap := `{ "imports": { "pkg": "vendor/pkg@1.0.0.js" } }`

	f, _ := os.Create("importmap.json")
	f.WriteString(currentImportmap)
	f.Close()

	if _, err := os.Stat("vendor/pkg@1.0.0.js"); !os.IsNotExist(err) {
		t.Errorf("Expected nil, got error %v", err)
	}

	mockedGenerator := &mockGenerator{}
	mockAuditor := &mockAuditor{}
	m := importmap.NewManager(".", mockedGenerator, mockAuditor)

	if err := m.Pristine(context.Background()); err != nil {
		t.Errorf("Expected nil, got error %v", err)
	}

	if _, err := os.Stat("vendor/pkg@1.0.0.js"); os.IsNotExist(err) {
		t.Error("Expected file exists")
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

		expected := `{ "imports": { "@pkg/one": "vendor/@pkg/one@1.0.0.js", "@pkg/three": "vendor/@pkg/three@1.0.0.js", "@pkg/two": "vendor/@pkg/two@1.0.0.js" } }`

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

		if !strings.Contains(buf.String(), "@pkg/one   to: vendor/@pkg/one@1.0.0.js") {
			t.Errorf("Expected '@pkg/one   to: vendor/@pkg/one@1.0.0.js', got '%v'", buf.String())
		}

		if !strings.Contains(buf.String(), "@pkg/two   to: vendor/@pkg/two@1.0.0.js") {
			t.Errorf("Expected '@pkg/two   to: vendor/@pkg/two@1.0.0.js', got '%v'", buf.String())
		}

		if !strings.Contains(buf.String(), "@pkg/three to: vendor/@pkg/three@1.0.0.js") {
			t.Errorf("Expected '@pkg/three to: vendor/@pkg/three@1.0.0.js', got '%v'", buf.String())
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

		if !strings.Contains(buf.String(), "[info] downloading internal/system/assets/vendor/pkg@1.0.0.js") {
			t.Errorf("Expected '[info] downloading internal/system/assets/vendor/pkg@1.0.0.js', got '%v'", buf.String())
		}

		if !strings.Contains(buf.String(), "[info] Packages pinned successfully") {
			t.Errorf("Expected '[info] Packages pinned successfully', got '%v'", buf.String())
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

		content := `{ "imports": { "pkg": "vendor/pkg@1.0.0.js"	} }`

		f, _ := os.Create("importmap.json")
		f.WriteString(content)
		f.Close()

		os.Args = []string{"importmap", "--importmap.folder=.", "unpin", "pkg"}
		ctx := context.Background()
		if err := importmap.Process(ctx); err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)

		if !strings.Contains(buf.String(), "[info] unpinning pkg@1.0.0") {
			t.Errorf("Expected 'unpinning pkg@1.0.0', got '%v'", buf.String())
		}

		if !strings.Contains(buf.String(), "[info] Packages unpinned successfully") {
			t.Errorf("Expected 'Packages unpinned successfully', got '%v'", buf.String())
		}
	})

	t.Run("correct process unpin with no package", func(t *testing.T) {
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

		os.Args = []string{"importmap", "--importmap.folder=.", "unpin"}

		ctx := context.Background()
		if err := importmap.Process(ctx); err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)

		if !strings.Contains(buf.String(), "[info] importmap unpin <package> [packages...]") {
			t.Errorf("Expected '[info] importmap unpin <package> [packages...]', got '%v'", buf.String())
		}
	})

	t.Run("correct process update", func(t *testing.T) {
		t.Cleanup(func() {
			os.Remove("importmap.json")
			os.RemoveAll("vendor")
		})

		r, w, _ := os.Pipe()
		stdOut := os.Stdout
		stdErr := os.Stderr

		os.Stdout = w
		os.Stderr = w
		t.Cleanup(func() {
			os.Stdout = stdOut
			os.Stderr = stdErr
		})

		content := `{ "imports": { "pkg": "vendor/pkg@1.0.0.js"	} }`

		f, _ := os.Create("importmap.json")
		f.WriteString(content)
		f.Close()

		os.Args = []string{"importmap", "--importmap.folder=.", "update"}

		ctx := context.Background()
		if err := importmap.Process(ctx); err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)

		if !strings.Contains(buf.String(), "[info] unpinning pkg@1.0.0") {
			t.Errorf("Expected '[info] unpinning pkg@1.0.0', got '%v'", buf.String())
		}

		if !strings.Contains(buf.String(), "[info] downloading vendor/pkg@1.0.0") {
			t.Errorf("Expected '[info] downloading vendor/pkg@1.0.0', got '%v'", buf.String())
		}

		if !strings.Contains(buf.String(), "[info] Packages updated successfully") {
			t.Errorf("Expected '[info] Packages updated successfully', got '%v'", buf.String())
		}
	})
	t.Run("incorrect process update error", func(t *testing.T) {
		t.Cleanup(func() {
			os.Remove("importmap.json")
			os.RemoveAll("vendor")
		})

		content := `{ "imports": { "pkg": "vendor/pkg@1.0.0.js"	} }`

		f, _ := os.Create("importmap.json")
		f.WriteString(content)
		f.Close()

		os.Args = []string{"importmap", "--importmap.folder=.", "update"}

		ctx := context.Background()
		ctx = inCtx(ctx, "download_error", true)
		if err := importmap.Process(ctx); err == nil {
			t.Errorf("Expected nil, got error %v", err)
		}
	})

	t.Run("correct process pristine", func(t *testing.T) {
		t.Cleanup(func() {
			os.Remove("importmap.json")
			os.RemoveAll("vendor")
		})

		r, w, _ := os.Pipe()
		stdOut := os.Stdout
		stdErr := os.Stderr

		os.Stdout = w
		os.Stderr = w
		t.Cleanup(func() {
			os.Stdout = stdOut
			os.Stderr = stdErr
		})

		content := `{ "imports": { "pkg": "vendor/pkg@1.0.0.js"	} }`

		f, _ := os.Create("importmap.json")
		f.WriteString(content)
		f.Close()

		os.Args = []string{"importmap", "--importmap.folder=.", "pristine"}

		ctx := context.Background()
		if err := importmap.Process(ctx); err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)

		if !strings.Contains(buf.String(), "[info] re-downloading pinned packages:") {
			t.Errorf("Expected '[info] re-downloading pinned packages:', got '%v'", buf.String())
		}

		if !strings.Contains(buf.String(), "[info] downloading vendor/pkg@1.0.0") {
			t.Errorf("Expected '[info] downloading vendor/pkg@1.0.0', got '%v'", buf.String())
		}

		if !strings.Contains(buf.String(), "[info] Packages downloaded successfully") {
			t.Errorf("Expected '[info] Packages downloaded successfully', got '%v'", buf.String())
		}

		if _, err := os.Stat("vendor/pkg@1.0.0.js"); err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}
	})

	t.Run("incorrect process pristine error", func(t *testing.T) {
		os.Remove("importmap.json")
		os.RemoveAll("vendor")

		content := `{ "imports": { "pkg": "vendor/pkg@1.0.0.js"	} }`

		f, _ := os.Create("importmap.json")
		f.WriteString(content)
		f.Close()

		os.Args = []string{"importmap", "--importmap.folder=.", "pristine"}

		ctx := context.Background()
		ctx = inCtx(ctx, "download_error", true)
		if err := importmap.Process(ctx); err == nil {
			t.Error("Expected error, got nil")
		}

		if _, err := os.Stat("vendor/pkg@1.0.0.js"); err == nil {
			t.Error("Expected no exists file, got nil")
		}
	})

	t.Run("correct process json", func(t *testing.T) {
		t.Cleanup(func() {
			os.Remove("importmap.json")
			os.RemoveAll("vendor")
		})

		r, w, _ := os.Pipe()
		stdOut := os.Stdout
		stdErr := os.Stderr

		os.Stdout = w
		os.Stderr = w
		t.Cleanup(func() {
			os.Stdout = stdOut
			os.Stderr = stdErr
		})

		content := `{ "imports": { "pkg": "vendor/pkg@1.0.0.js"	} }`

		f, _ := os.Create("importmap.json")
		f.WriteString(content)
		f.Close()

		os.Args = []string{"importmap", "--importmap.folder=.", "json"}

		ctx := context.Background()
		if err := importmap.Process(ctx); err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}

		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)

		if !strings.Contains(buf.String(), "{\n  \"imports\": {\n    \"pkg\": \"vendor/pkg@1.0.0.js\"\n  }\n}") {
			t.Errorf("Expected '{\n  \"imports\": {\n    \"pkg\": \"vendor/pkg@1.0.0.js\"\n  }\n}', got '%v'", buf.String())
		}
	})

	t.Run("correct process packages", func(t *testing.T) {
		t.Cleanup(func() {
			os.Remove("importmap.json")
			os.RemoveAll("vendor")
		})

		r, w, _ := os.Pipe()
		stdOut := os.Stdout
		stdErr := os.Stderr

		os.Stdout = w
		os.Stderr = w
		t.Cleanup(func() {
			os.Stdout = stdOut
			os.Stderr = stdErr
		})

		content := `{ "imports": { "pkg": "vendor/pkg@1.0.0.js"	} }`

		f, _ := os.Create("importmap.json")
		f.WriteString(content)
		f.Close()

		os.Args = []string{"importmap", "--importmap.folder=.", "packages"}

		ctx := context.Background()
		if err := importmap.Process(ctx); err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}
		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)

		if !strings.Contains(buf.String(), "[info] Pinned packages:") {
			t.Errorf("Expected '[info] Pinned packages:', got '%v'", buf.String())
		}

		if !strings.Contains(buf.String(), "pkg") {
			t.Errorf("Expected 'pkg', got '%v'", buf.String())
		}

		if !strings.Contains(buf.String(), "to: vendor/pkg@1.0.0.js") {
			t.Errorf("Expected 'to: vendor/pkg@1.0.0.js', got '%v'", buf.String())
		}
	})

	t.Run("correct process outdated packages", func(t *testing.T) {
		t.Cleanup(func() {
			os.Remove("importmap.json")
			os.RemoveAll("vendor")
		})

		r, w, _ := os.Pipe()
		stdOut := os.Stdout
		stdErr := os.Stderr

		os.Stdout = w
		os.Stderr = w
		t.Cleanup(func() {
			os.Stdout = stdOut
			os.Stderr = stdErr
		})

		content := `{ "imports": { "pkg": "vendor/pkg@1.0.0.js"	} }`

		f, _ := os.Create("importmap.json")
		f.WriteString(content)
		f.Close()

		os.Args = []string{"importmap", "--importmap.folder=.", "outdated"}

		ctx := context.Background()
		if err := importmap.Process(ctx); err != nil {
			t.Errorf("Expected nil, got error %v", err)
		}
		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)

		if !strings.Contains(buf.String(), "[info] outdated packages:") {
			t.Errorf("Expected '[info] outdated packages:', got '%v'", buf.String())
		}

		if !strings.Contains(buf.String(), "pkg") {
			t.Errorf("Expected 'pkg', got '%v'", buf.String())
		}

		if !strings.Contains(buf.String(), "pinned: 1.0.0") {
			t.Errorf("Expected 'pinned: 1.0.0', got '%v'", buf.String())
		}

		if !strings.Contains(buf.String(), "latest: 2.2.3") {
			t.Errorf("Expected 'latest: 2.2.3', got '%v'", buf.String())
		}
	})

	t.Run("correct process audit report", func(t *testing.T) {
		t.Cleanup(func() {
			os.Remove("importmap.json")
			os.RemoveAll("vendor")
		})

		r, w, _ := os.Pipe()
		stdOut := os.Stdout
		stdErr := os.Stderr

		os.Stdout = w
		os.Stderr = w
		t.Cleanup(func() {
			os.Stdout = stdOut
			os.Stderr = stdErr
		})

		content := `{ "imports": { "pkg": "vendor/pkg@1.0.0.js"	} }`

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

	t.Run("incorrect process audit report error", func(t *testing.T) {
		t.Cleanup(func() {
			os.Remove("importmap.json")
			os.RemoveAll("vendor")
		})

		content := `{ "imports": { "pkg": "vendor/pkg@1.0.0.js"	} }`

		f, _ := os.Create("importmap.json")
		f.WriteString(content)
		f.Close()

		os.Args = []string{"importmap", "--importmap.folder=.", "audit"}

		ctx := context.Background()
		ctx = inCtx(ctx, "download_error", true)
		if err := importmap.Process(ctx); err == nil {
			t.Errorf("Expected error, got nil")
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

type mockGenerator struct{}

func (g *mockGenerator) Generate(ctx context.Context, packages ...string) (map[string]string, error) {
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
	for _, pkg := range packages {
		p := pkg

		if !pkgVersionRegex.MatchString(p) {
			p += "@1.0.0"
		}

		matches := pkgVersionRegex.FindStringSubmatch(p)

		m[matches[1]] = fmt.Sprintf("https://ga.jspm.io/npm:%s/index.js", p)
	}

	return m, nil
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
