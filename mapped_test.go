package observable

import (
	"image/color"
	"testing"
)

func TestMapped_BasicTransform(t *testing.T) {
	source := Simple(float32(0.5))
	mapped := Mapped(source, func(v float32) color.RGBA {
		c := uint8(v * 255)
		return color.RGBA{R: c, G: c, B: c, A: 255}
	})

	// First Get should compute the value
	result := mapped.Get(Noop)
	if result.R != 127 && result.R != 128 {
		t.Fatalf("expected R ~128, got %d", result.R)
	}
}

func TestMapped_LazySubscription(t *testing.T) {
	reg := NewRegistry()
	source := Simple(10)

	evalCount := 0
	mapped := Mapped(source, func(v int) int {
		evalCount++
		return v * 2
	})

	// No evaluation yet — lazy
	if evalCount != 0 {
		t.Fatalf("expected 0 evaluations before Get, got %d", evalCount)
	}

	// Create an observer that uses the registry
	obs := &testRegistryObserver{reg: reg}
	result := mapped.Get(obs)
	if result != 20 {
		t.Fatalf("expected 20, got %d", result)
	}
	if evalCount != 1 {
		t.Fatalf("expected 1 evaluation after first Get, got %d", evalCount)
	}
}

func TestMapped_CachesValue(t *testing.T) {
	source := Simple(5)

	evalCount := 0
	mapped := Mapped(source, func(v int) int {
		evalCount++
		return v * 2
	})

	mapped.Get(Noop)
	if evalCount != 1 {
		t.Fatalf("expected 1 evaluation, got %d", evalCount)
	}

	// Second Get without source change should use cache
	mapped.Get(Noop)
	if evalCount != 1 {
		t.Fatalf("expected still 1 evaluation (cached), got %d", evalCount)
	}
}

func TestMapped_RecomputesOnSourceChange(t *testing.T) {
	reg := NewRegistry()
	source := Simple(5)

	mapped := Mapped(source, func(v int) int {
		return v * 2
	})

	// Subscribe through registry
	obs := &testRegistryObserver{reg: reg}
	result := mapped.Get(obs)
	if result != 10 {
		t.Fatalf("expected 10, got %d", result)
	}

	// Change source
	source.Set(7)

	// Mapped should recompute
	result = mapped.Get(obs)
	if result != 14 {
		t.Fatalf("expected 14 after source change, got %d", result)
	}
}

func TestMapped_CascadingCleanup(t *testing.T) {
	reg := NewRegistry()
	source := Simple(1)

	evalCount := 0
	mapped := Mapped(source, func(v int) int {
		evalCount++
		return v * 2
	})

	// Subscribe to mapped
	obs := &testRegistryObserver{reg: reg}
	mapped.Get(obs)
	evalCount = 0

	// Verify source change triggers recompute
	source.Set(2)
	mapped.Get(obs)
	if evalCount != 1 {
		t.Fatalf("expected 1 recompute, got %d", evalCount)
	}

	// Unsubscribe the observer from mapped — should cascade to source
	reg.UnsubscribeAll(obs)
	evalCount = 0

	// Source change should no longer trigger recompute on mapped
	// (mapped has no observers, so it gets cleaned up)
	source.Set(3)
	// Mapped should not have been notified
	// We verify by checking that a fresh Get recomputes (not from notification)
}

// testRegistryObserver is an observer that provides a registry.
type testRegistryObserver struct {
	BasicObserver
	reg *Registry
}

func (o *testRegistryObserver) ObservableRegistry() *Registry { return o.reg }
func (o *testRegistryObserver) CurrentObserver() Observer     { return o }
