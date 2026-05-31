package observable

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestObservable(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "UI Observable Suite")
}

// Test helpers for Registry with hybridset.Set

// hasSubscription checks if an observer is subscribed to an observable
func hasSubscription(reg *Registry, obs Observer, o Observable) bool {
	observables := reg.observerToObs[obs]
	if observables == nil {
		return false
	}
	return observables.Contains(o)
}

// hasObserver checks if an observable has a specific observer
func hasObserver(reg *Registry, o Observable, obs Observer) bool {
	observers := reg.obsToObservers[o]
	if observers == nil {
		return false
	}
	return observers.Contains(obs)
}

// subscriptionCount returns the number of observables an observer is subscribed to
func subscriptionCount(reg *Registry, obs Observer) int {
	observables := reg.observerToObs[obs]
	if observables == nil {
		return 0
	}
	return observables.Size()
}

// observerCount returns the number of observers for an observable
func observerCount(reg *Registry, o Observable) int {
	observers := reg.obsToObservers[o]
	if observers == nil {
		return 0
	}
	return observers.Size()
}

// isObserverRegistered checks if an observer has any subscriptions
func isObserverRegistered(reg *Registry, obs Observer) bool {
	observables := reg.observerToObs[obs]
	return observables != nil && observables.Size() > 0
}

// isObservableRegistered checks if an observable has any observers
func isObservableRegistered(reg *Registry, o Observable) bool {
	observers := reg.obsToObservers[o]
	return observers != nil && observers.Size() > 0
}

// registryIsEmpty checks if the registry has no subscriptions
func registryIsEmpty(reg *Registry) bool {
	return len(reg.observerToObs) == 0 && len(reg.obsToObservers) == 0
}
