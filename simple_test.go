package observable

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NonComparableValue", func() {
	Describe("basic operations", func() {
		It("stores and retrieves values", func() {
			v := SimpleNonComparable("test")
			var obs BasicObserver

			getter := v.Observe(&obs)
			Expect(getter.Get()).To(Equal("test"))
		})

		It("notifies observers on Set", func() {
			v := SimpleNonComparable("test")
			var obs BasicObserver

			getter := v.Observe(&obs)
			Expect(getter.Get()).To(Equal("test"))

			v.Set("test2")
			Expect(obs.IsUpdated()).To(BeTrue())
			Expect(obs.GetAndResetUpdated()).To(BeTrue())
			Expect(obs.IsUpdated()).To(BeFalse())
			Expect(getter.Get()).To(Equal("test2"))
		})

		It("supports Update for in-place modification", func() {
			v := SimpleNonComparable([]int{1, 2, 3})
			var obs BasicObserver

			getter := v.Observe(&obs)
			Expect(getter.Get()).To(Equal([]int{1, 2, 3}))

			v.Update(func(slice *[]int) {
				*slice = append(*slice, 4)
			})

			Expect(obs.IsUpdated()).To(BeTrue())
			Expect(getter.Get()).To(Equal([]int{1, 2, 3, 4}))
		})

		It("MaybeUpdate skips notification when callback returns false", func() {
			v := SimpleNonComparable(10)
			var obs BasicObserver

			_ = v.Observe(&obs)
			obs.GetAndResetUpdated()

			v.MaybeUpdate(func(val *int) bool {
				return false // no change
			})
			Expect(obs.IsUpdated()).To(BeFalse())
		})

		It("MaybeUpdate notifies when callback returns true", func() {
			v := SimpleNonComparable(10)
			var obs BasicObserver

			_ = v.Observe(&obs)
			obs.GetAndResetUpdated()

			v.MaybeUpdate(func(val *int) bool {
				*val = 20
				return true
			})
			Expect(obs.IsUpdated()).To(BeTrue())
		})
	})

	Describe("with registry", func() {
		It("uses registry path when RegistryProvider is available", func() {
			reg := NewRegistry()
			v := SimpleNonComparable("test")

			// Create a mock context that implements RegistryProvider
			ctx := &mockRegistryProvider{
				registry: reg,
				observer: &BasicObserver{},
			}

			// Get value - should lazy-bind to registry
			val := v.Get(ctx)
			Expect(val).To(Equal("test"))

			// Check that subscription went through registry
			Expect(hasSubscription(reg, ctx.observer, v)).To(BeTrue())
		})

		It("lazy binds to first registry", func() {
			reg := NewRegistry()
			v := SimpleNonComparable("test")

			Expect(v.registry.Load()).To(BeNil())

			ctx := &mockRegistryProvider{
				registry: reg,
				observer: &BasicObserver{},
			}

			v.Get(ctx)

			Expect(v.registry.Load()).To(Equal(reg))
		})

		It("falls back to weak pointers for different registry", func() {
			reg1 := NewRegistry()
			reg2 := NewRegistry()
			v := SimpleNonComparable("test")

			ctx1 := &mockRegistryProvider{registry: reg1, observer: &BasicObserver{}}
			ctx2 := &mockRegistryProvider{registry: reg2, observer: &BasicObserver{}}

			v.Get(ctx1) // binds to reg1
			v.Get(ctx2) // different registry, should use weak pointers

			// ctx1's observer should be in reg1
			Expect(hasSubscription(reg1, ctx1.observer, v)).To(BeTrue())

			// ctx2's observer should NOT be in reg2 (used weak pointers instead)
			Expect(isObserverRegistered(reg2, ctx2.observer)).To(BeFalse())

			// But weak pointer set should have ctx2
			Expect(v.observers).NotTo(BeNil())
		})
	})
})

// mockRegistryProvider implements RegistryProvider for testing
type mockRegistryProvider struct {
	BasicObserver
	registry *Registry
	observer Observer
}

func (m *mockRegistryProvider) ObservableRegistry() *Registry {
	return m.registry
}

func (m *mockRegistryProvider) CurrentObserver() Observer {
	return m.observer
}
