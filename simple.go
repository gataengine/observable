package observable

// SimpleNonComparable creates a new observable value with the given initial value.
// Unlike Simple, it does not deduplicate — every Set triggers a notification.
func SimpleNonComparable[T any](initial T) *NonComparableValue[T] {
	return &NonComparableValue[T]{
		value: initial,
	}
}

// NonComparableValue is a basic observable that holds a single value.
// It does not deduplicate notifications (use SimpleValue for comparable types).
type NonComparableValue[T any] struct {
	observableBase
	value T
}

// Set updates the value and notifies all observers.
func (v *NonComparableValue[T]) Set(n T) {
	v.value = n
	v.notifyChanged(v)
}

// Get returns the current value and subscribes the observer.
func (v *NonComparableValue[T]) Get(obs Observer) T {
	v.maybeAddObserver(v, obs)
	return v.value
}

// Peek returns the current value without subscribing any observer.
func (v *NonComparableValue[T]) Peek() T {
	return v.value
}

// Update allows in-place modification of the value.
func (v *NonComparableValue[T]) Update(cb func(*T)) {
	cb(&v.value)
	v.notifyChanged(v)
}

// MaybeUpdate allows conditional in-place modification.
// The callback returns true if the value was changed and observers should be notified.
func (v *NonComparableValue[T]) MaybeUpdate(cb func(*T) bool) {
	if cb(&v.value) {
		v.notifyChanged(v)
	}
}

// Observe returns a ValueGetter for repeated access without re-subscribing.
func (v *NonComparableValue[T]) Observe(obs Observer) ValueGetter[T] {
	v.maybeAddObserver(v, obs)
	return &NonComparableValueGetter[T]{
		val: &v.value,
	}
}

// NonComparableValueGetter provides direct access to the underlying value.
type NonComparableValueGetter[T any] struct {
	val *T
}

func (v NonComparableValueGetter[T]) Get() T {
	return *v.val
}
