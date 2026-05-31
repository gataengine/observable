package observable

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cleanup (regression)", func() {
	It("unsubscribe_simple does not hang", func() {
		done := make(chan bool)
		go func() {
			reg := NewRegistry()
			obs := &benchRegistryObserver{registry: reg}
			v := Simple(42)
			v.Get(obs)
			reg.UnsubscribeAll(obs)
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(1 * time.Second):
			Fail("UnsubscribeAll hung for simple case")
		}
	})

	It("unsubscribe_computed does not hang", func() {
		done := make(chan bool)
		go func() {
			reg := NewRegistry()
			obs := &benchRegistryObserver{registry: reg}

			source := Simple(1)
			c := NewComputed(func(o Observer) int { return source.Get(o) * 2 })
			c.Get(obs)

			reg.UnsubscribeAll(obs)
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(1 * time.Second):
			Fail("UnsubscribeAll hung for computed case")
		}
	})

	It("unsubscribe_chain does not hang", func() {
		done := make(chan bool)
		go func() {
			reg := NewRegistry()
			obs := &benchRegistryObserver{registry: reg}

			source := Simple(1)
			c1 := NewComputed(func(o Observer) int { return source.Get(o) * 2 })
			c2 := NewComputed(func(o Observer) int { return c1.Get(o) + 10 })
			c3 := NewComputed(func(o Observer) int { return c2.Get(o) * 3 })
			c3.Get(obs)

			reg.UnsubscribeAll(obs)
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(1 * time.Second):
			Fail("UnsubscribeAll hung for chain case")
		}
	})

	// Debug test to understand registry state
	It("debug: check registry state after subscribe", func() {
		reg := NewRegistry()
		obs := &benchRegistryObserver{registry: reg}
		v := Simple(42)

		// Before subscription
		Expect(len(reg.obsToObservers)).To(Equal(0))
		Expect(len(reg.observerToObs)).To(Equal(0))

		// Subscribe
		v.Get(obs)

		// After subscription
		Expect(len(reg.obsToObservers)).To(Equal(1))
		Expect(len(reg.observerToObs)).To(Equal(1))

		// Check what's registered
		GinkgoWriter.Printf("obsToObservers keys: %d\n", len(reg.obsToObservers))
		GinkgoWriter.Printf("observerToObs keys: %d\n", len(reg.observerToObs))

		for o := range reg.obsToObservers {
			GinkgoWriter.Printf("  observable type: %T\n", o)
			_, isDep := o.(DependentObservable)
			GinkgoWriter.Printf("  is DependentObservable: %v\n", isDep)
		}
	})
})
