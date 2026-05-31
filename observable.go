package observable

import (
	"github.com/gataengine/observable/internal/xset"
	"iter"
	"sync"
	"sync/atomic"
	"weak"
)

const (
	observableSetUpgradeThreshold = 3
)

// observableSet is a generic collection that auto-upgrades from slice to xset
// for efficient lookups once a threshold is reached.
// Zero value is ready to use.
type observableSet[T comparable] struct {
	mu    sync.RWMutex
	slice []T
	set   xset.Set[T]
}

// Add adds an item to the set (idempotent)
func (s *observableSet[T]) Add(item T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Lazy initialization for zero value
	if s.set == nil && s.slice == nil {
		s.slice = make([]T, 0, observableSetUpgradeThreshold+1)
	}

	if s.set != nil {
		s.set.Add(item)
		return
	}

	// Check if already exists in slice
	for _, existing := range s.slice {
		if existing == item {
			return
		}
	}

	s.slice = append(s.slice, item)

	// Upgrade to xset if threshold exceeded
	if len(s.slice) > observableSetUpgradeThreshold {
		s.upgradeToSet()
	}
}

// Remove removes an item from the set
func (s *observableSet[T]) Remove(item T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.set != nil {
		s.set.Remove(item)
		return
	}

	for i, existing := range s.slice {
		if existing == item {
			s.slice = append(s.slice[:i], s.slice[i+1:]...)
			return
		}
	}
}

// Contains checks if an item exists in the set
func (s *observableSet[T]) Contains(item T) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.set != nil {
		return s.set.Contains(item)
	}

	for _, existing := range s.slice {
		if existing == item {
			return true
		}
	}
	return false
}

// Len returns the number of items in the set
func (s *observableSet[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.set != nil {
		return s.set.Len()
	}
	return len(s.slice)
}

// Iter returns an iterator over all items in the set
func (s *observableSet[T]) Iter() iter.Seq[T] {
	return func(yield func(T) bool) {
		s.mu.RLock()
		defer s.mu.RUnlock()

		if s.set != nil {
			for item := range s.set.Keys() {
				if !yield(item) {
					return
				}
			}
		} else {
			for _, item := range s.slice {
				if !yield(item) {
					return
				}
			}
		}
	}
}

// upgradeToSet converts the slice to an xset (must be called with lock held)
func (s *observableSet[T]) upgradeToSet() {
	s.set = xset.NewSet[T]()
	for _, item := range s.slice {
		s.set.Add(item)
	}
	s.slice = nil
}

// observableBase provides core observable functionality with registry optimization.
// When registry is set, subscriptions go through registry (no local allocation).
// When registry is nil, falls back to local weak-pointer set.
// observableBase is safe for concurrent use.
type observableBase struct {
	registry  atomic.Pointer[Registry]
	observers atomic.Pointer[observableSet[weak.Pointer[observerState]]] // lazy, only for standalone observers
}

// RemoveObserver removes an observer from this observable.
func (v *observableBase) RemoveObserver(obs Observer) {
	v.removeObserver(obs)
}

func (v *observableBase) maybeAddObserver(self Observable, obs Observer) {
	// Skip subscription for no-op observers (used for peeking values)
	if obs.GetObserver() == nil {
		return
	}

	// Path 1: Registry-based subscription (preferred)
	// Check if observer provides registry context for lazy binding
	if provider, ok := obs.(RegistryProvider); ok {
		reg := provider.ObservableRegistry()
		if reg != nil {
			// Lazy bind: if no registry yet, adopt this one (atomic CAS)
			v.registry.CompareAndSwap(nil, reg)
			// Use registry path if it matches
			if v.registry.Load() == reg {
				reg.Subscribe(provider.CurrentObserver(), self)
				return
			}
			// Fall through to weak pointer path if registry mismatch
		}
	}

	// Path 2: Standalone observer - use local observableSet
	observers := v.ensureObservers()
	wref := weak.Make(obs.GetObserver())
	observers.Add(wref)
}

// ensureObservers returns the observers set, creating it if needed
func (v *observableBase) ensureObservers() *observableSet[weak.Pointer[observerState]] {
	observers := v.observers.Load()
	if observers != nil {
		return observers
	}
	// Create new set and try to swap it in
	newSet := &observableSet[weak.Pointer[observerState]]{}
	if v.observers.CompareAndSwap(nil, newSet) {
		return newSet
	}
	// Another goroutine won the race, use theirs
	return v.observers.Load()
}

func (v *observableBase) removeObserver(obs Observer) {
	// Remove from standalone set if present
	if observers := v.observers.Load(); observers != nil {
		wref := weak.Make(obs.GetObserver())
		observers.Remove(wref)
	}
	// Registry removal is handled by Registry.UnsubscribeAll()
}

func (v *observableBase) notifyChanged(self Observable) {
	// Path 1: Notify through registry (fast path)
	if reg := v.registry.Load(); reg != nil {
		reg.NotifyObservable(self)
	}

	// Path 2: Notify standalone observers (if any)
	observers := v.observers.Load()
	if observers == nil {
		return
	}

	var toDelete []weak.Pointer[observerState]

	for wref := range observers.Iter() {
		obs := wref.Value()
		if obs != nil {
			obs.MarkUpdated()
		} else {
			toDelete = append(toDelete, wref)
		}
	}

	// Clean up dead references
	if len(toDelete) > 0 {
		for _, wref := range toDelete {
			observers.Remove(wref)
		}
	}
}
