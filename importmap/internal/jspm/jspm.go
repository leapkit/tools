package jspm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const baseURL = "https://api.jspm.io/generate"

type client struct{}

// GeneratorResult contains the imports, scopes, and staticDeps returned by JSPM.
type GeneratorResult struct {
	Imports    map[string]string
	Scopes     map[string]map[string]string
	StaticDeps []string
}

// GetImports returns the imports map.
func (r *GeneratorResult) GetImports() map[string]string {
	return r.Imports
}

// GetScopes returns the scopes map.
func (r *GeneratorResult) GetScopes() map[string]map[string]string {
	return r.Scopes
}

// GetStaticDeps returns the static dependencies list.
func (r *GeneratorResult) GetStaticDeps() []string {
	return r.StaticDeps
}

// Generate return the the import maps for the given packages.
func (c *client) Generate(ctx context.Context, packages ...string) (any, error) {
	payload := map[string]any{
		"install": packages,
		"env":     []string{"browser", "production", "module"},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	b, _ := io.ReadAll(res.Body)

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("generate response error: %s", b)
	}

	var response struct {
		StaticDeps  []string `json:"staticDeps"`
		DynamicDeps []string `json:"dynamicDeps"`
		Map         struct {
			Imports map[string]string            `json:"imports"`
			Scopes  map[string]map[string]string `json:"scopes"`
		} `json:"map"`
	}

	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("error decoding response body: %w", err)
	}

	return &GeneratorResult{
		Imports:    response.Map.Imports,
		Scopes:     response.Map.Scopes,
		StaticDeps: response.StaticDeps,
	}, nil
}

func Client() *client {
	return new(client)
}
