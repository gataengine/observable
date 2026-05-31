package observable

// Mapped creates a read-only observable value derived from source.
func Mapped[T, U any](source ROValue[T], fn func(T) U) ROValue[U] {
	if _, ok := source.(staticValue[T]); ok {
		return staticValue[U]{value: fn(source.Peek())}
	}
	m := &MappedValue[T, U]{
		source: source,
		fn:     fn,
	}
	m.OnChange = func() { m.notifyChanged(m) }
	return m
}

// MappedValue is an observable that derives its value from a single source observable
// using a pure transform function. Unlike ComputedValue, it has a static dependency
// (no re-tracking needed) and subscribes lazily on first Get.
type MappedValue[T, U any] struct {
	BasicObserver
	source      ROValue[T]
	fn          func(T) U
	cached      U
	initialized bool
	observableBase
}

// Get returns the current value and subscribes obs.
func (m *MappedValue[T, U]) Get(obs Observer) U {
	m.maybeAddObserver(m, obs)
	return m.get()
}

// Peek returns the current value without subscribing an observer.
// If the value is dirty or uninitialized, it will be recomputed.
func (m *MappedValue[T, U]) Peek() U {
	return m.get()
}

func (m *MappedValue[T, U]) get() U {
	if !m.initialized || m.GetAndResetUpdated() {
		m.cached = m.fn(m.source.Get(m))
		m.initialized = true
		// Mark clean so next get() returns cached value unless notified.
		m.GetAndResetUpdated()
	}
	return m.cached
}

// Observe subscribes obs and returns a getter for repeated reads.
func (m *MappedValue[T, U]) Observe(obs Observer) ValueGetter[U] {
	m.maybeAddObserver(m, obs)
	return &mappedGetter[T, U]{m: m}
}

// RemoveObserver removes an observer from this MappedValue.
func (m *MappedValue[T, U]) RemoveObserver(obs Observer) {
	m.observableBase.RemoveObserver(obs)
}

// ObservableRegistry implements RegistryProvider.
func (m *MappedValue[T, U]) ObservableRegistry() *Registry {
	return m.registry.Load()
}

// CurrentObserver implements RegistryProvider.
func (m *MappedValue[T, U]) CurrentObserver() Observer {
	return m
}

type mappedGetter[T, U any] struct {
	m *MappedValue[T, U]
}

func (g *mappedGetter[T, U]) Get() U {
	return g.m.get()
}
