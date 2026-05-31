package observable

// ComputedValue is an observable that derives its value from other observables.
// It caches the computed result and re-computes when dependencies change.
type ComputedValue[T any] struct {
	BasicObserver
	cachedValue T
	f           func(obs Observer) T
	observableBase
}

// probeObserver marks an observer as a static-detection probe.
// ComputedValue.Get skips computation for probe observers to avoid
// corrupting dirty state of intermediate computeds during static folding detection.
type probeObserver interface {
	isProbe()
}

// Get returns the computed value and subscribes the observer.
func (c *ComputedValue[T]) Get(obs Observer) T {
	c.maybeAddObserver(c, obs)
	if _, ok := obs.(probeObserver); ok {
		return c.cachedValue
	}
	return c.get()
}

// Peek returns the computed value without subscribing any observer.
// If the value is dirty, it will be recomputed.
func (c *ComputedValue[T]) Peek() T {
	return c.get()
}

func (c *ComputedValue[T]) get() T {
	if c.GetAndResetUpdated() {
		c.cachedValue = c.f(c)
	}
	return c.cachedValue
}

// Observe returns a ValueGetter for repeated access without re-subscribing.
func (c *ComputedValue[T]) Observe(obs Observer) ValueGetter[T] {
	c.maybeAddObserver(c, obs)
	return &computedGetter[T]{
		v: c,
	}
}

// ObservableRegistry implements RegistryProvider.
// Returns the registry this computed is bound to (may be nil).
func (c *ComputedValue[T]) ObservableRegistry() *Registry {
	return c.registry.Load()
}

// CurrentObserver implements RegistryProvider.
// Returns the computed itself as the observer for dependency tracking.
func (c *ComputedValue[T]) CurrentObserver() Observer {
	return c
}

type computedGetter[T any] struct {
	v *ComputedValue[T]
}

func (c *computedGetter[T]) Get() T {
	return c.v.get()
}

// newComputedValue creates a ComputedValue without static folding.
func newComputedValue[T any](f func(obs Observer) T) *ComputedValue[T] {
	c := &ComputedValue[T]{
		f: f,
	}
	c.OnChange = func() { c.notifyChanged(c) }
	return c
}

// NewComputed creates a new computed observable.
// If f depends only on static values, returns a staticValue instead.
func NewComputed[T any](f func(obs Observer) T) ROValue[T] {
	probe := &staticProbe{}
	value := f(probe)
	if !probe.subscribed {
		return staticValue[T]{value: value}
	}
	return newComputedValue(f)
}

// NewCachedComputed creates a computed with a separate binding phase.
// bind is called once to capture dependencies, calc is called on each recompute.
// If bind discovers only static dependencies, returns a staticValue.
func NewCachedComputed[T, C any](bind func(obs Observer) C, calc func(C) T) ROValue[T] {
	probe := &staticProbe{}
	binder := bind(probe)
	if !probe.subscribed {
		return staticValue[T]{value: calc(binder)}
	}
	c := &ComputedValue[T]{}
	c.OnChange = func() { c.notifyChanged(c) }
	realBinder := bind(c)
	c.f = func(obs Observer) T {
		return calc(realBinder)
	}
	return c
}
