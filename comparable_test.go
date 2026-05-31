package observable

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SimpleValue", func() {
	It("stores and retrieves values", func() {
		v := Simple("test")
		var obs BasicObserver

		getter := v.Observe(&obs)
		Expect(getter.Get()).To(Equal("test"))
	})

	It("notifies observers on Set when value changes", func() {
		v := Simple("test")
		var obs BasicObserver

		_ = v.Observe(&obs)
		obs.GetAndResetUpdated()

		v.Set("test2")
		Expect(obs.IsUpdated()).To(BeTrue())
	})

	It("skips notification on Set when value is unchanged", func() {
		v := Simple("test")
		var obs BasicObserver

		_ = v.Observe(&obs)
		obs.GetAndResetUpdated()

		v.Set("test")
		Expect(obs.IsUpdated()).To(BeFalse())
	})

	It("notifies observers on Update when value changes", func() {
		v := Simple(10)
		var obs BasicObserver

		_ = v.Observe(&obs)
		obs.GetAndResetUpdated()

		v.Update(func(val *int) {
			*val = 20
		})
		Expect(obs.IsUpdated()).To(BeTrue())
	})

	It("skips notification on Update when value is unchanged", func() {
		v := Simple(10)
		var obs BasicObserver

		_ = v.Observe(&obs)
		obs.GetAndResetUpdated()

		v.Update(func(val *int) {
			// no-op: value stays 10
		})
		Expect(obs.IsUpdated()).To(BeFalse())
	})

	It("MaybeUpdate skips notification when callback returns false", func() {
		v := Simple(10)
		var obs BasicObserver

		_ = v.Observe(&obs)
		obs.GetAndResetUpdated()

		v.MaybeUpdate(func(val *int) bool {
			return false
		})
		Expect(obs.IsUpdated()).To(BeFalse())
	})

	It("MaybeUpdate notifies when callback returns true", func() {
		v := Simple(10)
		var obs BasicObserver

		_ = v.Observe(&obs)
		obs.GetAndResetUpdated()

		v.MaybeUpdate(func(val *int) bool {
			*val = 20
			return true
		})
		Expect(obs.IsUpdated()).To(BeTrue())
	})

	It("Peek returns value without subscribing", func() {
		v := Simple(42)
		Expect(v.Peek()).To(Equal(42))
	})

	It("Get subscribes the observer", func() {
		v := Simple("hello")
		var obs BasicObserver

		val := v.Get(&obs)
		Expect(val).To(Equal("hello"))

		obs.GetAndResetUpdated()
		v.Set("world")
		Expect(obs.IsUpdated()).To(BeTrue())
	})

	It("satisfies Value interface", func() {
		var _ Value[int] = Simple(0)
	})
})
