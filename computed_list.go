package observable

import "iter"

// ComputedListItem is what the user's compute function returns.
// K is used for diffing; V is the value exposed to consumers.
type ComputedListItem[K comparable, V any] struct {
	Key   K
	Value V
}

// ComputedList is a read-only observable list derived from other observables.
// The compute function returns items with user-defined keys; internally these
// are mapped to stable int64 keys via diffing, so consumers see the same
// ListGetter / ROList API as List.
type ComputedList[K comparable, V any] struct {
	BasicObserver
	observableBase

	f       func(obs Observer) []ComputedListItem[K, V]
	items   []ListItem[V]
	keyMap  map[K]int64
	nextKey int64
}

var _ ROList[int] = (*ComputedList[int, int])(nil)

// NewComputedList creates a read-only observable list derived from other
// observables.
// Duplicate keys in f's result cause a panic.
func NewComputedList[K comparable, V any](
	f func(obs Observer) []ComputedListItem[K, V],
) *ComputedList[K, V] {
	c := &ComputedList[K, V]{
		f:       f,
		keyMap:  make(map[K]int64),
		nextKey: 1,
	}
	c.OnChange = func() { c.notifyChanged(c) }
	c.recompute()
	return c
}

func (c *ComputedList[K, V]) recompute() {
	newItems := c.f(c)

	newKeyMap := make(map[K]int64, len(newItems))
	newListItems := make([]ListItem[V], len(newItems))

	for i, item := range newItems {
		if _, dup := newKeyMap[item.Key]; dup {
			panic("ComputedList: duplicate key in compute function result")
		}

		var key int64
		if existing, ok := c.keyMap[item.Key]; ok {
			key = existing
		} else {
			key = c.nextKey
			c.nextKey++
		}

		newKeyMap[item.Key] = key
		newListItems[i] = ListItem[V]{Key: key, Value: item.Value}
	}

	c.items = newListItems
	c.keyMap = newKeyMap
}

func (c *ComputedList[K, V]) get() {
	if c.GetAndResetUpdated() {
		c.recompute()
	}
}

// ObservableRegistry implements RegistryProvider.
func (c *ComputedList[K, V]) ObservableRegistry() *Registry {
	return c.registry.Load()
}

// CurrentObserver implements RegistryProvider.
func (c *ComputedList[K, V]) CurrentObserver() Observer {
	return c
}

// Observe subscribes obs and returns a getter for repeated reads.
func (c *ComputedList[K, V]) Observe(obs Observer) ListGetter[V] {
	c.maybeAddObserver(c, obs)
	return &ComputedListGetter[K, V]{list: c}
}

// PeekLen returns the number of items without subscribing an observer.
func (c *ComputedList[K, V]) PeekLen() int {
	c.get()
	return len(c.items)
}

// PeekAt returns the key and value at the given index without subscribing an observer.
func (c *ComputedList[K, V]) PeekAt(index int) (key int64, value V, ok bool) {
	c.get()
	if index < 0 || index >= len(c.items) {
		return 0, value, false
	}
	item := c.items[index]
	return item.Key, item.Value, true
}

// PeekAll returns an iterator over all key-value pairs without subscribing an observer.
func (c *ComputedList[K, V]) PeekAll() iter.Seq2[int64, V] {
	c.get()
	return func(yield func(int64, V) bool) {
		for _, item := range c.items {
			if !yield(item.Key, item.Value) {
				return
			}
		}
	}
}

// Len returns the number of items and subscribes obs.
func (c *ComputedList[K, V]) Len(obs Observer) int {
	c.maybeAddObserver(c, obs)
	c.get()
	return len(c.items)
}

// At returns the key and value at the given index and subscribes obs.
func (c *ComputedList[K, V]) At(obs Observer, index int) (key int64, value V, ok bool) {
	c.maybeAddObserver(c, obs)
	c.get()
	if index < 0 || index >= len(c.items) {
		return 0, value, false
	}
	item := c.items[index]
	return item.Key, item.Value, true
}

// All returns an iterator over all key-value pairs and subscribes obs.
func (c *ComputedList[K, V]) All(obs Observer) iter.Seq2[int64, V] {
	c.maybeAddObserver(c, obs)
	c.get()
	return func(yield func(int64, V) bool) {
		for _, item := range c.items {
			if !yield(item.Key, item.Value) {
				return
			}
		}
	}
}

// ComputedListGetter provides subscribed read access to a ComputedList.
type ComputedListGetter[K comparable, V any] struct {
	list *ComputedList[K, V]
}

func (g *ComputedListGetter[K, V]) Len() int {
	g.list.get()
	return len(g.list.items)
}

func (g *ComputedListGetter[K, V]) At(index int) (key int64, value V, ok bool) {
	g.list.get()
	if index < 0 || index >= len(g.list.items) {
		return 0, value, false
	}
	item := g.list.items[index]
	return item.Key, item.Value, true
}

func (g *ComputedListGetter[K, V]) Keys() []int64 {
	g.list.get()
	keys := make([]int64, len(g.list.items))
	for i, item := range g.list.items {
		keys[i] = item.Key
	}
	return keys
}

func (g *ComputedListGetter[K, V]) Values() []V {
	g.list.get()
	values := make([]V, len(g.list.items))
	for i, item := range g.list.items {
		values[i] = item.Value
	}
	return values
}

func (g *ComputedListGetter[K, V]) All() iter.Seq2[int64, V] {
	g.list.get()
	return func(yield func(int64, V) bool) {
		for _, item := range g.list.items {
			if !yield(item.Key, item.Value) {
				return
			}
		}
	}
}

func (g *ComputedListGetter[K, V]) IndexOf(key int64) int {
	g.list.get()
	for i, item := range g.list.items {
		if item.Key == key {
			return i
		}
	}
	return -1
}

func (g *ComputedListGetter[K, V]) ValueByKey(key int64) (value V, ok bool) {
	g.list.get()
	for _, item := range g.list.items {
		if item.Key == key {
			return item.Value, true
		}
	}
	return value, false
}
