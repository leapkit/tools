package jspm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const baseURL = "https://api.jspm.io/generate"

type client struct {
	env      string
	provider string
}

// Generate return the the import maps for the given packages.
func (c *client) Generate(ctx context.Context, packages ...string) (map[string]string, error) {
	form := struct {
		Install      []string `json:"install"`
		Env          []string `json:"env"`
		Provider     string   `json:"provider"`
		FlattenScope string   `json:"flattenScope"`
	}{
		Install:      packages,
		Env:          []string{"browser", "module", c.env},
		FlattenScope: "true",
		Provider:     c.provider,
	}

	formBytes, _ := json.Marshal(&form)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, bytes.NewReader(formBytes))
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	b, _ := io.ReadAll(res.Body)

	if strings.Contains(string(b), "error") {
		return nil, fmt.Errorf("%s", string(b))
	}

	var response struct {
		StaticDeps  []string `json:"staticDeps"`
		DynamicDeps []string `json:"dynamicDeps"`
		Map         struct {
			Imports map[string]string `json:"imports"`
		} `json:"map"`
	}

	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("error decoding response body: %w", err)
	}

	return response.Map.Imports, nil
}

type Option func(*client)

func WithEnv(env string) Option {
	return func(c *client) {
		c.env = env
	}
}

func WithProvider(provider string) Option {
	return func(c *client) {
		c.provider = provider
	}
}

func Client(options ...Option) *client {
	j := &client{
		env:      "production",
		provider: "jspm.io",
	}

	for _, option := range options {
		option(j)
	}

	return j
}
