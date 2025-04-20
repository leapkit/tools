package npm

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
)

const baseURL = "https://registry.npmjs.org"

type client struct{}

func (c *client) Outdated(ctx context.Context, packages map[string]string) (map[string]string, error) {
	outdated := make(map[string]string)
	for pkg, version := range packages {
		latest, err := c.latestVersion(ctx, pkg)
		if err != nil {
			return nil, fmt.Errorf("error downloading %q package: %w", pkg, err)
		}

		if cmp.Compare(version, latest) == -1 {
			outdated[pkg] = latest
		}
	}

	return outdated, nil
}

func (c *client) latestVersion(ctx context.Context, pkg string) (string, error) {
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

func (c *client) Audit(ctx context.Context, packages map[string]string) (map[string][]map[string]string, error) {
	path, _ := url.JoinPath(baseURL, "-/npm/v1/security/advisories/bulk")

	data := make(map[string][]string)
	for pkg, version := range packages {
		data[pkg] = []string{version}
	}

	form, _ := json.Marshal(data)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, path, bytes.NewBuffer(form))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	b, _ := io.ReadAll(res.Body)
	var vulnerabilities map[string][]vulnerability
	if err := json.Unmarshal(b, &vulnerabilities); err != nil {
		return nil, err
	}

	vulnerabilityLevels := map[string]int{
		"low":      1,
		"moderate": 2,
		"high":     3,
		"critical": 4,
	}

	report := make(map[string][]map[string]string)
	for pkg := range vulnerabilities {
		slices.SortFunc(vulnerabilities[pkg], func(a vulnerability, b vulnerability) int {
			return cmp.Compare(vulnerabilityLevels[a.Severity], vulnerabilityLevels[b.Severity])
		})

		for _, cve := range vulnerabilities[pkg] {
			report[pkg] = append(report[pkg], map[string]string{
				"severity":            cve.Severity,
				"description":         cve.Description,
				"vulnerable_versions": cve.VulnerableVersions,
			})
		}
	}

	return report, nil
}

type vulnerability struct {
	ID                 int      `json:"id"`
	Url                string   `json:"url"`
	Description        string   `json:"title"`
	Severity           string   `json:"severity"`
	VulnerableVersions string   `json:"vulnerable_versions"`
	CWE                []string `json:"cwe"`
	CVSS               struct {
		Score        float64 `json:"score"`
		VectorString string  `json:"vectorString"`
	} `json:"cvss"`
}

func Client() *client {
	return new(client)
}
