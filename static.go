package observable

// Static returns a Value that always reads as v and never notifies observers.
func Static[T any](v T) Value[T] {
	return staticValue[T]{value: v}
}

type staticValue[T any] struct {
	value T
}

func (s staticValue[T]) Set(T) {}

func (s staticValue[T]) Update(func(*T)) {}

func (s staticValue[T]) MaybeUpdate(func(*T) bool) {}

func (s staticValue[T]) Get(obs Observer) T {
	return s.value
}

func (s staticValue[T]) Peek() T {
	return s.value
}

func (s staticValue[T]) Observe(obs Observer) ValueGetter[T] {
	return staticValueGetter[T]{value: s.value}
}

func (s staticValue[T]) RemoveObserver(obs Observer) {}

type staticValueGetter[T any] struct {
	value T
}

func (g staticValueGetter[T]) Get() T {
	return g.value
}
