package observable

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Map", func() {
	Describe("concurrency", func() {
		It("handles concurrent writes", func(ctx SpecContext) {
			m := NewMap[int, int]()
			done := make(chan bool)

			// Multiple writers with different key ranges
			for i := 0; i < 10; i++ {
				go func(id int) {
					for j := 0; j < 100; j++ {
						m.Set(id*1000+j, j)
					}
					done <- true
				}(i)
			}

			for i := 0; i < 10; i++ {
				select {
				case <-done:
				case <-ctx.Done():
					Fail("timeout")
				}
			}

			Expect(m.Len()).To(Equal(1000))
		}, SpecTimeout(5*time.Second))

		It("handles concurrent reads and writes", func(ctx SpecContext) {
			m := NewMap[string, int]()
			m.Set("a", 1)
			m.Set("b", 2)
			m.Set("c", 3)
			done := make(chan bool)

			// Writer
			go func() {
				for i := 0; i < 100; i++ {
					m.Set("a", i)
					m.Set("temp", i)
					m.Delete("temp")
				}
				done <- true
			}()

			// Readers
			for i := 0; i < 5; i++ {
				go func() {
					obs := &BasicObserver{}
					getter := m.Observe(obs)
					for j := 0; j < 100; j++ {
						_ = getter.Len()
						_ = getter.Keys()
						_ = getter.Values()
						_, _ = getter.Get("a")
						_ = getter.Has("b")
					}
					done <- true
				}()
			}

			for i := 0; i < 6; i++ {
				select {
				case <-done:
				case <-ctx.Done():
					Fail("timeout")
				}
			}
		}, SpecTimeout(5*time.Second))

		It("handles concurrent Set on same key", func(ctx SpecContext) {
			m := NewMap[string, int]()
			done := make(chan bool)

			// Multiple writers updating same key
			for i := 0; i < 10; i++ {
				go func(id int) {
					for j := 0; j < 100; j++ {
						m.Set("shared", id*1000+j)
					}
					done <- true
				}(i)
			}

			for i := 0; i < 10; i++ {
				select {
				case <-done:
				case <-ctx.Done():
					Fail("timeout")
				}
			}

			// Should have exactly one key
			Expect(m.Len()).To(Equal(1))
		}, SpecTimeout(5*time.Second))

		It("handles concurrent Merge", func(ctx SpecContext) {
			m := NewMap[int, int]()
			done := make(chan bool)

			for i := 0; i < 10; i++ {
				go func(id int) {
					for j := 0; j < 50; j++ {
						m.Merge(map[int]int{
							id*100 + j:     j,
							id*100 + j + 1: j + 1,
						})
					}
					done <- true
				}(i)
			}

			for i := 0; i < 10; i++ {
				select {
				case <-done:
				case <-ctx.Done():
					Fail("timeout")
				}
			}

			// Each goroutine adds ~100 unique keys (some overlap at +1)
			Expect(m.Len()).To(BeNumerically(">=", 500))
		}, SpecTimeout(5*time.Second))

		It("handles concurrent Replace", func(ctx SpecContext) {
			m := NewMap[string, int]()
			m.Set("init", 0)
			done := make(chan bool)

			for i := 0; i < 10; i++ {
				go func(id int) {
					for j := 0; j < 50; j++ {
						m.Replace(map[string]int{
							"a": id,
							"b": j,
						})
					}
					done <- true
				}(i)
			}

			for i := 0; i < 10; i++ {
				select {
				case <-done:
				case <-ctx.Done():
					Fail("timeout")
				}
			}

			// After Replace, should have exactly 2 keys
			Expect(m.Len()).To(Equal(2))
		}, SpecTimeout(5*time.Second))

		It("handles observer notifications under contention", func(ctx SpecContext) {
			m := NewMap[string, int]()
			reg := NewRegistry()
			done := make(chan bool)
			notifyCount := make(chan int, 1000)

			// Multiple observers
			for i := 0; i < 5; i++ {
				widget := &BasicObserver{}
				widget.OnChange = func() {
					notifyCount <- 1
				}
				mockCtx := &mockRegistryProvider{registry: reg, observer: widget}
				m.Observe(mockCtx)
			}

			// Concurrent modifications
			for i := 0; i < 5; i++ {
				go func(id int) {
					for j := 0; j < 20; j++ {
						m.Set(string(rune('a'+id))+string(rune('0'+j)), j)
					}
					done <- true
				}(i)
			}

			for i := 0; i < 5; i++ {
				select {
				case <-done:
				case <-ctx.Done():
					Fail("timeout")
				}
			}

			close(notifyCount)
			total := 0
			for n := range notifyCount {
				total += n
			}
			// Each Set notifies all 5 observers, 100 sets total
			Expect(total).To(Equal(500))
		}, SpecTimeout(5*time.Second))

		It("handles concurrent Delete", func(ctx SpecContext) {
			m := NewMap[int, int]()
			// Pre-populate
			for i := 0; i < 100; i++ {
				m.Set(i, i)
			}
			done := make(chan bool)

			// Concurrent deleters
			for i := 0; i < 10; i++ {
				go func(id int) {
					for j := 0; j < 100; j++ {
						m.Delete(j) // All trying to delete same keys
					}
					done <- true
				}(i)
			}

			for i := 0; i < 10; i++ {
				select {
				case <-done:
				case <-ctx.Done():
					Fail("timeout")
				}
			}

			Expect(m.Len()).To(Equal(0))
		}, SpecTimeout(5*time.Second))
	})
	Describe("basic operations", func() {
		It("starts empty", func() {
			m := NewMap[string, int]()
			Expect(m.Len()).To(Equal(0))
		})

		It("Set adds items", func() {
			m := NewMap[string, int]()
			m.Set("a", 1)
			m.Set("b", 2)

			Expect(m.Len()).To(Equal(2))

			obs := &BasicObserver{}
			getter := m.Observe(obs)

			val, ok := getter.Get("a")
			Expect(ok).To(BeTrue())
			Expect(val).To(Equal(1))

			val, ok = getter.Get("b")
			Expect(ok).To(BeTrue())
			Expect(val).To(Equal(2))
		})

		It("Set replaces existing key", func() {
			m := NewMap[string, int]()
			m.Set("a", 1)
			m.Set("a", 2)

			Expect(m.Len()).To(Equal(1))

			obs := &BasicObserver{}
			getter := m.Observe(obs)

			val, ok := getter.Get("a")
			Expect(ok).To(BeTrue())
			Expect(val).To(Equal(2))
		})

		It("Delete removes item", func() {
			m := NewMap[string, int]()
			m.Set("a", 1)
			m.Set("b", 2)

			Expect(m.Delete("a")).To(BeTrue())
			Expect(m.Len()).To(Equal(1))

			obs := &BasicObserver{}
			getter := m.Observe(obs)

			_, ok := getter.Get("a")
			Expect(ok).To(BeFalse())

			val, ok := getter.Get("b")
			Expect(ok).To(BeTrue())
			Expect(val).To(Equal(2))
		})

		It("Delete returns false for non-existent key", func() {
			m := NewMap[string, int]()
			m.Set("a", 1)
			Expect(m.Delete("b")).To(BeFalse())
			Expect(m.Len()).To(Equal(1))
		})
	})

	Describe("Replace", func() {
		It("replaces all items", func() {
			m := NewMap[string, int]()
			m.Set("a", 1)
			m.Set("b", 2)

			m.Replace(map[string]int{"x": 10, "y": 20, "z": 30})

			Expect(m.Len()).To(Equal(3))

			obs := &BasicObserver{}
			getter := m.Observe(obs)

			_, ok := getter.Get("a")
			Expect(ok).To(BeFalse())

			val, ok := getter.Get("x")
			Expect(ok).To(BeTrue())
			Expect(val).To(Equal(10))
		})

		It("can replace with empty", func() {
			m := NewMap[string, int]()
			m.Set("a", 1)
			m.Replace(map[string]int{})
			Expect(m.Len()).To(Equal(0))
		})
	})

	Describe("Merge", func() {
		It("adds new items without removing existing", func() {
			m := NewMap[string, int]()
			m.Set("a", 1)
			m.Set("b", 2)

			m.Merge(map[string]int{"c": 3, "d": 4})

			Expect(m.Len()).To(Equal(4))

			obs := &BasicObserver{}
			getter := m.Observe(obs)

			val, _ := getter.Get("a")
			Expect(val).To(Equal(1))
			val, _ = getter.Get("c")
			Expect(val).To(Equal(3))
		})

		It("updates existing keys", func() {
			m := NewMap[string, int]()
			m.Set("a", 1)
			m.Set("b", 2)

			m.Merge(map[string]int{"a": 10, "c": 3})

			Expect(m.Len()).To(Equal(3))

			obs := &BasicObserver{}
			getter := m.Observe(obs)

			val, _ := getter.Get("a")
			Expect(val).To(Equal(10))
		})

		It("empty merge is no-op", func() {
			m := NewMap[string, int]()
			m.Set("a", 1)

			obs := &BasicObserver{}
			m.Observe(obs)
			obs.GetAndResetUpdated()

			m.Merge(map[string]int{})
			Expect(obs.IsUpdated()).To(BeFalse())
		})
	})

	Describe("Clear", func() {
		It("removes all items", func() {
			m := NewMap[string, int]()
			m.Set("a", 1)
			m.Set("b", 2)
			m.Clear()
			Expect(m.Len()).To(Equal(0))
		})

		It("no-op on empty map", func() {
			m := NewMap[string, int]()

			obs := &BasicObserver{}
			m.Observe(obs)
			obs.GetAndResetUpdated()

			m.Clear()
			Expect(obs.IsUpdated()).To(BeFalse())
		})
	})

	Describe("MapGetter", func() {
		It("Has checks key existence", func() {
			m := NewMap[string, int]()
			m.Set("a", 1)

			obs := &BasicObserver{}
			getter := m.Observe(obs)

			Expect(getter.Has("a")).To(BeTrue())
			Expect(getter.Has("b")).To(BeFalse())
		})

		It("Keys returns all keys", func() {
			m := NewMap[string, int]()
			m.Set("a", 1)
			m.Set("b", 2)
			m.Set("c", 3)

			obs := &BasicObserver{}
			getter := m.Observe(obs)

			keys := getter.Keys()
			Expect(keys).To(HaveLen(3))
			Expect(keys).To(ContainElements("a", "b", "c"))
		})

		It("Values returns all values", func() {
			m := NewMap[string, int]()
			m.Set("a", 1)
			m.Set("b", 2)

			obs := &BasicObserver{}
			getter := m.Observe(obs)

			values := getter.Values()
			Expect(values).To(HaveLen(2))
			Expect(values).To(ContainElements(1, 2))
		})

		It("All iterates key-value pairs", func() {
			m := NewMap[string, int]()
			m.Set("a", 1)
			m.Set("b", 2)

			obs := &BasicObserver{}
			getter := m.Observe(obs)

			collected := make(map[string]int)
			for k, v := range getter.All() {
				collected[k] = v
			}
			Expect(collected).To(Equal(map[string]int{"a": 1, "b": 2}))
		})
	})

	Describe("observer notifications", func() {
		It("Set notifies observers", func() {
			m := NewMap[string, int]()
			obs := &BasicObserver{}
			m.Observe(obs)
			obs.GetAndResetUpdated()

			m.Set("a", 1)
			Expect(obs.IsUpdated()).To(BeTrue())
		})

		It("Set on existing key notifies observers", func() {
			m := NewMap[string, int]()
			m.Set("a", 1)
			obs := &BasicObserver{}
			m.Observe(obs)
			obs.GetAndResetUpdated()

			m.Set("a", 2)
			Expect(obs.IsUpdated()).To(BeTrue())
		})

		It("Delete notifies observers", func() {
			m := NewMap[string, int]()
			m.Set("a", 1)
			obs := &BasicObserver{}
			m.Observe(obs)
			obs.GetAndResetUpdated()

			m.Delete("a")
			Expect(obs.IsUpdated()).To(BeTrue())
		})

		It("Delete on non-existent does not notify", func() {
			m := NewMap[string, int]()
			m.Set("a", 1)
			obs := &BasicObserver{}
			m.Observe(obs)
			obs.GetAndResetUpdated()

			m.Delete("b")
			Expect(obs.IsUpdated()).To(BeFalse())
		})

		It("Replace notifies observers", func() {
			m := NewMap[string, int]()
			m.Set("a", 1)
			obs := &BasicObserver{}
			m.Observe(obs)
			obs.GetAndResetUpdated()

			m.Replace(map[string]int{"x": 10})
			Expect(obs.IsUpdated()).To(BeTrue())
		})

		It("Merge notifies observers", func() {
			m := NewMap[string, int]()
			m.Set("a", 1)
			obs := &BasicObserver{}
			m.Observe(obs)
			obs.GetAndResetUpdated()

			m.Merge(map[string]int{"b": 2})
			Expect(obs.IsUpdated()).To(BeTrue())
		})

		It("Clear notifies observers", func() {
			m := NewMap[string, int]()
			m.Set("a", 1)
			obs := &BasicObserver{}
			m.Observe(obs)
			obs.GetAndResetUpdated()

			m.Clear()
			Expect(obs.IsUpdated()).To(BeTrue())
		})
	})

	Describe("subscribing reads", func() {
		It("Get subscribes the observer", func() {
			m := NewMap[string, int]()
			m.Set("a", 1)

			obs := &BasicObserver{}
			val, ok := m.Get(obs, "a")
			Expect(ok).To(BeTrue())
			Expect(val).To(Equal(1))
			obs.GetAndResetUpdated() // clear initial dirty state

			m.Set("a", 2)
			Expect(obs.GetAndResetUpdated()).To(BeTrue())
		})

		It("Get returns false for missing key", func() {
			m := NewMap[string, int]()
			obs := &BasicObserver{}
			_, ok := m.Get(obs, "missing")
			Expect(ok).To(BeFalse())
		})

		It("All subscribes the observer and iterates all entries", func() {
			m := NewMap[string, int]()
			m.Set("x", 10)

			obs := &BasicObserver{}
			got := make(map[string]int)
			for k, v := range m.All(obs) {
				got[k] = v
			}
			Expect(got).To(Equal(map[string]int{"x": 10}))

			m.Set("y", 20)
			Expect(obs.GetAndResetUpdated()).To(BeTrue())
		})
	})

	Describe("registry integration", func() {
		It("subscribes through registry when available", func() {
			reg := NewRegistry()
			m := NewMap[string, int]()
			m.Set("a", 1)

			widget := &BasicObserver{}
			ctx := &mockRegistryProvider{registry: reg, observer: widget}

			getter := m.Observe(ctx)
			Expect(getter.Len()).To(Equal(1))
			Expect(hasSubscription(reg, widget, m)).To(BeTrue())
		})
	})
})

func TestMap_SatisfiesROMap(t *testing.T) {
	m := NewMap[string, int]()
	m.Set("a", 1)
	m.Set("b", 2)

	var ro ROMap[string, int] = m
	if ro.PeekLen() != 2 {
		t.Fatalf("expected 2, got %d", ro.PeekLen())
	}
	v, ok := ro.Peek("a")
	if !ok || v != 1 {
		t.Fatalf("expected (1, true), got (%d, %v)", v, ok)
	}

	count := 0
	for range ro.PeekAll() {
		count++
	}
	if count != 2 {
		t.Fatalf("expected 2 items from PeekAll, got %d", count)
	}
}
