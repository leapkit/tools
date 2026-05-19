package importmap

import "context"

// Bundler is the interface that wraps the Bundle method.
// Implementations are responsible for resolving package versions
// and fetching their bundled file contents.
type Bundler interface {
	// Bundle resolves and fetches the given packages, returning a map of
	// "name@version" -> file contents, e.g. "some-package@1.2.3" -> []byte.
	Bundle(ctx context.Context, packages ...string) (map[string][]byte, error)
}
