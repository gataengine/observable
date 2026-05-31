package observable

import (
	"iter"
	"sync"
)

var _ ROList[int] = (*List[int])(nil)

// ListItem is an item in an observable list. Key is stable across moves and
// swaps so callers can diff identity separately from position.
type ListItem[T any] struct {
	Key   int64
	Value T
}

// List is a mutable observable list with stable item keys. Add, Insert, Set,
// and Replace assign new keys; Move and Swap preserve keys. List is safe for
// concurrent use.
type List[T any] struct {
	observableBase
	mu      sync.RWMutex
	items   []ListItem[T]
	nextKey int64
}

// NewList creates a new observable list.
func NewList[T any]() *List[T] {
	return &List[T]{
		items:   make([]ListItem[T], 0),
		nextKey: 1,
	}
}

// Observe subscribes obs and returns a getter for repeated reads.
func (l *List[T]) Observe(obs Observer) ListGetter[T] {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.maybeAddObserver(l, obs)
	return &BaseListGetter[T]{list: l}
}

// PeekLen returns the number of items without subscribing any observer.
func (l *List[T]) PeekLen() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.items)
}

// PeekAt returns the key and value at the given index without subscribing any observer.
func (l *List[T]) PeekAt(index int) (key int64, value T, ok bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if index < 0 || index >= len(l.items) {
		return 0, value, false
	}
	item := l.items[index]
	return item.Key, item.Value, true
}

// PeekAll returns an iterator over all key-value pairs without subscribing any observer.
// Note: The iterator holds a read lock for the duration of iteration.
func (l *List[T]) PeekAll() iter.Seq2[int64, T] {
	return func(yield func(int64, T) bool) {
		l.mu.RLock()
		defer l.mu.RUnlock()
		for _, item := range l.items {
			if !yield(item.Key, item.Value) {
				return
			}
		}
	}
}

// Len returns the number of items and subscribes obs.
func (l *List[T]) Len(obs Observer) int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	l.maybeAddObserver(l, obs)
	return len(l.items)
}

// At returns the key and value at the given index and subscribes obs.
func (l *List[T]) At(obs Observer, index int) (key int64, value T, ok bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	l.maybeAddObserver(l, obs)
	if index < 0 || index >= len(l.items) {
		return 0, value, false
	}
	item := l.items[index]
	return item.Key, item.Value, true
}

// All returns an iterator over all key-value pairs and subscribes obs.
func (l *List[T]) All(obs Observer) iter.Seq2[int64, T] {
	l.mu.RLock()
	l.maybeAddObserver(l, obs)
	l.mu.RUnlock()
	return func(yield func(int64, T) bool) {
		l.mu.RLock()
		defer l.mu.RUnlock()
		for _, item := range l.items {
			if !yield(item.Key, item.Value) {
				return
			}
		}
	}
}

// =============================================================================
// Mutations - all assign new keys except Move/Swap which preserve keys
// =============================================================================

// Add appends one or more values to the end of the list.
func (l *List[T]) Add(values ...T) {
	if len(values) == 0 {
		return
	}
	l.mu.Lock()
	for _, v := range values {
		l.items = append(l.items, ListItem[T]{Key: l.nextKey, Value: v})
		l.nextKey++
	}
	l.mu.Unlock()
	l.notifyChanged(l)
}

// Insert adds one or more values at the specified index.
func (l *List[T]) Insert(index int, values ...T) bool {
	l.mu.Lock()
	if index < 0 || index > len(l.items) {
		l.mu.Unlock()
		return false
	}
	if len(values) == 0 {
		l.mu.Unlock()
		return true
	}
	newItems := make([]ListItem[T], len(values))
	for i, v := range values {
		newItems[i] = ListItem[T]{Key: l.nextKey, Value: v}
		l.nextKey++
	}
	l.items = append(l.items[:index], append(newItems, l.items[index:]...)...)
	l.mu.Unlock()
	l.notifyChanged(l)
	return true
}

// RemoveAt removes the item at the specified index.
func (l *List[T]) RemoveAt(index int) bool {
	l.mu.Lock()
	if index < 0 || index >= len(l.items) {
		l.mu.Unlock()
		return false
	}
	l.items = append(l.items[:index], l.items[index+1:]...)
	l.mu.Unlock()
	l.notifyChanged(l)
	return true
}

// RemoveByKey removes the item with the specified key.
func (l *List[T]) RemoveByKey(key int64) bool {
	l.mu.Lock()
	for i, item := range l.items {
		if item.Key == key {
			l.items = append(l.items[:i], l.items[i+1:]...)
			l.mu.Unlock()
			l.notifyChanged(l)
			return true
		}
	}
	l.mu.Unlock()
	return false
}

