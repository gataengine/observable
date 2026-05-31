package xset

import (
	"iter"
	"maps"
)

func NewSet[E comparable]() Set[E] {
	return make(Set[E], 10)
}

type Set[E comparable] map[E]struct{}

func (s Set[E]) Add(e E) {
	s[e] = struct{}{}
}
func (s Set[E]) Remove(e E) {
	delete(s, e)
}

func (s Set[E]) Contains(e E) bool {
	_, ok := s[e]
	return ok
}

func (s Set[E]) Has(e E) bool {
	return s.Contains(e)
}

func (s Set[E]) Keys() iter.Seq[E] {
	return maps.Keys(s)
}

func (s Set[E]) Len() int {
	return len(s)
}
