// Package internal provides bundler implementations used by the importmap manager.
// Each bundler must satisfy the Bundler interface defined in the parent package.
package bundlejs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"sync"
)

// baseURL is the root endpoint for the bundleJS API.
const baseURL = "https://deno.bundlejs.com"

// Client is the Bundler implementation backed by deno.bundleJS.com.
// It resolves package versions and fetches minified ES module bundles
// in parallel, one goroutine per package.
type Client struct{}

// Bundle resolves and fetches each package concurrently.
// It returns a map of "name@version" -> minified JS file contents.
// All packages are processed in parallel; the first error encountered
// is returned and the partial map is discarded.
func (c *Client) Bundle(ctx context.Context, packages ...string) (map[string][]byte, error) {
	var mu sync.Mutex
	var wg sync.WaitGroup

	pkgMap := make(map[string][]byte)
	errCh := make(chan error, len(packages))

	for pkg := range slices.Values(packages) {
		wg.Add(1)
		go func() {
			defer wg.Done()

			pkgName, err := c.resolveVersion(ctx, pkg)
			if err != nil {
				errCh <- err
				return
			}

			content, err := c.fetchBundle(ctx, pkgName)
			if err != nil {
				errCh <- err
				return
			}

			mu.Lock()
			defer mu.Unlock()
			pkgMap[pkgName] = content
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return nil, err
		}
	}

	return pkgMap, nil
}

// resolveVersion calls the bundleJS metadata endpoint to determine the exact resolved
// version for pkg (which may include a semver range or no version at all).
// It returns the full "name@version" spec, e.g.:
//
//	"some-package"        -> "some-package@1.2.3"
//	"@scope/pkg@^2"       -> "@scope/pkg@2.0.0"
func (c *Client) resolveVersion(ctx context.Context, pkg string) (string, error) {
	params := make(url.Values)
	params.Set("q", pkg)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return "", err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("resolve: unexpected status %d for %q", res.StatusCode, pkg)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var meta struct {
		Version string `json:"version"`
	}

	if err := json.Unmarshal(body, &meta); err != nil {
		return "", fmt.Errorf("resolve: parse response JSON: %w", err)
	}

	if meta.Version == "" {
		return "", fmt.Errorf("resolve: response contains no version field for %q", pkg)
	}

	return meta.Version, nil
}

// fetchBundle downloads the minified ES module bundle for the given "name@version" spec
// from the bundleJS file endpoint and returns the raw bytes.
func (c *Client) fetchBundle(ctx context.Context, pkg string) ([]byte, error) {
	params := make(url.Values)
	params.Set("q", pkg)
	params.Set("file", "true")
	params.Set("minify", "true")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch: unexpected status %d for %q", res.StatusCode, pkg)
	}

	return io.ReadAll(res.Body)
}
