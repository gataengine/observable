package observable

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Registry", func() {
	Describe("Subscribe", func() {
		It("tracks observer -> observable mappings", func() {
			reg := NewRegistry()
			obs := &BasicObserver{}
			v := Simple("test")

			reg.Subscribe(obs, v)

			Expect(hasSubscription(reg, obs, v)).To(BeTrue())
			Expect(hasObserver(reg, v, obs)).To(BeTrue())
		})

		It("is idempotent (fast path for already subscribed)", func() {
			reg := NewRegistry()
			obs := &BasicObserver{}
			v := Simple("test")

			reg.Subscribe(obs, v)
			reg.Subscribe(obs, v) // should be fast no-op
			reg.Subscribe(obs, v)

			Expect(subscriptionCount(reg, obs)).To(Equal(1))
			Expect(observerCount(reg, v)).To(Equal(1))
		})

		It("allows multiple observers per observable", func() {
			reg := NewRegistry()
			obs1 := &BasicObserver{}
			obs2 := &BasicObserver{}
			v := Simple("test")

			reg.Subscribe(obs1, v)
			reg.Subscribe(obs2, v)

			Expect(observerCount(reg, v)).To(Equal(2))
		})

		It("allows multiple observables per observer", func() {
			reg := NewRegistry()
			obs := &BasicObserver{}
			v1 := Simple("test1")
			v2 := Simple("test2")

			reg.Subscribe(obs, v1)
			reg.Subscribe(obs, v2)

			Expect(subscriptionCount(reg, obs)).To(Equal(2))
		})
	})

	Describe("NotifyObservable", func() {
		It("calls MarkUpdated on all observers", func() {
			reg := NewRegistry()
			obs1 := &BasicObserver{}
			obs2 := &BasicObserver{}
			v := Simple("test")

			reg.Subscribe(obs1, v)
			reg.Subscribe(obs2, v)

			// Clear initial state
			obs1.GetAndResetUpdated()
			obs2.GetAndResetUpdated()

			reg.NotifyObservable(v)

			// NotifyObservable calls MarkUpdated on each observer
			// Dirty tracking is delegated to observers via their OnChange callbacks
			Expect(obs1.IsUpdated()).To(BeTrue())
			Expect(obs2.IsUpdated()).To(BeTrue())
		})
	})

	Describe("UnsubscribeAll", func() {
		It("removes all subscriptions for an observer", func() {
			reg := NewRegistry()
			obs := &BasicObserver{}
			v1 := Simple("test1")
			v2 := Simple("test2")

			reg.Subscribe(obs, v1)
			reg.Subscribe(obs, v2)

			reg.UnsubscribeAll(obs)

			Expect(isObserverRegistered(reg, obs)).To(BeFalse())
			Expect(isObservableRegistered(reg, v1)).To(BeFalse())
			Expect(isObservableRegistered(reg, v2)).To(BeFalse())
		})

		It("removes observer from dirty set", func() {
			reg := NewRegistry()
			obs := &BasicObserver{}
			v := Simple("test")

			reg.Subscribe(obs, v)
			reg.MarkDirty(obs)

			Expect(reg.dirty).To(HaveKey(Observer(obs)))

			reg.UnsubscribeAll(obs)

			Expect(reg.dirty).NotTo(HaveKey(Observer(obs)))
		})

		Describe("cascading cleanup", func() {
			It("cleans up orphaned computed when widget unsubscribes", func() {
				reg := NewRegistry()
				source := Simple("test")
				computed := NewComputed(func(obs Observer) string {
					return source.Get(obs) + "_computed"
				}).(*ComputedValue[string])

				// Simulate widget subscribing through context
				widget := &BasicObserver{}
				ctx := &mockRegistryProvider{registry: reg, observer: widget}

				// Access computed - establishes: widget -> computed -> source
				computed.Get(ctx)

				// Verify chain exists
				Expect(hasSubscription(reg, widget, computed)).To(BeTrue())
				Expect(hasSubscription(reg, computed, source)).To(BeTrue())

				// Unsubscribe widget - should cascade to computed
				reg.UnsubscribeAll(widget)

				// Widget should be gone
				Expect(isObserverRegistered(reg, widget)).To(BeFalse())

				// Computed should also be cleaned up (no more observers)
				Expect(isObserverRegistered(reg, computed)).To(BeFalse())
				Expect(isObservableRegistered(reg, computed)).To(BeFalse())

				// Source should have no observers
				Expect(isObservableRegistered(reg, source)).To(BeFalse())
			})

			It("cleans up chain of computeds", func() {
				reg := NewRegistry()
				source := Simple(1)
				c1 := NewComputed(func(obs Observer) int { return source.Get(obs) * 2 }).(*ComputedValue[int])
				c2 := NewComputed(func(obs Observer) int { return c1.Get(obs) + 10 }).(*ComputedValue[int])
				c3 := NewComputed(func(obs Observer) int { return c2.Get(obs) * 3 }).(*ComputedValue[int])

				// Widget observes c3
				widget := &BasicObserver{}
				ctx := &mockRegistryProvider{registry: reg, observer: widget}

				c3.Get(ctx)

				// Verify chain: widget -> c3 -> c2 -> c1 -> source
				Expect(hasSubscription(reg, widget, c3)).To(BeTrue())
				Expect(hasSubscription(reg, c3, c2)).To(BeTrue())
				Expect(hasSubscription(reg, c2, c1)).To(BeTrue())
				Expect(hasSubscription(reg, c1, source)).To(BeTrue())

				// Unsubscribe widget
				reg.UnsubscribeAll(widget)

				// All should be cleaned up
				Expect(registryIsEmpty(reg)).To(BeTrue())
			})

			It("preserves computed with multiple observers", func() {
				reg := NewRegistry()
				source := Simple("test")
				computed := NewComputed(func(obs Observer) string {
					return source.Get(obs) + "_computed"
				}).(*ComputedValue[string])

				// Two widgets observe the same computed
				widget1 := &BasicObserver{}
				widget2 := &BasicObserver{}
				ctx1 := &mockRegistryProvider{registry: reg, observer: widget1}
				ctx2 := &mockRegistryProvider{registry: reg, observer: widget2}

				computed.Get(ctx1)
				computed.Get(ctx2)

				// Verify both subscribed
				Expect(observerCount(reg, computed)).To(Equal(2))

				// Unsubscribe widget1
				reg.UnsubscribeAll(widget1)

				// Computed should still exist (widget2 still observing)
				Expect(observerCount(reg, computed)).To(Equal(1))
				Expect(hasSubscription(reg, computed, source)).To(BeTrue())

				// Unsubscribe widget2 - now cascade
				reg.UnsubscribeAll(widget2)

				// Now everything should be cleaned
				Expect(registryIsEmpty(reg)).To(BeTrue())
			})

			It("handles diamond dependency pattern", func() {
				reg := NewRegistry()
				source := Simple(10)

				// Diamond: source <- c1, c2 <- combined
				c1 := NewComputed(func(obs Observer) int { return source.Get(obs) * 2 }).(*ComputedValue[int])
				c2 := NewComputed(func(obs Observer) int { return source.Get(obs) + 5 }).(*ComputedValue[int])
				combined := NewComputed(func(obs Observer) int {
					return c1.Get(obs) + c2.Get(obs)
				}).(*ComputedValue[int])

				widget := &BasicObserver{}
				ctx := &mockRegistryProvider{registry: reg, observer: widget}

				result := combined.Get(ctx)
				// source=10: c1=20, c2=15, combined=35
				Expect(result).To(Equal(35))

				// Verify structure
				Expect(subscriptionCount(reg, combined)).To(Equal(2)) // c1 and c2
				Expect(observerCount(reg, source)).To(Equal(2))       // c1 and c2

				// Unsubscribe widget
				reg.UnsubscribeAll(widget)

				// All should be cleaned
				Expect(registryIsEmpty(reg)).To(BeTrue())
			})
		})
	})

	Describe("DrainDirty", func() {
		It("returns and clears dirty observers", func() {
			reg := NewRegistry()
			obs1 := &BasicObserver{}
			obs2 := &BasicObserver{}

			reg.MarkDirty(obs1)
			reg.MarkDirty(obs2)

			dirty := reg.DrainDirty()

			Expect(dirty).To(HaveLen(2))
			Expect(dirty).To(ContainElement(Observer(obs1)))
			Expect(dirty).To(ContainElement(Observer(obs2)))

			// Should be empty now
			Expect(reg.dirty).To(BeEmpty())
			Expect(reg.DrainDirty()).To(BeNil())
		})
	})
})
