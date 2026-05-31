package hybridset

import (
	"sync"

	"github.com/puzpuzpuz/xsync/v4"
)

const (
	// DefaultPrealloc is the default preallocated capacity for slice and map.
	// Should be tuned based on expected average set size.
	DefaultPrealloc = 4

	// DefaultThreshold is the default size at which the set upgrades from slice+map to xsync.MapOf.
	// This is based on the cost of Add/Delete operations - at some point xsync becomes more efficient.
	DefaultThreshold = 32
)

// Set is a hybrid set that uses slice+map for small sizes and upgrades to xsync.MapOf
// for larger sizes. This provides fast iteration for small sets and good concurrency
// for larger sets.
//
// Set is safe for concurrent use. Small mode uses RWMutex, large mode uses xsync.MapOf.
type Set[T comparable] struct {
	mu        sync.RWMutex // protects small mode
	threshold int

	// Small mode (len < threshold)
	slice []T
	smap  map[T]struct{}

	// Large mode (len >= threshold)
	xmap *xsync.MapOf[T, struct{}]
}

// New creates a new hybrid set with default prealloc and threshold.
func New[T comparable]() *Set[T] {
	return NewWithConfig[T](DefaultPrealloc, DefaultThreshold)
}

// NewWithPrealloc creates a new hybrid set with custom prealloc and default threshold.
func NewWithPrealloc[T comparable](prealloc int) *Set[T] {
	return NewWithConfig[T](prealloc, DefaultThreshold)
}

// NewWithThreshold creates a new hybrid set with default prealloc and custom threshold.
func NewWithThreshold[T comparable](threshold int) *Set[T] {
	return NewWithConfig[T](DefaultPrealloc, threshold)
}

// NewWithConfig creates a new hybrid set with custom prealloc and threshold.
// - prealloc: initial capacity for slice and map (based on expected average size)
// - threshold: size at which to upgrade to xsync.MapOf (based on Add/Delete cost)
func NewWithConfig[T comparable](prealloc, threshold int) *Set[T] {
	return &Set[T]{
		threshold: threshold,
		slice:     make([]T, 0, prealloc),
		smap:      make(map[T]struct{}, prealloc),
	}
}

// Add adds an item to the set. Returns true if the item was added (not already present).
func (s *Set[T]) Add(item T) bool {
	if s.xmap != nil {
		// Large mode - xsync is concurrent-safe
		_, loaded := s.xmap.LoadOrStore(item, struct{}{})
		return !loaded
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Small mode - check if already exists
	if _, exists := s.smap[item]; exists {
		return false
	}

	s.smap[item] = struct{}{}
	s.slice = append(s.slice, item)

	// Check if we need to upgrade
	if len(s.slice) >= s.threshold {
		s.upgrade()
	}

	return true
}

// Delete removes an item from the set. Returns true if the item was present.
func (s *Set[T]) Delete(item T) bool {
	if s.xmap != nil {
		// Large mode - xsync is concurrent-safe
		_, loaded := s.xmap.LoadAndDelete(item)
		return loaded
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Small mode
	if _, exists := s.smap[item]; !exists {
		return false
	}

	delete(s.smap, item)

	// Remove from slice (swap with last element for O(1))
	for i, v := range s.slice {
		if v == item {
			s.slice[i] = s.slice[len(s.slice)-1]
			s.slice = s.slice[:len(s.slice)-1]
			break
		}
	}

	return true
}

// Contains returns true if the item exists in the set.
func (s *Set[T]) Contains(item T) bool {
	if s.xmap != nil {
		// Large mode - xsync is concurrent-safe
		_, exists := s.xmap.Load(item)
		return exists
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.smap[item]
	return exists
}

// Size returns the number of items in the set.
func (s *Set[T]) Size() int {
	if s.xmap != nil {
		// Large mode - xsync is concurrent-safe
		return s.xmap.Size()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.slice)
}

// Range iterates over items if in xsync mode and returns false.
// Returns true without iterating if in small mode (caller should use CopyTo instead).
// This design allows zero-allocation iteration in both modes when used with Pool.
func (s *Set[T]) Range(fn func(item T) bool) (useCopyTo bool) {
	if s.xmap != nil {
		// Large mode - xsync.Range is concurrent-safe, iterate directly
		s.xmap.Range(func(item T, _ struct{}) bool {
			return fn(item)
		})
		return false
	}
	// Small mode - signal caller to use CopyTo with pooled slice
	return true
}

// CopyTo copies all items into dst and returns the count copied.
// dst must have capacity >= Size(). Safe for concurrent use.
// Use with Pool for zero-allocation iteration in small mode.
func (s *Set[T]) CopyTo(dst []T) int {
	if s.xmap != nil {
		// Large mode
		i := 0
		s.xmap.Range(func(item T, _ struct{}) bool {
			dst[i] = item
			i++
			return true
		})
		return i
	}

	s.mu.RLock()
	n := copy(dst, s.slice)
	s.mu.RUnlock()
	return n
}

// IsUpgraded returns true if the set has been upgraded to xsync mode.
func (s *Set[T]) IsUpgraded() bool {
	return s.xmap != nil
}

// upgrade converts from small mode to large mode
func (s *Set[T]) upgrade() {
	s.xmap = xsync.NewMapOf[T, struct{}](xsync.WithPresize(len(s.slice)))
	for _, item := range s.slice {
		s.xmap.Store(item, struct{}{})
	}
	// Clear small mode storage
	s.slice = nil
	s.smap = nil
}
