package observable

import (
	"testing"
)

// Mock observer that implements RegistryProvider for registry path
type benchRegistryObserver struct {
	BasicObserver
	registry *Registry
}

func (o *benchRegistryObserver) ObservableRegistry() *Registry {
	return o.registry
}

func (o *benchRegistryObserver) CurrentObserver() Observer {
	return o
}

// =============================================================================
// Subscribe Benchmarks
// =============================================================================

func BenchmarkSubscribe(b *testing.B) {
	b.Run("registry_first", func(b *testing.B) {
		reg := NewRegistry()
		observers := make([]*benchRegistryObserver, b.N)
		for i := range observers {
			observers[i] = &benchRegistryObserver{registry: reg}
		}
		values := make([]*SimpleValue[int], b.N)
		for i := range values {
			values[i] = Simple(i)
		}

		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			values[i].Get(observers[i])
		}
	})

	b.Run("registry_repeat", func(b *testing.B) {
		reg := NewRegistry()
		obs := &benchRegistryObserver{registry: reg}
		v := Simple(42)
		v.Get(obs) // first subscription

		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			v.Get(obs) // repeat subscription (should be fast path)
		}
	})

	b.Run("weak_pointer_first", func(b *testing.B) {
		observers := make([]*BasicObserver, b.N)
		for i := range observers {
			observers[i] = &BasicObserver{}
		}
		values := make([]*SimpleValue[int], b.N)
		for i := range values {
			values[i] = Simple(i)
		}

		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			values[i].Get(observers[i])
		}
	})

	b.Run("weak_pointer_repeat", func(b *testing.B) {
		obs := &BasicObserver{}
		v := Simple(42)
		v.Get(obs) // first subscription

		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			v.Get(obs) // repeat subscription
		}
	})
}

// =============================================================================
// Notify Benchmarks
// =============================================================================

func BenchmarkNotify(b *testing.B) {
	for _, numObservers := range []int{1, 10, 100} {
		b.Run(sprintf("registry_%d_observers", numObservers), func(b *testing.B) {
			reg := NewRegistry()
			v := Simple(0)

			observers := make([]*benchRegistryObserver, numObservers)
			for i := range observers {
				observers[i] = &benchRegistryObserver{registry: reg}
				v.Get(observers[i])
			}

			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				v.Set(i)
			}
		})

		b.Run(sprintf("weak_pointer_%d_observers", numObservers), func(b *testing.B) {
			v := Simple(0)

			observers := make([]*BasicObserver, numObservers)
			for i := range observers {
				observers[i] = &BasicObserver{}
				v.Get(observers[i])
			}

			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				v.Set(i)
			}
		})
	}
}

// =============================================================================
// Get/Set Cycle Benchmarks
// =============================================================================

func BenchmarkGetSet(b *testing.B) {
	b.Run("registry", func(b *testing.B) {
		reg := NewRegistry()
		obs := &benchRegistryObserver{registry: reg}
		v := Simple(0)

		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			v.Set(i)
			_ = v.Get(obs)
		}
	})

	b.Run("weak_pointer", func(b *testing.B) {
		obs := &BasicObserver{}
		v := Simple(0)

		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			v.Set(i)
			_ = v.Get(obs)
		}
	})
}

// =============================================================================
// Computed Chain Benchmarks
// =============================================================================

func BenchmarkComputedChain(b *testing.B) {
	b.Run("registry_depth_3", func(b *testing.B) {
		reg := NewRegistry()
		obs := &benchRegistryObserver{registry: reg}

		source := Simple(1)
		c1 := NewComputed(func(o Observer) int { return source.Get(o) * 2 })
		c2 := NewComputed(func(o Observer) int { return c1.Get(o) + 10 })
		c3 := NewComputed(func(o Observer) int { return c2.Get(o) * 3 })

		// Initial subscription
		_ = c3.Get(obs)

		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			source.Set(i)
			_ = c3.Get(obs)
		}
	})

	b.Run("weak_pointer_depth_3", func(b *testing.B) {
		obs := &BasicObserver{}

		source := Simple(1)
		c1 := NewComputed(func(o Observer) int { return source.Get(o) * 2 })
		c2 := NewComputed(func(o Observer) int { return c1.Get(o) + 10 })
		c3 := NewComputed(func(o Observer) int { return c2.Get(o) * 3 })

		// Initial subscription
		_ = c3.Get(obs)

		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			source.Set(i)
			_ = c3.Get(obs)
		}
	})

	b.Run("registry_depth_5", func(b *testing.B) {
		reg := NewRegistry()
		obs := &benchRegistryObserver{registry: reg}

		source := Simple(1)
		c1 := NewComputed(func(o Observer) int { return source.Get(o) * 2 })
		c2 := NewComputed(func(o Observer) int { return c1.Get(o) + 10 })
		c3 := NewComputed(func(o Observer) int { return c2.Get(o) * 3 })
		c4 := NewComputed(func(o Observer) int { return c3.Get(o) - 5 })
		c5 := NewComputed(func(o Observer) int { return c4.Get(o) / 2 })

		_ = c5.Get(obs)

		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			source.Set(i + 1) // avoid division issues
			_ = c5.Get(obs)
		}
	})

	b.Run("weak_pointer_depth_5", func(b *testing.B) {
		obs := &BasicObserver{}

		source := Simple(1)
		c1 := NewComputed(func(o Observer) int { return source.Get(o) * 2 })
		c2 := NewComputed(func(o Observer) int { return c1.Get(o) + 10 })
		c3 := NewComputed(func(o Observer) int { return c2.Get(o) * 3 })
		c4 := NewComputed(func(o Observer) int { return c3.Get(o) - 5 })
		c5 := NewComputed(func(o Observer) int { return c4.Get(o) / 2 })

		_ = c5.Get(obs)

		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			source.Set(i + 1)
			_ = c5.Get(obs)
		}
	})
}

