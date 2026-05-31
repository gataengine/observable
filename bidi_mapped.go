package observable

// NewBidiMapped creates a two-way mapped value backed by source.
func NewBidiMapped[S, T any](source Value[S], forward func(S) T, reverse func(T) (S, bool)) *BidiMappedValue[S, T] {
	b := &BidiMappedValue[S, T]{
		source:  source,
		forward: forward,
		reverse: reverse,
		valid:   Simple(true),
	}
	b.OnChange = func() { b.onSourceChange() }
	return b
}

// BidiMappedValue is a bidirectional mapped observable with a write-through cache.
// It implements Value[T]; reads return the cached value, if any, or forward(source),
// and writes attempt to parse via reverse and propagate to source.
type BidiMappedValue[S, T any] struct {
	BasicObserver
	observableBase

	source  Value[S]
	forward func(S) T
	reverse func(T) (S, bool)
	valid   *SimpleValue[bool]

	cached    *T   // nil = no cache, use forward(source)
	selfWrite bool // true while propagating Set to source.Set to suppress echo
}

// Get returns the current value and subscribes obs.
func (b *BidiMappedValue[S, T]) Get(obs Observer) T {
	b.maybeAddObserver(b, obs)
	return b.get(obs)
}

func (b *BidiMappedValue[S, T]) get(obs Observer) T {
	if b.cached != nil {
		// Still subscribe to source so we get notified of external changes.
		b.source.Get(b)
		return *b.cached
	}
	return b.forward(b.source.Get(b))
}

// Set caches the value and attempts to reverse-parse it to the source.
// If reverse succeeds, source is updated. If it fails, source stays unchanged.
// Either way, the cached value is stored so Get() returns exactly what was Set().
func (b *BidiMappedValue[S, T]) Set(t T) {
	b.cached = &t
	if s, ok := b.reverse(t); ok {
		b.valid.Set(true)
		b.selfWrite = true
		b.source.Set(s)
		b.selfWrite = false
	} else {
		b.valid.Set(false)
	}
	b.notifyChanged(b)
}

// Update reads the current value, applies cb, then sets the result.
func (b *BidiMappedValue[S, T]) Update(cb func(*T)) {
	v := b.Peek()
	cb(&v)
	b.Set(v)
}

// MaybeUpdate allows conditional in-place modification.
// The callback returns true if the value was changed and observers should be notified.
func (b *BidiMappedValue[S, T]) MaybeUpdate(cb func(*T) bool) {
	v := b.Peek()
	if cb(&v) {
		b.Set(v)
	}
}

// ClearCache clears the local cache and resets Valid to true.
// After ClearCache, Get() returns forward(source.Get()).
func (b *BidiMappedValue[S, T]) ClearCache() {
	b.cached = nil
	b.valid.Set(true)
	b.notifyChanged(b)
}

// Valid returns an observable boolean reflecting whether the last Set()
// successfully parsed. True initially, true after ClearCache or external
// source change, false after a failed reverse parse.
func (b *BidiMappedValue[S, T]) Valid() ROValue[bool] {
	return b.valid
}

// onSourceChange is called when the source notifies us of a change.
// If the change came from our own Set(), we ignore it (cache is already correct).
// Otherwise, we clear the cache so Get() returns the new forward(source).
func (b *BidiMappedValue[S, T]) onSourceChange() {
	if b.selfWrite {
		return
	}
	b.cached = nil
	b.valid.Set(true)
	b.notifyChanged(b)
}

// Observe subscribes obs and returns a getter for repeated reads.
func (b *BidiMappedValue[S, T]) Observe(obs Observer) ValueGetter[T] {
	b.maybeAddObserver(b, obs)
	return &bidiMappedGetter[S, T]{b: b}
}

// RemoveObserver removes an observer from this BidiMappedValue.
func (b *BidiMappedValue[S, T]) RemoveObserver(obs Observer) {
	b.observableBase.RemoveObserver(obs)
}

// ObservableRegistry implements RegistryProvider.
func (b *BidiMappedValue[S, T]) ObservableRegistry() *Registry {
	return b.registry.Load()
}

// CurrentObserver implements RegistryProvider.
func (b *BidiMappedValue[S, T]) CurrentObserver() Observer {
	return b
}

// Peek returns the current value without subscribing an observer.
func (b *BidiMappedValue[S, T]) Peek() T {
	if b.cached != nil {
		return *b.cached
	}
	return b.forward(b.source.Peek())
}

type bidiMappedGetter[S, T any] struct {
	b *BidiMappedValue[S, T]
}

func (g *bidiMappedGetter[S, T]) Get() T {
	return g.b.Peek()
}
