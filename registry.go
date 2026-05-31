package observable

import (
	"github.com/gataengine/observable/hybridset"
	"sync"
)

// Pools for zero-allocation iteration
var (
	observerPool   = hybridset.NewPool[Observer](32)
	observablePool = hybridset.NewPool[Observable](32)
)

// Registry tracks observable subscriptions and dirty observers.
// It provides an optimized subscription path that avoids per-observable
// allocation overhead when observers and observables share a registry.
// Registry is safe for concurrent use.
type Registry struct {
	mu             sync.RWMutex
	obsToObservers map[Observable]*hybridset.Set[Observer]
	observerToObs  map[Observer]*hybridset.Set[Observable]
	dirty          map[Observer]struct{}
}

// NewRegistry creates a new subscription registry.
func NewRegistry() *Registry {
	return &Registry{
		obsToObservers: make(map[Observable]*hybridset.Set[Observer]),
		observerToObs:  make(map[Observer]*hybridset.Set[Observable]),
		dirty:          make(map[Observer]struct{}),
	}
}

// Subscribe registers an observer to receive notifications from an observable.
// Fast path: if already subscribed, returns immediately.
func (r *Registry) Subscribe(obs Observer, o Observable) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Fast path: check if already subscribed
	if observables := r.observerToObs[obs]; observables != nil {
		if observables.Contains(o) {
			return // already subscribed
		}
	}

	// observable -> observers mapping
	observers := r.obsToObservers[o]
	if observers == nil {
		observers = hybridset.New[Observer]()
		r.obsToObservers[o] = observers
	}
	observers.Add(obs)

	// observer -> observables mapping
	observables := r.observerToObs[obs]
	if observables == nil {
		observables = hybridset.New[Observable]()
		r.observerToObs[obs] = observables
	}
	observables.Add(o)
}

// NotifyObservable notifies all observers of an observable that it changed.
// Each observer's MarkUpdated() handles its own dirty tracking via OnChange callbacks.
func (r *Registry) NotifyObservable(o Observable) {
	r.mu.RLock()
	observers := r.obsToObservers[o]
	r.mu.RUnlock()

	if observers == nil {
		return
	}

	size := observers.Size()
	if size == 0 {
		return
	}

	// Try Range first (handles xsync mode with zero alloc)
	useCopyTo := observers.Range(func(obs Observer) bool {
		obs.MarkUpdated()
		return true
	})

	// Small mode: use pooled slice for zero-alloc iteration
	if useCopyTo {
		ptr := observerPool.Get(size)
		n := observers.CopyTo(*ptr)
		for i := 0; i < n; i++ {
			(*ptr)[i].MarkUpdated()
		}
		observerPool.Put(ptr)
	}
}

// MarkDirty marks an observer as needing update.
func (r *Registry) MarkDirty(obs Observer) {
	if obs == nil {
		return
	}
	r.mu.Lock()
	r.dirty[obs] = struct{}{}
	r.mu.Unlock()
}

// HasDirty reports whether any observers are currently marked dirty.
func (r *Registry) HasDirty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.dirty) > 0
}

// DrainDirty returns and clears all dirty observers.
func (r *Registry) DrainDirty() []Observer {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.dirty) == 0 {
		return nil
	}
	out := make([]Observer, 0, len(r.dirty))
	for obs := range r.dirty {
		out = append(out, obs)
	}
	clear(r.dirty)
	return out
}

// Unsubscribe removes an observer's subscription to an observable.
func (r *Registry) Unsubscribe(obs Observer, o Observable) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if observers := r.obsToObservers[o]; observers != nil {
		observers.Delete(obs)
		if observers.Size() == 0 {
			delete(r.obsToObservers, o)
		}
	}
	if observables := r.observerToObs[obs]; observables != nil {
		observables.Delete(o)
		if observables.Size() == 0 {
			delete(r.observerToObs, obs)
		}
	}
}

// UnsubscribeAll removes all subscriptions for an observer.
// If any observable the observer was subscribed to is a DependentObservable
// (e.g., ComputedValue) and now has zero observers, it is also cleaned up recursively.
func (r *Registry) UnsubscribeAll(obs Observer) {
	r.mu.Lock()

	observables := r.observerToObs[obs]
	if observables == nil || observables.Size() == 0 {
		delete(r.dirty, obs)
		r.mu.Unlock()
		return
	}

	// Collect DependentObservables that need cascading cleanup
	var toCleanup []Observer

	// Process each observable this observer was subscribed to
	processObservable := func(o Observable) {
		if observers := r.obsToObservers[o]; observers != nil {
			observers.Delete(obs)
			if observers.Size() == 0 {
				delete(r.obsToObservers, o)
				// Check for cascade: observable has no observers and is itself an observer
				if dep, ok := o.(DependentObservable); ok {
					toCleanup = append(toCleanup, dep)
				}
			}
		}
	}

	size := observables.Size()
	useCopyTo := observables.Range(func(o Observable) bool {
		processObservable(o)
		return true
	})

	if useCopyTo {
		ptr := observablePool.Get(size)
		n := observables.CopyTo(*ptr)
		for i := 0; i < n; i++ {
			processObservable((*ptr)[i])
		}
		observablePool.Put(ptr)
	}

	delete(r.observerToObs, obs)
	delete(r.dirty, obs)

	r.mu.Unlock()

	// Cascade cleanup for orphaned DependentObservables (outside lock to avoid deadlock)
	for _, dep := range toCleanup {
		r.UnsubscribeAll(dep)
	}
}

// UnsubscribeObservable removes all subscriptions to an observable.
func (r *Registry) UnsubscribeObservable(o Observable) {
	r.mu.Lock()
	defer r.mu.Unlock()

	observers := r.obsToObservers[o]
	if observers == nil || observers.Size() == 0 {
		return
	}

	// Process each observer subscribed to this observable
	processObserver := func(obs Observer) {
		if observables := r.observerToObs[obs]; observables != nil {
			observables.Delete(o)
			if observables.Size() == 0 {
				delete(r.observerToObs, obs)
			}
		}
	}

	size := observers.Size()
	useCopyTo := observers.Range(func(obs Observer) bool {
		processObserver(obs)
		return true
	})

	if useCopyTo {
		ptr := observerPool.Get(size)
		n := observers.CopyTo(*ptr)
		for i := 0; i < n; i++ {
			processObserver((*ptr)[i])
		}
		observerPool.Put(ptr)
	}

	delete(r.obsToObservers, o)
}
