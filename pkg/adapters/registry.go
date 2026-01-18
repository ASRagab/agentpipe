package adapters

import (
	"fmt"
	"sync"
)

// AdapterRegistry manages registration and retrieval of adapter factories.
type AdapterRegistry struct {
	mu        sync.RWMutex
	factories map[string]AdapterFactory
}

// NewRegistry creates a new adapter registry.
func NewRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		factories: make(map[string]AdapterFactory),
	}
}

// Register adds an adapter factory to the registry.
func (r *AdapterRegistry) Register(name string, factory AdapterFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[name] = factory
}

// Get retrieves an adapter factory by name and creates a new instance.
func (r *AdapterRegistry) Get(name string) (AgentAdapter, error) {
	r.mu.RLock()
	factory, ok := r.factories[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("adapter not found: %s", name)
	}

	return factory(), nil
}

// Has checks if an adapter is registered.
func (r *AdapterRegistry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.factories[name]
	return ok
}

// List returns all registered adapter names.
func (r *AdapterRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// DefaultRegistry is the global adapter registry used by default.
var DefaultRegistry = NewRegistry()

// Register adds an adapter factory to the default registry.
func Register(name string, factory AdapterFactory) {
	DefaultRegistry.Register(name, factory)
}

// Get retrieves an adapter from the default registry.
func Get(name string) (AgentAdapter, error) {
	return DefaultRegistry.Get(name)
}

// Has checks if an adapter is registered in the default registry.
func Has(name string) bool {
	return DefaultRegistry.Has(name)
}

// List returns all adapter names from the default registry.
func List() []string {
	return DefaultRegistry.List()
}
