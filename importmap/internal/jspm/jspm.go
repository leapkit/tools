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

// Generate return the the import maps for the given packages.
func (c *client) Generate(ctx context.Context, packages ...string) (map[string]string, error) {
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
			Imports map[string]string `json:"imports"`
		} `json:"map"`
	}

	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("error decoding response body: %w", err)
	}

	return response.Map.Imports, nil
}

func Client() *client {
	return new(client)
}
