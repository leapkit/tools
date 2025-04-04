package npm

import (
	"cmp"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

const baseURL = "https://registry.npmjs.org"

func OutdatedPackages(ctx context.Context, packages map[string]string) map[string]string {
	outdated := make(map[string]string)
	for pkg, version := range packages {
		latest, err := latestVersion(ctx, pkg)
		if err != nil {
			continue
		}

		if cmp.Compare(version, latest) == -1 {
			outdated[pkg] = latest
		}
	}

	return outdated
}

func latestVersion(ctx context.Context, pkg string) (string, error) {
	path, _ := url.JoinPath(baseURL, pkg)

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	b, _ := io.ReadAll(res.Body)

	var response struct {
		Versions map[string]any `json:"versions"`
	}

	if err := json.Unmarshal(b, &response); err != nil {
		return "", err
	}

	var latest string
	for version := range response.Versions {
		latest = max(latest, version)
	}

	return latest, nil
}
