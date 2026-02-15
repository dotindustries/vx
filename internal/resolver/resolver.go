package resolver

import (
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"
)

const defaultMaxConcurrency = 10

// VaultReader abstracts reading key-value pairs from a Vault KV v2 path.
type VaultReader interface {
	ReadKV(path string) (map[string]string, error)
}

// Option configures a Resolver.
type Option func(*Resolver)

// WithMaxConcurrency sets the maximum number of concurrent Vault reads.
// Values less than 1 are ignored.
func WithMaxConcurrency(n int) Option {
	return func(r *Resolver) {
		if n > 0 {
			r.maxConcurrency = n
		}
	}
}

// WithCache attaches an in-memory cache to the resolver. Nil values are
// ignored.
func WithCache(c *Cache) Option {
	return func(r *Resolver) {
		if c != nil {
			r.cache = c
		}
	}
}

// Resolver resolves environment variable names to secret values by reading
// from Vault KV v2 paths. It groups secrets by path prefix and fetches
// each group concurrently.
type Resolver struct {
	vaultClient    VaultReader
	basePath       string
	maxConcurrency int
	cache          *Cache
}

// New creates a Resolver with the given VaultReader and base path.
// Functional options can override defaults.
func New(client VaultReader, basePath string, opts ...Option) *Resolver {
	r := &Resolver{
		vaultClient:    client,
		basePath:       basePath,
		maxConcurrency: defaultMaxConcurrency,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Resolve maps environment variable names to their secret values by reading
// from Vault. The secrets map keys are env var names and values are Vault
// path templates (e.g. "${env}/database/url"). The env parameter is
// interpolated into each path template.
//
// The input map is not mutated.
func (r *Resolver) Resolve(secrets map[string]string, env string) (map[string]string, error) {
	if len(secrets) == 0 {
		return map[string]string{}, nil
	}

	groups := GroupByPath(secrets, env)

	results, err := r.fetchAll(groups)
	if err != nil {
		return nil, fmt.Errorf("resolve secrets: %w", err)
	}

	return r.mapResults(groups, results), nil
}

// fetchAll reads all Vault paths concurrently with bounded concurrency.
// Returns a map of vault-path to its KV data.
func (r *Resolver) fetchAll(groups map[string][]SecretMapping) (map[string]map[string]string, error) {
	var mu sync.Mutex
	results := make(map[string]map[string]string, len(groups))

	g := new(errgroup.Group)
	g.SetLimit(r.maxConcurrency)

	for path := range groups {
		g.Go(r.fetchPath(path, &mu, results))
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

// fetchPath returns a function that reads a single Vault path and stores
// the result. It checks the cache first when available.
func (r *Resolver) fetchPath(
	path string,
	mu *sync.Mutex,
	results map[string]map[string]string,
) func() error {
	return func() error {
		data, err := r.readWithCache(path)
		if err != nil {
			return fmt.Errorf("read vault path %q: %w", path, err)
		}

		mu.Lock()
		results[path] = data
		mu.Unlock()

		return nil
	}
}

// readWithCache reads from cache first (if available), falling back to the
// Vault client.
func (r *Resolver) readWithCache(path string) (map[string]string, error) {
	fullPath := r.fullPath(path)

	if r.cache != nil {
		if data, ok := r.cache.Get(fullPath); ok {
			return data, nil
		}
	}

	data, err := r.vaultClient.ReadKV(fullPath)
	if err != nil {
		return nil, err
	}

	if r.cache != nil {
		r.cache.Set(fullPath, data)
	}

	return data, nil
}

// fullPath joins the base path with the given relative path.
func (r *Resolver) fullPath(path string) string {
	if r.basePath == "" {
		return path
	}

	return r.basePath + "/" + path
}

// mapResults builds the final env-var-to-value map from the fetched Vault
// data and the grouped secret mappings.
func (r *Resolver) mapResults(
	groups map[string][]SecretMapping,
	results map[string]map[string]string,
) map[string]string {
	resolved := make(map[string]string)

	for path, mappings := range groups {
		data := results[path]
		for _, m := range mappings {
			if val, ok := data[m.Key]; ok {
				resolved[m.EnvVar] = val
			}
		}
	}

	return resolved
}
