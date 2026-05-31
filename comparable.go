package observable

// Simple creates an observable value for comparable values.
func Simple[T comparable](initial T) *SimpleValue[T] {
	return &SimpleValue[T]{
		value: initial,
	}
}

// SimpleValue is an observable that skips notifications when the value
// is unchanged. Requires a comparable type parameter for == checks.
type SimpleValue[T comparable] struct {
	observableBase
	value T
}

// Set updates the value and notifies observers only if the value changed.
func (v *SimpleValue[T]) Set(n T) {
	if v.value == n {
		return
	}
	v.value = n
	v.notifyChanged(v)
}

// Get returns the current value and subscribes obs.
func (v *SimpleValue[T]) Get(obs Observer) T {
	v.maybeAddObserver(v, obs)
	return v.value
}

// Peek returns the current value without subscribing an observer.
func (v *SimpleValue[T]) Peek() T {
	return v.value
}

// Update allows in-place modification of the value.
// Notifies observers only if the value changed.
func (v *SimpleValue[T]) Update(cb func(*T)) {
	old := v.value
	cb(&v.value)
	if old != v.value {
		v.notifyChanged(v)
	}
}

// MaybeUpdate allows conditional in-place modification.
// The callback returns true if the value was changed and observers should be notified.
func (v *SimpleValue[T]) MaybeUpdate(cb func(*T) bool) {
	if cb(&v.value) {
		v.notifyChanged(v)
	}
}

// Observe subscribes obs and returns a getter for repeated reads.
func (v *SimpleValue[T]) Observe(obs Observer) ValueGetter[T] {
	v.maybeAddObserver(v, obs)
	return &SimpleValueGetter[T]{
		val: &v.value,
	}
}

// SimpleValueGetter provides direct access to the underlying value.
type SimpleValueGetter[T comparable] struct {
	val *T
}

func (v SimpleValueGetter[T]) Get() T {
	return *v.val
}