// Set replaces the value at the specified index with a new key.
func (l *List[T]) Set(index int, value T) bool {
	l.mu.Lock()
	if index < 0 || index >= len(l.items) {
		l.mu.Unlock()
		return false
	}
	l.items[index] = ListItem[T]{Key: l.nextKey, Value: value}
	l.nextKey++
	l.mu.Unlock()
	l.notifyChanged(l)
	return true
}

// Move moves an item from one index to another, preserving its key.
func (l *List[T]) Move(fromIndex, toIndex int) bool {
	l.mu.Lock()
	if fromIndex < 0 || fromIndex >= len(l.items) ||
		toIndex < 0 || toIndex >= len(l.items) {
		l.mu.Unlock()
		return false
	}
	if fromIndex == toIndex {
		l.mu.Unlock()
		return true
	}
	item := l.items[fromIndex]
	l.items = append(l.items[:fromIndex], l.items[fromIndex+1:]...)
	l.items = append(l.items[:toIndex], append([]ListItem[T]{item}, l.items[toIndex:]...)...)
	l.mu.Unlock()
	l.notifyChanged(l)
	return true
}

// Swap swaps items at two indices, preserving their keys.
func (l *List[T]) Swap(i, j int) bool {
	l.mu.Lock()
	if i < 0 || i >= len(l.items) || j < 0 || j >= len(l.items) {
		l.mu.Unlock()
		return false
	}
	if i == j {
		l.mu.Unlock()
		return true
	}
	l.items[i], l.items[j] = l.items[j], l.items[i]
	l.mu.Unlock()
	l.notifyChanged(l)
	return true
}

// Replace replaces all items with new values, assigning new keys.
func (l *List[T]) Replace(values []T) {
	l.mu.Lock()
	l.items = make([]ListItem[T], len(values))
	for i, v := range values {
		l.items[i] = ListItem[T]{Key: l.nextKey, Value: v}
		l.nextKey++
	}
	l.mu.Unlock()
	l.notifyChanged(l)
}

// Clear removes all items from the list.
func (l *List[T]) Clear() {
	l.mu.Lock()
	if len(l.items) == 0 {
		l.mu.Unlock()
		return
	}
	l.items = make([]ListItem[T], 0)
	l.mu.Unlock()
	l.notifyChanged(l)
}

// =============================================================================
// BaseListGetter - read-only access to list state
// =============================================================================

// BaseListGetter provides read-only access to list items.
type BaseListGetter[T any] struct {
	list *List[T]
}

// Len returns the number of items in the list.
func (g *BaseListGetter[T]) Len() int {
	g.list.mu.RLock()
	defer g.list.mu.RUnlock()
	return len(g.list.items)
}

// At returns the key and value at the specified index.
func (g *BaseListGetter[T]) At(index int) (key int64, value T, ok bool) {
	g.list.mu.RLock()
	defer g.list.mu.RUnlock()
	if index < 0 || index >= len(g.list.items) {
		return 0, value, false
	}
	item := g.list.items[index]
	return item.Key, item.Value, true
}

// Keys returns all keys in the list.
func (g *BaseListGetter[T]) Keys() []int64 {
	g.list.mu.RLock()
	defer g.list.mu.RUnlock()
	keys := make([]int64, len(g.list.items))
	for i, item := range g.list.items {
		keys[i] = item.Key
	}
	return keys
}

// Values returns all values in the list.
func (g *BaseListGetter[T]) Values() []T {
	g.list.mu.RLock()
	defer g.list.mu.RUnlock()
	values := make([]T, len(g.list.items))
	for i, item := range g.list.items {
		values[i] = item.Value
	}
	return values
}

// All returns an iterator over all key-value pairs in the list.
// Note: The iterator holds a read lock for the duration of iteration.
func (g *BaseListGetter[T]) All() iter.Seq2[int64, T] {
	return func(yield func(int64, T) bool) {
		g.list.mu.RLock()
		defer g.list.mu.RUnlock()
		for _, item := range g.list.items {
			if !yield(item.Key, item.Value) {
				return
			}
		}
	}
}

// IndexOf returns the index of the item with the given key, or -1 if not found.
func (g *BaseListGetter[T]) IndexOf(key int64) int {
	g.list.mu.RLock()
	defer g.list.mu.RUnlock()
	for i, item := range g.list.items {
		if item.Key == key {
			return i
		}
	}
	return -1
}

// ValueByKey returns the value for the given key.
func (g *BaseListGetter[T]) ValueByKey(key int64) (value T, ok bool) {
	g.list.mu.RLock()
	defer g.list.mu.RUnlock()
	for _, item := range g.list.items {
		if item.Key == key {
			return item.Value, true
		}
	}
	return value, false
}