// =============================================================================
// Cleanup Benchmarks
// =============================================================================

func BenchmarkCleanup(b *testing.B) {
	b.Run("unsubscribe_simple", func(b *testing.B) {
		// Pre-allocate all test data
		regs := make([]*Registry, b.N)
		observers := make([]*benchRegistryObserver, b.N)
		values := make([]*SimpleValue[int], b.N)

		for i := 0; b.Loop(); i++ {
			regs[i] = NewRegistry()
			observers[i] = &benchRegistryObserver{registry: regs[i]}
			values[i] = Simple(42)
			values[i].Get(observers[i])
		}

		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			regs[i].UnsubscribeAll(observers[i])
		}
	})

	b.Run("unsubscribe_computed_chain_3", func(b *testing.B) {
		// Pre-allocate all test data
		regs := make([]*Registry, b.N)
		observers := make([]*benchRegistryObserver, b.N)

		for i := 0; b.Loop(); i++ {
			regs[i] = NewRegistry()
			observers[i] = &benchRegistryObserver{registry: regs[i]}

			source := Simple(1)
			c1 := NewComputed(func(o Observer) int { return source.Get(o) * 2 })
			c2 := NewComputed(func(o Observer) int { return c1.Get(o) + 10 })
			c3 := NewComputed(func(o Observer) int { return c2.Get(o) * 3 })
			c3.Get(observers[i])
		}

		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			regs[i].UnsubscribeAll(observers[i])
		}
	})

	b.Run("unsubscribe_many_observables", func(b *testing.B) {
		// Pre-allocate all test data
		regs := make([]*Registry, b.N)
		observers := make([]*benchRegistryObserver, b.N)

		for i := 0; b.Loop(); i++ {
			regs[i] = NewRegistry()
			observers[i] = &benchRegistryObserver{registry: regs[i]}

			for j := 0; j < 100; j++ {
				v := Simple(j)
				v.Get(observers[i])
			}
		}

		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			regs[i].UnsubscribeAll(observers[i])
		}
	})
}

// =============================================================================
// Memory Allocation Benchmarks
// =============================================================================

func BenchmarkMemoryAlloc(b *testing.B) {
	b.Run("registry_subscribe", func(b *testing.B) {
		reg := NewRegistry()
		obs := &benchRegistryObserver{registry: reg}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			v := Simple(i)
			v.Get(obs)
		}
	})

	b.Run("weak_pointer_subscribe", func(b *testing.B) {
		obs := &BasicObserver{}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			v := Simple(i)
			v.Get(obs)
		}
	})

	b.Run("registry_notify", func(b *testing.B) {
		reg := NewRegistry()
		obs := &benchRegistryObserver{registry: reg}
		v := Simple(0)
		v.Get(obs)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			v.Set(i)
		}
	})

	b.Run("weak_pointer_notify", func(b *testing.B) {
		obs := &BasicObserver{}
		v := Simple(0)
		v.Get(obs)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			v.Set(i)
		}
	})
}

// =============================================================================
// Realistic UI Scenario Benchmark
// =============================================================================

func BenchmarkUIScenario(b *testing.B) {
	// Simulates: widget observes computed that depends on 3 values
	b.Run("registry_widget_with_3_deps", func(b *testing.B) {
		reg := NewRegistry()
		widget := &benchRegistryObserver{registry: reg}

		name := Simple("John")
		age := Simple(30)
		active := Simple(true)

		computed := NewComputed(func(o Observer) string {
			n := name.Get(o)
			a := age.Get(o)
			act := active.Get(o)
			if act {
				return sprintf("%s (%d) - active", n, a)
			}
			return sprintf("%s (%d) - inactive", n, a)
		})

		_ = computed.Get(widget)

		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			age.Set(30 + i%50)
			_ = computed.Get(widget)
		}
	})

	b.Run("weak_pointer_widget_with_3_deps", func(b *testing.B) {
		widget := &BasicObserver{}

		name := Simple("John")
		age := Simple(30)
		active := Simple(true)

		computed := NewComputed(func(o Observer) string {
			n := name.Get(o)
			a := age.Get(o)
			act := active.Get(o)
			if act {
				return sprintf("%s (%d) - active", n, a)
			}
			return sprintf("%s (%d) - inactive", n, a)
		})

		_ = computed.Get(widget)

		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			age.Set(30 + i%50)
			_ = computed.Get(widget)
		}
	})
}

// sprintf is a simple helper to avoid importing fmt in benchmarks
func sprintf(format string, args ...any) string {
	// Simple implementation for benchmark strings
	switch format {
	case "registry_%d_observers":
		return "registry_" + itoa(args[0].(int)) + "_observers"
	case "weak_pointer_%d_observers":
		return "weak_pointer_" + itoa(args[0].(int)) + "_observers"
	case "%s (%d) - active":
		return args[0].(string) + " (" + itoa(args[1].(int)) + ") - active"
	case "%s (%d) - inactive":
		return args[0].(string) + " (" + itoa(args[1].(int)) + ") - inactive"
	}
	return format
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	if i < 0 {
		return "-" + itoa(-i)
	}
	digits := ""
	for i > 0 {
		digits = string(rune('0'+i%10)) + digits
		i /= 10
	}
	return digits
}
