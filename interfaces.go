package observable

import "iter"

// Observer receives change notifications from values it has subscribed to.
// BasicObserver is the normal implementation. Passing an Observer to Get or
// Observe subscribes it to later updates unless the observer is Noop.
// Standalone observers use weak-pointer subscriptions by default.
type Observer interface {
	MarkUpdated()
	GetObserver() *observerState
}

// Observable is a source that can remove a previously subscribed observer.
type Observable interface {
	RemoveObserver(obs Observer)
}

// RegistryProvider lets an observer route subscriptions through a Registry.
// This is useful for observers with explicit lifecycles; observers that do not
// provide a registry use the standalone weak-pointer path. CurrentObserver
// returns the observer identity stored in the registry.
type RegistryProvider interface {
	ObservableRegistry() *Registry
	CurrentObserver() Observer
}

// DependentObservable is both an observable source and an observer of upstream
// sources. Registry uses this shape for recursive cleanup of computed and
// mapped values.
type DependentObservable interface {
	Observable
	Observer
}

// ROValue is a read-only observable value. Get subscribes the observer, Peek
// reads without subscribing, and Observe subscribes once and returns a getter.
type ROValue[T any] interface {
	Get(obs Observer) T
	Peek() T
	Observe(obs Observer) ValueGetter[T]
	RemoveObserver(obs Observer)
}

// Value is a mutable observable value.
type Value[T any] interface {
	ROValue[T]
	Set(T)
	Update(func(*T))
	MaybeUpdate(func(*T) bool)
}

// ValueGetter reads a subscribed value without re-subscribing.
type ValueGetter[T any] interface {
	Get() T
}

// ROList is a read-only observable list with stable item keys. Len, At, and All
// subscribe the observer; PeekLen, PeekAt, and PeekAll do not.
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

// ListGetter reads a subscribed list without re-subscribing.
type ListGetter[T any] interface {
	Len() int
	At(index int) (key int64, value T, ok bool)
	Keys() []int64
	Values() []T
	All() iter.Seq2[int64, T]
	IndexOf(key int64) int
	ValueByKey(key int64) (value T, ok bool)
}

// ROMap is a read-only observable map. Get and All subscribe the observer;
// Peek, PeekLen, and PeekAll do not.
type ROMap[K comparable, V any] interface {
	Observe(obs Observer) *MapGetter[K, V]
	Get(obs Observer, key K) (V, bool)
	All(obs Observer) iter.Seq2[K, V]
	Peek(key K) (V, bool)
	PeekLen() int
	PeekAll() iter.Seq2[K, V]
	RemoveObserver(obs Observer)
}
