package observable

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ComputedList", func() {
	It("computes initial values from source observable", func() {
		source := NewList[string]()
		source.Add("a", "b", "c")

		cl := NewComputedList(func(obs Observer) []ComputedListItem[string, string] {
			getter := source.Observe(obs)
			var items []ComputedListItem[string, string]
			for _, v := range getter.All() {
				items = append(items, ComputedListItem[string, string]{Key: v, Value: v})
			}
			return items
		})

		Expect(cl.PeekLen()).To(Equal(3))

		var values []string
		for _, v := range cl.PeekAll() {
			values = append(values, v)
		}
		Expect(values).To(Equal([]string{"a", "b", "c"}))
	})

	It("recomputes when source observable changes", func() {
		source := Simple("hello")

		cl := NewComputedList(func(obs Observer) []ComputedListItem[int, string] {
			val := source.Get(obs)
			return []ComputedListItem[int, string]{
				{Key: 1, Value: val},
			}
		})

		_, v, _ := cl.PeekAt(0)
		Expect(v).To(Equal("hello"))

		source.Set("world")

		_, v, _ = cl.PeekAt(0)
		Expect(v).To(Equal("world"))
	})

	It("preserves int64 keys for items with same user key", func() {
		source := NewList[string]()
		source.Add("a", "b", "c")

		cl := NewComputedList(func(obs Observer) []ComputedListItem[string, string] {
			getter := source.Observe(obs)
			var items []ComputedListItem[string, string]
			for _, v := range getter.All() {
				items = append(items, ComputedListItem[string, string]{Key: v, Value: v})
			}
			return items
		})

		obs := &BasicObserver{}
		getter := cl.Observe(obs)
		initialKeys := getter.Keys()
		Expect(initialKeys).To(HaveLen(3))

		source.RemoveAt(1)

		newKeys := getter.Keys()
		Expect(newKeys).To(HaveLen(2))
		Expect(newKeys[0]).To(Equal(initialKeys[0]))
		Expect(newKeys[1]).To(Equal(initialKeys[2]))
	})

	It("assigns new int64 keys for new user keys", func() {
		source := NewList[string]()
		source.Add("a", "b")

		cl := NewComputedList(func(obs Observer) []ComputedListItem[string, string] {
			getter := source.Observe(obs)
			var items []ComputedListItem[string, string]
			for _, v := range getter.All() {
				items = append(items, ComputedListItem[string, string]{Key: v, Value: v})
			}
			return items
		})

		obs := &BasicObserver{}
		getter := cl.Observe(obs)
		initialKeys := getter.Keys()

		source.Add("c")

		newKeys := getter.Keys()
		Expect(newKeys).To(HaveLen(3))
		Expect(newKeys[0]).To(Equal(initialKeys[0]))
		Expect(newKeys[1]).To(Equal(initialKeys[1]))
		Expect(newKeys[2]).NotTo(Equal(initialKeys[0]))
		Expect(newKeys[2]).NotTo(Equal(initialKeys[1]))
	})

	It("panics on duplicate keys in compute result", func() {
		Expect(func() {
			NewComputedList(func(obs Observer) []ComputedListItem[string, string] {
				return []ComputedListItem[string, string]{
					{Key: "same", Value: "a"},
					{Key: "same", Value: "b"},
				}
			})
		}).To(Panic())
	})

	It("notifies observers when dependencies change", func() {
		source := Simple("hello")

		cl := NewComputedList(func(obs Observer) []ComputedListItem[int, string] {
			val := source.Get(obs)
			return []ComputedListItem[int, string]{
				{Key: 1, Value: val},
			}
		})

		obs := &BasicObserver{}
		_ = cl.Observe(obs)
		obs.GetAndResetUpdated()

		source.Set("world")

		Expect(obs.IsUpdated()).To(BeTrue())
	})

	It("handles empty compute result", func() {
		cl := NewComputedList(func(obs Observer) []ComputedListItem[string, string] {
			return nil
		})

		Expect(cl.PeekLen()).To(Equal(0))

		var values []string
		for _, v := range cl.PeekAll() {
			values = append(values, v)
		}
		Expect(values).To(BeEmpty())
	})

	It("Len subscribes the observer", func() {
		src := NewList[string]()
		src.Add("a", "b")

		cl := NewComputedList(func(obs Observer) []ComputedListItem[string, string] {
			var items []ComputedListItem[string, string]
			for _, v := range src.All(obs) {
				items = append(items, ComputedListItem[string, string]{Key: v, Value: v})
			}
			return items
		})

		obs := &BasicObserver{}
		n := cl.Len(obs)
		Expect(n).To(Equal(2))

		src.Add("c")
		Expect(obs.GetAndResetUpdated()).To(BeTrue())
	})

	It("At subscribes the observer", func() {
		src := NewList[string]()
		src.Add("x", "y")

		cl := NewComputedList(func(obs Observer) []ComputedListItem[string, string] {
			var items []ComputedListItem[string, string]
			for _, v := range src.All(obs) {
				items = append(items, ComputedListItem[string, string]{Key: v, Value: v})
			}
			return items
		})

		obs := &BasicObserver{}
		_, val, ok := cl.At(obs, 1)
		Expect(ok).To(BeTrue())
		Expect(val).To(Equal("y"))

		src.Add("z")
		Expect(obs.GetAndResetUpdated()).To(BeTrue())
	})

	It("All subscribes the observer", func() {
		src := NewList[string]()
		src.Add("a", "b")

		cl := NewComputedList(func(obs Observer) []ComputedListItem[string, string] {
			var items []ComputedListItem[string, string]
			for _, v := range src.All(obs) {
				items = append(items, ComputedListItem[string, string]{Key: v, Value: v})
			}
			return items
		})

		obs := &BasicObserver{}
		var vals []string
		for _, v := range cl.All(obs) {
			vals = append(vals, v)
		}
		Expect(vals).To(Equal([]string{"a", "b"}))

		src.Add("c")
		Expect(obs.GetAndResetUpdated()).To(BeTrue())
	})

	Describe("ComputedListGetter", func() {
		var (
			cl     *ComputedList[string, string]
			getter ListGetter[string]
		)

		BeforeEach(func() {
			cl = NewComputedList(func(obs Observer) []ComputedListItem[string, string] {
				return []ComputedListItem[string, string]{
					{Key: "k1", Value: "alpha"},
					{Key: "k2", Value: "beta"},
					{Key: "k3", Value: "gamma"},
				}
			})
			obs := &BasicObserver{}
			getter = cl.Observe(obs)
		})

		It("Len returns count", func() {
			Expect(getter.Len()).To(Equal(3))
		})

		It("At returns item by index", func() {
			key, val, ok := getter.At(1)
			Expect(ok).To(BeTrue())
			Expect(val).To(Equal("beta"))
			Expect(key).NotTo(BeZero())
		})

		It("At returns false for out of bounds", func() {
			_, _, ok := getter.At(5)
			Expect(ok).To(BeFalse())
			_, _, ok = getter.At(-1)
			Expect(ok).To(BeFalse())
		})

		It("Keys returns all int64 keys", func() {
			keys := getter.Keys()
			Expect(keys).To(HaveLen(3))
		})

		It("Values returns all values in order", func() {
			Expect(getter.Values()).To(Equal([]string{"alpha", "beta", "gamma"}))
		})

		It("All iterates key-value pairs", func() {
			var vals []string
			for _, v := range getter.All() {
				vals = append(vals, v)
			}
			Expect(vals).To(Equal([]string{"alpha", "beta", "gamma"}))
		})

		It("IndexOf finds item by int64 key", func() {
			keys := getter.Keys()
			Expect(getter.IndexOf(keys[1])).To(Equal(1))
			Expect(getter.IndexOf(99999)).To(Equal(-1))
		})

		It("ValueByKey finds value by int64 key", func() {
			keys := getter.Keys()
			val, ok := getter.ValueByKey(keys[2])
			Expect(ok).To(BeTrue())
			Expect(val).To(Equal("gamma"))

			_, ok = getter.ValueByKey(99999)
			Expect(ok).To(BeFalse())
		})
	})
})
