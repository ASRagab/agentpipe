package adapters

import (
	"context"
	"errors"
	"io"
	"sort"
	"sync"
	"testing"

	"github.com/ASRagab/agentpipe/pkg/v2/core"
)

// testAdapter is a simple adapter for testing purposes
type testAdapter struct {
	model           string
	initializeError error
	initialized     bool
}

func (t *testAdapter) Initialize(agent core.Agent) error {
	if t.initializeError != nil {
		return t.initializeError
	}
	t.initialized = true
	return nil
}

func (t *testAdapter) SendMessage(ctx context.Context, messages []core.Message) (string, *core.Metrics, error) {
	return "test response", nil, nil
}

func (t *testAdapter) StreamMessage(ctx context.Context, messages []core.Message, writer io.Writer) (*core.Metrics, error) {
	return nil, nil
}

func (t *testAdapter) IsAvailable() bool {
	return true
}

func (t *testAdapter) GetModel() string {
	return t.model
}

func (t *testAdapter) HealthCheck(ctx context.Context) error {
	return nil
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	if registry == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if registry.factories == nil {
		t.Error("factories map should be initialized")
	}
}

func TestRegister(t *testing.T) {
	registry := NewRegistry()

	registry.Register("test-adapter", func() AgentAdapter {
		return &testAdapter{model: "test-model"}
	})

	// Verify it's registered
	names := registry.List()
	if len(names) != 1 {
		t.Errorf("expected 1 registered adapter, got %d", len(names))
	}
	if names[0] != "test-adapter" {
		t.Errorf("expected 'test-adapter', got %q", names[0])
	}
}

func TestRegister_Overwrite(t *testing.T) {
	registry := NewRegistry()

	// Register first
	registry.Register("test", func() AgentAdapter {
		return &testAdapter{model: "model-1"}
	})

	// Overwrite
	registry.Register("test", func() AgentAdapter {
		return &testAdapter{model: "model-2"}
	})

	adapter, err := registry.Get("test")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if adapter.GetModel() != "model-2" {
		t.Errorf("expected model 'model-2', got %q", adapter.GetModel())
	}
}

func TestGet(t *testing.T) {
	registry := NewRegistry()

	registry.Register("test-adapter", func() AgentAdapter {
		return &testAdapter{model: "test-model"}
	})

	adapter, err := registry.Get("test-adapter")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}
	if adapter.GetModel() != "test-model" {
		t.Errorf("expected model 'test-model', got %q", adapter.GetModel())
	}
}

func TestGet_CreatesNewInstance(t *testing.T) {
	registry := NewRegistry()

	registry.Register("test-adapter", func() AgentAdapter {
		return &testAdapter{model: "test-model"}
	})

	adapter1, _ := registry.Get("test-adapter")
	adapter2, _ := registry.Get("test-adapter")

	// Should be different instances
	if adapter1 == adapter2 {
		t.Error("Get should return new instances each time")
	}
}

func TestGetUnknown(t *testing.T) {
	registry := NewRegistry()

	adapter, err := registry.Get("unknown")
	if err == nil {
		t.Error("expected error for unknown adapter")
	}
	if adapter != nil {
		t.Error("expected nil adapter for unknown")
	}
	if err.Error() != "adapter not found: unknown" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestHas(t *testing.T) {
	registry := NewRegistry()

	if registry.Has("test") {
		t.Error("should not have unregistered adapter")
	}

	registry.Register("test", func() AgentAdapter {
		return &testAdapter{}
	})

	if !registry.Has("test") {
		t.Error("should have registered adapter")
	}
}

func TestList(t *testing.T) {
	registry := NewRegistry()

	// Empty registry
	names := registry.List()
	if len(names) != 0 {
		t.Errorf("expected 0 names, got %d", len(names))
	}

	// Add adapters
	registry.Register("charlie", func() AgentAdapter { return &testAdapter{} })
	registry.Register("alpha", func() AgentAdapter { return &testAdapter{} })
	registry.Register("bravo", func() AgentAdapter { return &testAdapter{} })

	names = registry.List()
	if len(names) != 3 {
		t.Errorf("expected 3 names, got %d", len(names))
	}

	// Note: List() doesn't guarantee order, so we sort for comparison
	sorted := make([]string, len(names))
	copy(sorted, names)
	sort.Strings(sorted)

	expected := []string{"alpha", "bravo", "charlie"}
	for i, name := range expected {
		if sorted[i] != name {
			t.Errorf("expected %q at position %d, got %q", name, i, sorted[i])
		}
	}
}

func TestConcurrentAccess(t *testing.T) {
	registry := NewRegistry()

	var wg sync.WaitGroup

	// Concurrent registrations
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			name := string(rune('a' + n%26))
			registry.Register(name, func() AgentAdapter {
				return &testAdapter{model: name}
			})
		}(i)
	}

	// Concurrent gets
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			name := string(rune('a' + n%26))
			_, _ = registry.Get(name)
		}(i)
	}

	// Concurrent Has checks
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			name := string(rune('a' + n%26))
			_ = registry.Has(name)
		}(i)
	}

	// Concurrent List
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = registry.List()
		}()
	}

	wg.Wait()
	// Test passes if no race conditions detected with -race flag
}

// Tests for the default registry (package-level functions)

func TestDefaultRegistry_Register(t *testing.T) {
	// Register a test adapter
	Register("test-default-adapter", func() AgentAdapter {
		return &testAdapter{model: "default-model"}
	})
	defer func() {
		// Clean up - we can't really unregister, but we can overwrite
		// This is fine for testing purposes
	}()

	if !Has("test-default-adapter") {
		t.Error("adapter should be registered in default registry")
	}
}

func TestDefaultRegistry_Get(t *testing.T) {
	Register("test-default-get", func() AgentAdapter {
		return &testAdapter{model: "get-model"}
	})

	adapter, err := Get("test-default-get")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if adapter.GetModel() != "get-model" {
		t.Errorf("expected model 'get-model', got %q", adapter.GetModel())
	}
}

func TestDefaultRegistry_Has(t *testing.T) {
	if Has("definitely-not-registered-12345") {
		t.Error("should not have unregistered adapter")
	}
}

func TestDefaultRegistry_List(t *testing.T) {
	names := List()
	// Just verify it returns without error and includes some adapters
	if names == nil {
		t.Error("List should not return nil")
	}
}

// Test that Get returns a new adapter instance

func TestGetReturnsNewInstance(t *testing.T) {
	registry := NewRegistry()

	var creationCount int
	registry.Register("counter", func() AgentAdapter {
		creationCount++
		return &testAdapter{}
	})

	_, _ = registry.Get("counter")
	_, _ = registry.Get("counter")
	_, _ = registry.Get("counter")

	if creationCount != 3 {
		t.Errorf("expected 3 adapter creations, got %d", creationCount)
	}
}

// Test with an adapter that fails to initialize
// Note: The registry itself doesn't call Initialize - that's done by the caller

func TestAdapterFactoryCreatesAdapter(t *testing.T) {
	registry := NewRegistry()

	initErr := errors.New("init failed")
	registry.Register("failing", func() AgentAdapter {
		return &testAdapter{initializeError: initErr}
	})

	adapter, err := registry.Get("failing")
	if err != nil {
		t.Fatalf("Get should succeed even if adapter has init error: %v", err)
	}

	// The caller is responsible for initializing
	agent := core.Agent{ID: "test"}
	err = adapter.Initialize(agent)
	if err == nil {
		t.Error("Initialize should return error")
	}
	if !errors.Is(err, initErr) {
		t.Errorf("expected initErr, got %v", err)
	}
}
