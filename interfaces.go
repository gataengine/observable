package observable

import "iter"

// Observer is notified when an observable changes.
type Observer interface {
	MarkUpdated()
	GetObserver() *observerState
}

// Observable represents a value that can be observed.
type Observable interface {
	RemoveObserver(obs Observer)
}

// RegistryProvider allows observers to provide registry context for lazy binding.
// When an observer implements this, observables can auto-bind to the registry.
type RegistryProvider interface {
	ObservableRegistry() *Registry
	CurrentObserver() Observer
}

// DependentObservable is an observable that also observes other observables.
// Used for cascading cleanup in UnsubscribeAll - when a DependentObservable
// has no remaining observers, its own subscriptions are also cleaned up.
type DependentObservable interface {
	Observable
	Observer
}

// ROValue is a read-only observable value.
type ROValue[T any] interface {
	Get(obs Observer) T
	Peek() T
	Observe(obs Observer) ValueGetter[T]
	RemoveObserver(obs Observer)
}

// Value is a read-write observable value.
type Value[T any] interface {
	ROValue[T]
	Set(T)
	Update(func(*T))
	MaybeUpdate(func(*T) bool)
}

// ValueGetter provides cached access to a value without re-subscribing.
type ValueGetter[T any] interface {
	Get() T
}

// ROList is a read-only observable list with stable int64 keys.
type ROList[T any] interface {
	Observe(obs Observer) ListGetter[T]
	Len(obs Observer) int
	At(obs Observer, index int) (key int64, value T, ok bool)
	All(obs Observer) iter.Seq2[int64, T]
	PeekLen() int
	PeekAt(index int) (key int64, value T, ok bool)
	PeekAll() iter.Seq2[int64, T]
	RemoveObserver(obs Observer)
}

// ListGetter provides subscribed read-only access to list items.
type ListGetter[T any] interface {
	Len() int
	At(index int) (key int64, value T, ok bool)
	Keys() []int64
	Values() []T
	All() iter.Seq2[int64, T]
	IndexOf(key int64) int
	ValueByKey(key int64) (value T, ok bool)
}

// ROMap is a read-only observable map.
type ROMap[K comparable, V any] interface {
	Observe(obs Observer) *MapGetter[K, V]
	Get(obs Observer, key K) (V, bool)
	All(obs Observer) iter.Seq2[K, V]
	Peek(key K) (V, bool)
	PeekLen() int
	PeekAll() iter.Seq2[K, V]
	RemoveObserver(obs Observer)
}
