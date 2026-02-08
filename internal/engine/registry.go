package engine

import (
	"fmt"
	"sync"
)

// Factory is a function that creates an Engine instance.
type Factory func(opts Options) (Engine, error)

// registry is the global engine registry instance.
var registry = &Registry{
	engines: make(map[string]Factory),
}

// Registry manages engine factories for different database dialects.
type Registry struct {
	mu      sync.RWMutex
	engines map[string]Factory
}

// Register adds an engine factory to the registry.
// Panics if the dialect is already registered.
func (r *Registry) Register(dialect string, factory Factory) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.engines[dialect]; exists {
		panic(fmt.Sprintf("engine: dialect %q already registered", dialect))
	}

	r.engines[dialect] = factory
}

// New creates an Engine for the specified dialect.
// Returns an error if the dialect is not registered.
func (r *Registry) New(dialect string, opts Options) (Engine, error) {
	r.mu.RLock()
	factory, exists := r.engines[dialect]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unsupported database dialect: %s", dialect)
	}

	return factory(opts)
}

// List returns all registered dialect names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	dialects := make([]string, 0, len(r.engines))
	for dialect := range r.engines {
		dialects = append(dialects, dialect)
	}

	return dialects
}

// IsRegistered reports whether a dialect is registered.
func (r *Registry) IsRegistered(dialect string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.engines[dialect]
	return exists
}

// Register allows external packages to register custom engines.
// This is useful for plugins or custom database support.
func Register(dialect string, factory Factory) {
	registry.Register(dialect, factory)
}

// ListRegistered returns all registered dialect names.
func ListRegistered() []string {
	return registry.List()
}

// IsDialectSupported reports whether a dialect is supported.
func IsDialectSupported(dialect string) bool {
	return registry.IsRegistered(dialect)
}
