package observable

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("List", func() {
	Describe("concurrency", func() {
		It("handles concurrent writes", func(ctx SpecContext) {
			list := NewList[int]()
			done := make(chan bool)

			// Multiple writers
			for i := 0; i < 10; i++ {
				go func(id int) {
					for j := 0; j < 100; j++ {
						list.Add(id*1000 + j)
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

			Expect(list.PeekLen()).To(Equal(1000))
		}, SpecTimeout(5*time.Second))

		It("handles concurrent reads and writes", func(ctx SpecContext) {
			list := NewList[int]()
			list.Add(1, 2, 3, 4, 5)
			done := make(chan bool)

			// Writer
			go func() {
				for i := 0; i < 100; i++ {
					list.Add(i)
					list.Set(0, i)
					if list.PeekLen() > 10 {
						list.RemoveAt(list.PeekLen() - 1)
					}
				}
				done <- true
			}()

			// Readers
			for i := 0; i < 5; i++ {
				go func() {
					obs := &BasicObserver{}
					getter := list.Observe(obs)
					for j := 0; j < 100; j++ {
						_ = getter.Len()
						_ = getter.Keys()
						_ = getter.Values()
						_, _, _ = getter.At(0)
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

		It("handles concurrent Move and Swap", func(ctx SpecContext) {
			list := NewList[int]()
			for i := 0; i < 20; i++ {
				list.Add(i)
			}
			done := make(chan bool)

			// Swappers
			for i := 0; i < 5; i++ {
				go func() {
					for j := 0; j < 50; j++ {
						list.Swap(j%10, (j+5)%10)
					}
					done <- true
				}()
			}

			// Movers
			for i := 0; i < 5; i++ {
				go func() {
					for j := 0; j < 50; j++ {
						list.Move(j%10, (j+3)%10)
					}
					done <- true
				}()
			}

			for i := 0; i < 10; i++ {
				select {
				case <-done:
				case <-ctx.Done():
					Fail("timeout")
				}
			}

			// List should still have 20 items (Move/Swap preserve count)
			Expect(list.PeekLen()).To(Equal(20))
		}, SpecTimeout(5*time.Second))

		It("handles concurrent Replace", func(ctx SpecContext) {
			list := NewList[int]()
			list.Add(1, 2, 3)
			done := make(chan bool)

			for i := 0; i < 10; i++ {
				go func(id int) {
					for j := 0; j < 50; j++ {
						list.Replace([]int{id, j, id + j})
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

			Expect(list.PeekLen()).To(Equal(3))
		}, SpecTimeout(5*time.Second))

		It("handles observer notifications under contention", func(ctx SpecContext) {
			list := NewList[int]()
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
				list.Observe(mockCtx)
			}

			// Concurrent modifications
			for i := 0; i < 5; i++ {
				go func(id int) {
					for j := 0; j < 20; j++ {
						list.Add(id*100 + j)
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
			// Each Add notifies all 5 observers, 100 adds total
			Expect(total).To(Equal(500))
		}, SpecTimeout(5*time.Second))
	})
	Describe("basic operations", func() {
		It("starts empty", func() {
			list := NewList[string]()
			Expect(list.PeekLen()).To(Equal(0))
		})

		It("Add appends items with unique keys", func() {
			list := NewList[string]()
			list.Add("a", "b", "c")

			Expect(list.PeekLen()).To(Equal(3))

			obs := &BasicObserver{}
			getter := list.Observe(obs)

			keys := getter.Keys()
			Expect(keys).To(HaveLen(3))
			Expect(keys[0]).NotTo(Equal(keys[1]))
			Expect(keys[1]).NotTo(Equal(keys[2]))

			values := getter.Values()
			Expect(values).To(Equal([]string{"a", "b", "c"}))
		})

		It("Insert adds items at index", func() {
			list := NewList[int]()
			list.Add(1, 3)
			list.Insert(1, 2)

			obs := &BasicObserver{}
			getter := list.Observe(obs)
			Expect(getter.Values()).To(Equal([]int{1, 2, 3}))
		})

		It("Insert at beginning", func() {
			list := NewList[int]()
			list.Add(2, 3)
			list.Insert(0, 1)

			obs := &BasicObserver{}
			getter := list.Observe(obs)
			Expect(getter.Values()).To(Equal([]int{1, 2, 3}))
		})

		It("Insert at end", func() {
			list := NewList[int]()
			list.Add(1, 2)
			list.Insert(2, 3)

			obs := &BasicObserver{}
			getter := list.Observe(obs)
			Expect(getter.Values()).To(Equal([]int{1, 2, 3}))
		})

		It("Insert fails with invalid index", func() {
			list := NewList[int]()
			list.Add(1)
			Expect(list.Insert(-1, 0)).To(BeFalse())
			Expect(list.Insert(5, 0)).To(BeFalse())
		})

		It("RemoveAt removes item", func() {
			list := NewList[string]()
			list.Add("a", "b", "c")
			list.RemoveAt(1)

			obs := &BasicObserver{}
			getter := list.Observe(obs)
			Expect(getter.Values()).To(Equal([]string{"a", "c"}))
		})

		It("RemoveAt fails with invalid index", func() {
			list := NewList[int]()
			list.Add(1)
			Expect(list.RemoveAt(-1)).To(BeFalse())
			Expect(list.RemoveAt(5)).To(BeFalse())
		})

		It("RemoveByKey removes correct item", func() {
			list := NewList[string]()
			list.Add("a", "b", "c")

			obs := &BasicObserver{}
			getter := list.Observe(obs)
			keys := getter.Keys()

			Expect(list.RemoveByKey(keys[1])).To(BeTrue())
			Expect(getter.Values()).To(Equal([]string{"a", "c"}))
		})

		It("RemoveByKey returns false for unknown key", func() {
			list := NewList[int]()
			list.Add(1)
			Expect(list.RemoveByKey(9999)).To(BeFalse())
		})
	})

	Describe("Set", func() {
		It("replaces value with new key", func() {
			list := NewList[string]()
			list.Add("a", "b", "c")

			obs := &BasicObserver{}
			getter := list.Observe(obs)
			oldKeys := getter.Keys()

			list.Set(1, "B")

			newKeys := getter.Keys()
			Expect(getter.Values()).To(Equal([]string{"a", "B", "c"}))
			// Key at index 1 should be different (new key)
			Expect(newKeys[1]).NotTo(Equal(oldKeys[1]))
			// Other keys unchanged
			Expect(newKeys[0]).To(Equal(oldKeys[0]))
			Expect(newKeys[2]).To(Equal(oldKeys[2]))
		})

		It("fails with invalid index", func() {
			list := NewList[int]()
			list.Add(1)
			Expect(list.Set(-1, 0)).To(BeFalse())
			Expect(list.Set(5, 0)).To(BeFalse())
		})
	})

	Describe("Move", func() {
		It("moves item preserving key", func() {
			list := NewList[string]()
			list.Add("a", "b", "c")

			obs := &BasicObserver{}
			getter := list.Observe(obs)
			oldKeys := getter.Keys()
			keyA := oldKeys[0]

			list.Move(0, 2)

			Expect(getter.Values()).To(Equal([]string{"b", "c", "a"}))
			// Key should be preserved
			newKeys := getter.Keys()
			Expect(newKeys[2]).To(Equal(keyA))
		})

		It("move to same index is no-op", func() {
			list := NewList[int]()
			list.Add(1, 2, 3)
			Expect(list.Move(1, 1)).To(BeTrue())

			obs := &BasicObserver{}
			getter := list.Observe(obs)
			Expect(getter.Values()).To(Equal([]int{1, 2, 3}))
		})

		It("fails with invalid indices", func() {
			list := NewList[int]()
			list.Add(1, 2)
			Expect(list.Move(-1, 0)).To(BeFalse())
			Expect(list.Move(0, 5)).To(BeFalse())
		})
	})

	Describe("Swap", func() {
		It("swaps items preserving keys", func() {
			list := NewList[string]()
			list.Add("a", "b", "c")

			obs := &BasicObserver{}
			getter := list.Observe(obs)
			oldKeys := getter.Keys()

			list.Swap(0, 2)

			Expect(getter.Values()).To(Equal([]string{"c", "b", "a"}))
			// Keys should be swapped too
			newKeys := getter.Keys()
			Expect(newKeys[0]).To(Equal(oldKeys[2]))
			Expect(newKeys[2]).To(Equal(oldKeys[0]))
			Expect(newKeys[1]).To(Equal(oldKeys[1]))
		})

		It("swap same index is no-op", func() {
			list := NewList[int]()
			list.Add(1, 2)
			Expect(list.Swap(0, 0)).To(BeTrue())

			obs := &BasicObserver{}
			getter := list.Observe(obs)
			Expect(getter.Values()).To(Equal([]int{1, 2}))
		})

		It("fails with invalid indices", func() {
			list := NewList[int]()
			list.Add(1, 2)
			Expect(list.Swap(-1, 0)).To(BeFalse())
			Expect(list.Swap(0, 5)).To(BeFalse())
		})
	})

	Describe("Replace", func() {
		It("replaces all items with new keys", func() {
			list := NewList[string]()
			list.Add("a", "b")

			obs := &BasicObserver{}
			getter := list.Observe(obs)
			oldKeys := getter.Keys()

			list.Replace([]string{"x", "y", "z"})

			Expect(getter.Values()).To(Equal([]string{"x", "y", "z"}))
			newKeys := getter.Keys()
			// All keys should be new
			for _, oldKey := range oldKeys {
				for _, newKey := range newKeys {
					Expect(newKey).NotTo(Equal(oldKey))
				}
			}
		})

		It("can replace with empty", func() {
			list := NewList[int]()
			list.Add(1, 2, 3)
			list.Replace([]int{})
			Expect(list.PeekLen()).To(Equal(0))
		})
	})

	Describe("Clear", func() {
		It("removes all items", func() {
			list := NewList[int]()
			list.Add(1, 2, 3)
			list.Clear()
			Expect(list.PeekLen()).To(Equal(0))
		})

		It("no-op on empty list", func() {
			list := NewList[int]()
			list.Clear() // should not panic
			Expect(list.PeekLen()).To(Equal(0))
		})
	})

	Describe("ListGetter", func() {
		It("At returns key and value", func() {
			list := NewList[string]()
			list.Add("a", "b")

			obs := &BasicObserver{}
			getter := list.Observe(obs)

			key, val, ok := getter.At(0)
			Expect(ok).To(BeTrue())
			Expect(val).To(Equal("a"))
			Expect(key).To(BeNumerically(">", 0))
		})

		It("At returns false for invalid index", func() {
			list := NewList[int]()
			list.Add(1)

			obs := &BasicObserver{}
			getter := list.Observe(obs)

			_, _, ok := getter.At(-1)
			Expect(ok).To(BeFalse())
			_, _, ok = getter.At(5)
			Expect(ok).To(BeFalse())
		})

		It("IndexOf finds key", func() {
			list := NewList[string]()
			list.Add("a", "b", "c")

			obs := &BasicObserver{}
			getter := list.Observe(obs)
			keys := getter.Keys()

			Expect(getter.IndexOf(keys[1])).To(Equal(1))
			Expect(getter.IndexOf(9999)).To(Equal(-1))
		})

		It("ValueByKey returns value", func() {
			list := NewList[string]()
			list.Add("a", "b")

			obs := &BasicObserver{}
			getter := list.Observe(obs)
			keys := getter.Keys()

			val, ok := getter.ValueByKey(keys[0])
			Expect(ok).To(BeTrue())
			Expect(val).To(Equal("a"))

			_, ok = getter.ValueByKey(9999)
			Expect(ok).To(BeFalse())
		})

		It("All iterates key-value pairs", func() {
			list := NewList[string]()
			list.Add("a", "b", "c")

			obs := &BasicObserver{}
			getter := list.Observe(obs)

			var collected []string
			for _, v := range getter.All() {
				collected = append(collected, v)
			}
			Expect(collected).To(Equal([]string{"a", "b", "c"}))
		})
	})

	Describe("observer notifications", func() {
		It("Add notifies observers", func() {
			list := NewList[int]()
			obs := &BasicObserver{}
			list.Observe(obs)
			obs.GetAndResetUpdated()

			list.Add(1)
			Expect(obs.IsUpdated()).To(BeTrue())
		})

		It("Set notifies observers", func() {
			list := NewList[int]()
			list.Add(1)
			obs := &BasicObserver{}
			list.Observe(obs)
			obs.GetAndResetUpdated()

			list.Set(0, 2)
			Expect(obs.IsUpdated()).To(BeTrue())
		})

		It("RemoveAt notifies observers", func() {
			list := NewList[int]()
			list.Add(1)
			obs := &BasicObserver{}
			list.Observe(obs)
			obs.GetAndResetUpdated()

			list.RemoveAt(0)
			Expect(obs.IsUpdated()).To(BeTrue())
		})

		It("Move notifies observers", func() {
			list := NewList[int]()
			list.Add(1, 2)
			obs := &BasicObserver{}
			list.Observe(obs)
			obs.GetAndResetUpdated()

			list.Move(0, 1)
			Expect(obs.IsUpdated()).To(BeTrue())
		})

		It("Swap notifies observers", func() {
			list := NewList[int]()
			list.Add(1, 2)
			obs := &BasicObserver{}
			list.Observe(obs)
			obs.GetAndResetUpdated()

			list.Swap(0, 1)
			Expect(obs.IsUpdated()).To(BeTrue())
		})

		It("Replace notifies observers", func() {
			list := NewList[int]()
			list.Add(1)
			obs := &BasicObserver{}
			list.Observe(obs)
			obs.GetAndResetUpdated()

			list.Replace([]int{2, 3})
			Expect(obs.IsUpdated()).To(BeTrue())
		})

		It("Clear notifies observers", func() {
			list := NewList[int]()
			list.Add(1)
			obs := &BasicObserver{}
			list.Observe(obs)
			obs.GetAndResetUpdated()

			list.Clear()
			Expect(obs.IsUpdated()).To(BeTrue())
		})

		It("Clear on empty does not notify", func() {
			list := NewList[int]()
			obs := &BasicObserver{}
			list.Observe(obs)
			obs.GetAndResetUpdated()

			list.Clear()
			Expect(obs.IsUpdated()).To(BeFalse())
		})
	})

	Describe("subscribing reads", func() {
		It("Len subscribes the observer", func() {
			list := NewList[string]()
			list.Add("a", "b")

			obs := &BasicObserver{}
			n := list.Len(obs)
			Expect(n).To(Equal(2))
			obs.GetAndResetUpdated() // clear initial dirty state

			list.Add("c")
			Expect(obs.GetAndResetUpdated()).To(BeTrue())
		})

		It("At subscribes the observer", func() {
			list := NewList[string]()
			list.Add("x", "y")

			obs := &BasicObserver{}
			_, val, ok := list.At(obs, 1)
			Expect(ok).To(BeTrue())
			Expect(val).To(Equal("y"))

			list.Set(1, "z")
			Expect(obs.GetAndResetUpdated()).To(BeTrue())
		})

		It("At returns false for out-of-range index", func() {
			list := NewList[int]()
			obs := &BasicObserver{}
			_, _, ok := list.At(obs, 5)
			Expect(ok).To(BeFalse())
		})

		It("All subscribes the observer and iterates all items", func() {
			list := NewList[string]()
			list.Add("a", "b", "c")

			obs := &BasicObserver{}
			var vals []string
			for _, v := range list.All(obs) {
				vals = append(vals, v)
			}
			Expect(vals).To(Equal([]string{"a", "b", "c"}))

			list.Add("d")
			Expect(obs.GetAndResetUpdated()).To(BeTrue())
		})
	})

	Describe("registry integration", func() {
		It("subscribes through registry when available", func() {
			reg := NewRegistry()
			list := NewList[int]()
			list.Add(1, 2, 3)

			widget := &BasicObserver{}
			ctx := &mockRegistryProvider{registry: reg, observer: widget}

			getter := list.Observe(ctx)
			Expect(getter.Len()).To(Equal(3))
			Expect(hasSubscription(reg, widget, list)).To(BeTrue())
		})
	})
})
