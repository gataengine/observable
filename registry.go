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

// Registry owns subscriptions for observers with explicit lifecycles.
// RegistryProvider uses this path automatically. A Registry can unsubscribe all
// subscriptions for an observer, drain dirty observers, and cascade cleanup
// through computed and mapped values.
// Registry is safe for concurrent use.
type Registry struct {
	mu             sync.RWMutex
	obsToObservers map[Observable]*hybridset.Set[Observer]
	observerToObs  map[Observer]*hybridset.Set[Observable]
	dirty          map[Observer]struct{}
}

// NewRegistry creates an empty subscription registry.
func NewRegistry() *Registry {
	return &Registry{
		obsToObservers: make(map[Observable]*hybridset.Set[Observer]),
		observerToObs:  make(map[Observer]*hybridset.Set[Observable]),
		dirty:          make(map[Observer]struct{}),
	}
}

// Subscribe records that obs depends on o.
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

// NotifyObservable marks every observer subscribed to o as updated.
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

// MarkDirty marks obs as updated inside the registry.
func (r *Registry) MarkDirty(obs Observer) {
	if obs == nil {
		return
	}
	r.mu.Lock()
	r.dirty[obs] = struct{}{}
	r.mu.Unlock()
}

// HasDirty reports whether updated observers are waiting to drain.
func (r *Registry) HasDirty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.dirty) > 0
}

// DrainDirty returns updated observers and clears the dirty set.
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

// Unsubscribe removes one observer/source subscription.
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

// UnsubscribeAll removes every subscription owned by obs and recursively
// removes orphaned DependentObservable upstream subscriptions.
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

// UnsubscribeObservable removes o and all observer subscriptions pointing to it.
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
