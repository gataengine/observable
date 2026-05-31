package observable

import (
	"iter"
	"sync"
)

// Map is an observable map with user-provided keys.
// Widgets diff keys to determine what changed (added/removed).
// Values are assumed to be observable for value-change notifications.
// Map is safe for concurrent use.
type Map[K comparable, V any] struct {
	observableBase
	mu    sync.RWMutex
	items map[K]V
}

// NewMap creates a new observable map.
func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		items: make(map[K]V),
	}
}

// Observe subscribes and returns a getter for reading map state.
func (m *Map[K, V]) Observe(obs Observer) *MapGetter[K, V] {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maybeAddObserver(m, obs)
	return &MapGetter[K, V]{m: m}
}

// Len returns the number of items in the map.
func (m *Map[K, V]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.items)
}

// PeekLen returns the number of items without subscribing any observer.
// Alias for Len, consistent with ROList.PeekLen naming.
func (m *Map[K, V]) PeekLen() int {
	return m.Len()
}

// Peek returns the value for the given key without subscribing any observer.
func (m *Map[K, V]) Peek(key K) (value V, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok = m.items[key]
	return
}

// PeekAll returns an iterator over all key-value pairs without subscribing any observer.
// Note: The iterator holds a read lock for the duration of iteration.
func (m *Map[K, V]) PeekAll() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		m.mu.RLock()
		defer m.mu.RUnlock()
		for k, v := range m.items {
			if !yield(k, v) {
				return
			}
		}
	}
}

// Get subscribes the observer and returns the value for the given key.
func (m *Map[K, V]) Get(obs Observer, key K) (value V, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.maybeAddObserver(m, obs)
	value, ok = m.items[key]
	return
}

// All subscribes the observer and returns an iterator over all key-value pairs.
func (m *Map[K, V]) All(obs Observer) iter.Seq2[K, V] {
	m.mu.RLock()
	m.maybeAddObserver(m, obs)
	m.mu.RUnlock()
	return func(yield func(K, V) bool) {
		m.mu.RLock()
		defer m.mu.RUnlock()
		for k, v := range m.items {
			if !yield(k, v) {
				return
			}
		}
	}
}

// =============================================================================
// Mutations
// =============================================================================

// Set adds or replaces a value for the given key.
func (m *Map[K, V]) Set(key K, value V) {
	m.mu.Lock()
	m.items[key] = value
	m.mu.Unlock()
	m.notifyChanged(m)
}

// Delete removes the item with the given key.
// Returns true if the key existed.
func (m *Map[K, V]) Delete(key K) bool {
	m.mu.Lock()
	if _, exists := m.items[key]; !exists {
		m.mu.Unlock()
		return false
	}
	delete(m.items, key)
	m.mu.Unlock()
	m.notifyChanged(m)
	return true
}

// Replace replaces all items with the provided map.
func (m *Map[K, V]) Replace(items map[K]V) {
	m.mu.Lock()
	m.items = make(map[K]V, len(items))
	for k, v := range items {
		m.items[k] = v
	}
	m.mu.Unlock()
	m.notifyChanged(m)
}

// Merge adds or updates multiple items without removing existing ones.
func (m *Map[K, V]) Merge(items map[K]V) {
	if len(items) == 0 {
		return
	}
	m.mu.Lock()
	for k, v := range items {
		m.items[k] = v
	}
	m.mu.Unlock()
	m.notifyChanged(m)
}

// Clear removes all items from the map.
func (m *Map[K, V]) Clear() {
	m.mu.Lock()
	if len(m.items) == 0 {
		m.mu.Unlock()
		return
	}
	m.items = make(map[K]V)
	m.mu.Unlock()
	m.notifyChanged(m)
}

// =============================================================================
// MapGetter - read-only access to map state
// =============================================================================

// MapGetter provides read-only access to map items.
type MapGetter[K comparable, V any] struct {
	m *Map[K, V]
}

// Len returns the number of items in the map.
func (g *MapGetter[K, V]) Len() int {
	g.m.mu.RLock()
	defer g.m.mu.RUnlock()
	return len(g.m.items)
}

// Get returns the value for the given key.
func (g *MapGetter[K, V]) Get(key K) (value V, ok bool) {
	g.m.mu.RLock()
	defer g.m.mu.RUnlock()
	value, ok = g.m.items[key]
	return
}

// Has returns true if the key exists in the map.
func (g *MapGetter[K, V]) Has(key K) bool {
	g.m.mu.RLock()
	defer g.m.mu.RUnlock()
	_, ok := g.m.items[key]
	return ok
}

// Keys returns all keys in the map.
// Note: order is not guaranteed.
func (g *MapGetter[K, V]) Keys() []K {
	g.m.mu.RLock()
	defer g.m.mu.RUnlock()
	keys := make([]K, 0, len(g.m.items))
	for k := range g.m.items {
		keys = append(keys, k)
	}
	return keys
}

// Values returns all values in the map.
// Note: order is not guaranteed.
func (g *MapGetter[K, V]) Values() []V {
	g.m.mu.RLock()
	defer g.m.mu.RUnlock()
	values := make([]V, 0, len(g.m.items))
	for _, v := range g.m.items {
		values = append(values, v)
	}
	return values
}

// All returns an iterator over all key-value pairs in the map.
// Note: The iterator holds a read lock for the duration of iteration.
func (g *MapGetter[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		g.m.mu.RLock()
		defer g.m.mu.RUnlock()
		for k, v := range g.m.items {
			if !yield(k, v) {
				return
			}
		}
	}
}
