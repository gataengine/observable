package hybridset

import "sync"

// Pool is a typed slice pool to avoid allocations.
type Pool[T any] struct {
	pool sync.Pool
}

// NewPool creates a new typed slice pool with the given default capacity.
func NewPool[T any](defaultCap int) *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() any {
				s := make([]T, 0, defaultCap)
				return &s
			},
		},
	}
}

// Get returns a slice pointer with the requested size.
// The slice may have larger capacity from a previous use.
func (p *Pool[T]) Get(size int) *[]T {
	ptr := p.pool.Get().(*[]T)
	if cap(*ptr) < size {
		*ptr = make([]T, size)
	} else {
		*ptr = (*ptr)[:size]
	}
	return ptr
}

// Put returns a slice pointer to the pool.
// The slice is cleared before returning.
func (p *Pool[T]) Put(ptr *[]T) {
	clear(*ptr)
	*ptr = (*ptr)[:0]
	p.pool.Put(ptr)
}
