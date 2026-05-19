package bundlejs_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"go.leapkit.dev/tools/importmap/internal/bundlejs"
)

// roundTripper is a test http.RoundTripper backed by a function.
type roundTripper func(*http.Request) (*http.Response, error)

func (r roundTripper) RoundTrip(req *http.Request) (*http.Response, error) { return r(req) }

// withTransport installs a mock HTTP transport for the duration of the test.
func withTransport(t *testing.T, fn roundTripper) {
	t.Helper()
	orig := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: fn}
	t.Cleanup(func() { http.DefaultClient = orig })
}

// mockTransport returns a RoundTripper that handles bundlejs requests:
//   - resolveVersion: GET /?q=pkg       -> {"version":"name@version"}
//   - fetchBundle:    GET /?q=...&file=true -> JS bundle bytes
//
// The resolved version defaults to 1.0.0 unless pkg already has @version.
func mockTransport(bundleBody string) roundTripper {
	return func(req *http.Request) (*http.Response, error) {
		q := req.URL.Query()
		pkg := q.Get("q")

		if q.Get("file") == "true" {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(bundleBody)),
			}, nil
		}

		// resolveVersion: echo back full spec
		version := "1.0.0"
		name := pkg
		if idx := strings.LastIndex(pkg, "@"); idx > 0 {
			version = pkg[idx+1:]
			name = pkg[:idx]
		}
		body := fmt.Sprintf(`{"version":%q}`, name+"@"+version)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	}
}

func TestBundleResolvesVersion(t *testing.T) {
	withTransport(t, mockTransport("/* bundle */"))

	result, err := new(bundlejs.Client).Bundle(context.Background(), "some-package")
	if err != nil {
		t.Fatalf("Bundle: %v", err)
	}

	if _, ok := result["some-package@1.0.0"]; !ok {
		t.Errorf("expected key some-package@1.0.0, got keys: %v", keys(result))
	}
}

func TestBundleRespectsExplicitVersion(t *testing.T) {
	withTransport(t, mockTransport("/* bundle */"))

	result, err := new(bundlejs.Client).Bundle(context.Background(), "some-package@2.3.4")
	if err != nil {
		t.Fatalf("Bundle: %v", err)
	}

	if _, ok := result["some-package@2.3.4"]; !ok {
		t.Errorf("expected key some-package@2.3.4, got keys: %v", keys(result))
	}
}

func TestBundleScopedPackage(t *testing.T) {
	withTransport(t, mockTransport("/* bundle */"))

	result, err := new(bundlejs.Client).Bundle(context.Background(), "@scope/pkg")
	if err != nil {
		t.Fatalf("Bundle: %v", err)
	}

	if _, ok := result["@scope/pkg@1.0.0"]; !ok {
		t.Errorf("expected key @scope/pkg@1.0.0, got keys: %v", keys(result))
	}
}

func TestBundleReturnsFileContent(t *testing.T) {
	withTransport(t, mockTransport("var x=1;"))

	result, err := new(bundlejs.Client).Bundle(context.Background(), "some-package")
	if err != nil {
		t.Fatalf("Bundle: %v", err)
	}

	content := result["some-package@1.0.0"]
	if string(content) != "var x=1;" {
		t.Errorf("content = %q, want %q", content, "var x=1;")
	}
}

func TestBundleMultiplePackages(t *testing.T) {
	withTransport(t, mockTransport("/* bundle */"))

	result, err := new(bundlejs.Client).Bundle(context.Background(), "pkg-a", "pkg-b", "pkg-c")
	if err != nil {
		t.Fatalf("Bundle: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 entries, got %d: %v", len(result), keys(result))
	}
}

func TestBundleErrorOnResolveFailure(t *testing.T) {
	withTransport(t, func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	})

	_, err := new(bundlejs.Client).Bundle(context.Background(), "no-such-package")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestBundleErrorOnFetchFailure(t *testing.T) {
	withTransport(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Query().Get("file") == "true" {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		}
		// resolveVersion succeeds
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"version":"some-package@1.0.0"}`)),
		}, nil
	})

	_, err := new(bundlejs.Client).Bundle(context.Background(), "some-package")
	if err == nil {
		t.Error("expected error on fetch failure, got nil")
	}
}

func TestBundleErrorOnMalformedJSON(t *testing.T) {
	withTransport(t, func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("not json")),
		}, nil
	})

	_, err := new(bundlejs.Client).Bundle(context.Background(), "some-package")
	if err == nil {
		t.Error("expected error for malformed JSON, got nil")
	}
}

func TestBundleErrorOnMissingVersionField(t *testing.T) {
	withTransport(t, func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"size":1234}`)),
		}, nil
	})

	_, err := new(bundlejs.Client).Bundle(context.Background(), "some-package")
	if err == nil {
		t.Error("expected error for missing version field, got nil")
	}
}

// keys returns the keys of a map for use in error messages.
func keys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
