package observable

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ComputedValue", func() {
	Describe("basic operations", func() {
		It("computes derived values", func() {
			v := Simple("test")
			computed := NewComputed(func(obs Observer) string {
				return v.Get(obs) + "_computed"
			})

			var obs BasicObserver
			getter := computed.Observe(&obs)

			Expect(getter.Get()).To(Equal("test_computed"))
		})

		It("recomputes when dependency changes", func() {
			v := Simple("test")
			computed := NewComputed(func(obs Observer) string {
				return v.Get(obs) + "_computed"
			})

			var obs BasicObserver
			obs.GetAndResetUpdated() // clear initial dirty state

			getter := computed.Observe(&obs)
			Expect(getter.Get()).To(Equal("test_computed"))

			v.Set("test2")
			Expect(obs.IsUpdated()).To(BeTrue())
			Expect(getter.Get()).To(Equal("test2_computed"))
		})

		It("caches value when clean", func() {
			callCount := 0
			v := Simple("test")
			computed := NewComputed(func(obs Observer) string {
				callCount++
				return v.Get(obs) + "_computed"
			})

			// NewComputed calls f once during static-folding probe detection.
			Expect(callCount).To(Equal(1))

			var obs BasicObserver
			getter := computed.Observe(&obs)

			// First real get computes (establishes subscriptions)
			Expect(getter.Get()).To(Equal("test_computed"))
			Expect(callCount).To(Equal(2))

			// Second call uses cache
			Expect(getter.Get()).To(Equal("test_computed"))
			Expect(callCount).To(Equal(2))

			// After change, recomputes
			v.Set("test2")
			Expect(getter.Get()).To(Equal("test2_computed"))
			Expect(callCount).To(Equal(3))
		})
	})

	Describe("chaining", func() {
		It("chains computed values", func() {
			v := Simple("test")
			computed1 := NewComputed(func(obs Observer) string {
				return v.Get(obs) + "_c1"
			})
			computed2 := NewComputed(func(obs Observer) string {
				return computed1.Get(obs) + "_c2"
			})

			var obs BasicObserver
			obs.GetAndResetUpdated()

			getter := computed2.Observe(&obs)
			Expect(getter.Get()).To(Equal("test_c1_c2"))

			v.Set("test2")
			Expect(obs.IsUpdated()).To(BeTrue())
			Expect(getter.Get()).To(Equal("test2_c1_c2"))
		})

		It("chains three levels deep", func() {
			v := Simple(1)
			c1 := NewComputed(func(obs Observer) int { return v.Get(obs) * 2 })
			c2 := NewComputed(func(obs Observer) int { return c1.Get(obs) + 10 })
			c3 := NewComputed(func(obs Observer) int { return c2.Get(obs) * 3 })

			var obs BasicObserver
			getter := c3.Observe(&obs)

			// (1 * 2 + 10) * 3 = 36
			Expect(getter.Get()).To(Equal(36))

			v.Set(5)
			// (5 * 2 + 10) * 3 = 60
			Expect(getter.Get()).To(Equal(60))
		})
	})

	Describe("multiple dependencies", func() {
		It("tracks multiple dependencies", func() {
			v1 := Simple("hello")
			v2 := Simple("world")
			computed := NewComputed(func(obs Observer) string {
				return v1.Get(obs) + " " + v2.Get(obs)
			})

			var obs BasicObserver
			obs.GetAndResetUpdated()

			getter := computed.Observe(&obs)
			Expect(getter.Get()).To(Equal("hello world"))

			v1.Set("hi")
			Expect(obs.IsUpdated()).To(BeTrue())
			obs.GetAndResetUpdated()
			Expect(getter.Get()).To(Equal("hi world"))

			v2.Set("there")
			Expect(obs.IsUpdated()).To(BeTrue())
			Expect(getter.Get()).To(Equal("hi there"))
		})
	})

	Describe("with registry", func() {
		It("uses registry for computed -> dependency subscriptions", func() {
			reg := NewRegistry()
			v := Simple("test")

			computed := NewComputed(func(obs Observer) string {
				return v.Get(obs) + "_computed"
			}).(*ComputedValue[string])

			// Create context with registry
			ctx := &mockRegistryProvider{
				registry: reg,
				observer: &BasicObserver{},
			}

			// Access computed through context
			result := computed.Get(ctx)
			Expect(result).To(Equal("test_computed"))

			// computed should be bound to registry
			Expect(computed.registry.Load()).To(Equal(reg))

			// v should also be bound to registry (lazy binding from computed)
			Expect(v.registry.Load()).To(Equal(reg))

			// Registry should have: ctx.observer -> computed -> v
			Expect(hasSubscription(reg, ctx.observer, computed)).To(BeTrue())
			Expect(hasSubscription(reg, computed, v)).To(BeTrue())
		})

		It("propagates changes through registry", func() {
			reg := NewRegistry()
			v := Simple("test")

			computed := NewComputed(func(obs Observer) string {
				return v.Get(obs) + "_computed"
			})

			widget := &BasicObserver{}
			ctx := &mockRegistryProvider{
				registry: reg,
				observer: widget,
			}

			getter := computed.Observe(ctx)
			Expect(getter.Get()).To(Equal("test_computed"))

			// Change source
			v.Set("changed")

			// Widget should be marked dirty
			Expect(widget.IsUpdated()).To(BeTrue())
			Expect(getter.Get()).To(Equal("changed_computed"))
		})
	})

	Describe("static folding", func() {
		It("returns staticValue when all deps are static", func() {
			s1 := Static(10)
			s2 := Static(20)
			result := NewComputed(func(obs Observer) int {
				return s1.Get(obs) + s2.Get(obs)
			})
			Expect(result.Get(Noop)).To(Equal(30))
			Expect(result.Peek()).To(Equal(30))
			_, isStatic := result.(staticValue[int])
			Expect(isStatic).To(BeTrue())
		})

		It("returns ComputedValue when any dep is dynamic", func() {
			s := Static(10)
			d := Simple(20)
			result := NewComputed(func(obs Observer) int {
				return s.Get(obs) + d.Get(obs)
			})
			Expect(result.Get(Noop)).To(Equal(30))
			_, isStatic := result.(staticValue[int])
			Expect(isStatic).To(BeFalse())
		})

		It("returns ComputedValue when all deps are dynamic", func() {
			d1 := Simple(10)
			d2 := Simple(20)
			result := NewComputed(func(obs Observer) int {
				return d1.Get(obs) + d2.Get(obs)
			})
			Expect(result.Get(Noop)).To(Equal(30))
			_, isStatic := result.(staticValue[int])
			Expect(isStatic).To(BeFalse())
		})

		It("returns staticValue for pure computation with no deps", func() {
			result := NewComputed(func(obs Observer) int {
				return 42
			})
			Expect(result.Get(Noop)).To(Equal(42))
			_, isStatic := result.(staticValue[int])
			Expect(isStatic).To(BeTrue())
		})
	})

	Describe("NewCachedComputed", func() {
		It("caches binding phase separately", func() {
			v := Simple("test")

			type cache struct {
				value ValueGetter[string]
			}

			computed := NewCachedComputed(
				func(obs Observer) *cache {
					return &cache{value: v.Observe(obs)}
				},
				func(c *cache) string {
					return c.value.Get() + "_cached"
				},
			)

			var obs BasicObserver
			getter := computed.Observe(&obs)

			Expect(getter.Get()).To(Equal("test_cached"))
		})

		It("works with multiple cached dependencies", func() {
			v1 := Simple("hello")
			v2 := Simple(42)

			type cache struct {
				str ValueGetter[string]
				num ValueGetter[int]
			}

			computed := NewCachedComputed(
				func(obs Observer) *cache {
					return &cache{
						str: v1.Observe(obs),
						num: v2.Observe(obs),
					}
				},
				func(c *cache) string {
					return fmt.Sprintf("%s_%d", c.str.Get(), c.num.Get())
				},
			)

			var obs BasicObserver
			obs.GetAndResetUpdated()

			getter := computed.Observe(&obs)
			Expect(getter.Get()).To(Equal("hello_42"))

			v1.Set("world")
			Expect(obs.IsUpdated()).To(BeTrue())
			obs.GetAndResetUpdated()
			Expect(getter.Get()).To(Equal("world_42"))

			v2.Set(99)
			Expect(obs.IsUpdated()).To(BeTrue())
			Expect(getter.Get()).To(Equal("world_99"))
		})

		It("returns staticValue when bind has only static deps", func() {
			s := Static("hello")

			type cache struct {
				value ValueGetter[string]
			}

			result := NewCachedComputed(
				func(obs Observer) *cache {
					return &cache{value: s.Observe(obs)}
				},
				func(c *cache) string {
					return c.value.Get() + "_cached"
				},
			)

			Expect(result.Get(Noop)).To(Equal("hello_cached"))
			_, isStatic := result.(staticValue[string])
			Expect(isStatic).To(BeTrue())
		})

		It("returns ComputedValue when bind has dynamic deps", func() {
			d := Simple("hello")

			type cache struct {
				value ValueGetter[string]
			}

			result := NewCachedComputed(
				func(obs Observer) *cache {
					return &cache{value: d.Observe(obs)}
				},
				func(c *cache) string {
					return c.value.Get() + "_cached"
				},
			)

			Expect(result.Get(Noop)).To(Equal("hello_cached"))
			_, isStatic := result.(staticValue[string])
			Expect(isStatic).To(BeFalse())
		})
	})
})
