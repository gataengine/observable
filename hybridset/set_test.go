package hybridset

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/puzpuzpuz/xsync/v4"
)

func TestHybridSet(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HybridSet Suite")
}

var _ = Describe("Set", func() {
	Describe("Add", func() {
		It("adds new items and returns true", func() {
			s := New[int]()
			Expect(s.Add(1)).To(BeTrue())
			Expect(s.Size()).To(Equal(1))
		})

		It("returns false for duplicate items", func() {
			s := New[int]()
			s.Add(1)
			Expect(s.Add(1)).To(BeFalse())
			Expect(s.Size()).To(Equal(1))
		})

		It("handles multiple items", func() {
			s := New[int]()
			s.Add(1)
			s.Add(2)
			s.Add(3)
			Expect(s.Size()).To(Equal(3))
		})
	})

	Describe("Delete", func() {
		It("removes existing items and returns true", func() {
			s := New[int]()
			s.Add(1)
			s.Add(2)
			Expect(s.Delete(1)).To(BeTrue())
			Expect(s.Size()).To(Equal(1))
			Expect(s.Contains(1)).To(BeFalse())
		})

		It("returns false for non-existent items", func() {
			s := New[int]()
			s.Add(1)
			Expect(s.Delete(2)).To(BeFalse())
			Expect(s.Size()).To(Equal(1))
		})

		It("maintains correct state after multiple deletes", func() {
			s := New[int]()
			s.Add(1)
			s.Add(2)
			s.Add(3)
			s.Delete(2)
			Expect(s.Contains(1)).To(BeTrue())
			Expect(s.Contains(2)).To(BeFalse())
			Expect(s.Contains(3)).To(BeTrue())
		})
	})

	Describe("Contains", func() {
		It("returns true for existing items", func() {
			s := New[int]()
			s.Add(1)
			Expect(s.Contains(1)).To(BeTrue())
		})

		It("returns false for non-existent items", func() {
			s := New[int]()
			s.Add(1)
			Expect(s.Contains(2)).To(BeFalse())
		})
	})

	Describe("Range", func() {
		It("returns useCopyTo=true in small mode without iterating", func() {
			s := NewWithThreshold[int](10)
			for i := 0; i < 5; i++ {
				s.Add(i)
			}

			callCount := 0
			useCopyTo := s.Range(func(item int) bool {
				callCount++
				return true
			})

			Expect(useCopyTo).To(BeTrue())
			Expect(callCount).To(Equal(0)) // callback never called in small mode
		})

		It("iterates and returns useCopyTo=false in xsync mode", func() {
			s := NewWithThreshold[int](4)
			for i := 0; i < 10; i++ {
				s.Add(i)
			}

			Expect(s.IsUpgraded()).To(BeTrue())

			collected := make([]int, 0)
			useCopyTo := s.Range(func(item int) bool {
				collected = append(collected, item)
				return true
			})

			Expect(useCopyTo).To(BeFalse())
			Expect(collected).To(HaveLen(10))
			Expect(collected).To(ContainElements(0, 1, 2, 3, 4, 5, 6, 7, 8, 9))
		})

		It("stops when callback returns false in xsync mode", func() {
			s := NewWithThreshold[int](4)
			for i := 0; i < 10; i++ {
				s.Add(i)
			}

			count := 0
			s.Range(func(item int) bool {
				count++
				return count < 3
			})

			Expect(count).To(Equal(3))
		})
	})

	Describe("CopyTo", func() {
		It("copies all items in small mode", func() {
			s := NewWithThreshold[int](10)
			for i := 0; i < 5; i++ {
				s.Add(i)
			}

			dst := make([]int, s.Size())
			n := s.CopyTo(dst)

			Expect(n).To(Equal(5))
			Expect(dst).To(ContainElements(0, 1, 2, 3, 4))
		})

		It("copies all items in xsync mode", func() {
			s := NewWithThreshold[int](4)
			for i := 0; i < 10; i++ {
				s.Add(i)
			}

			dst := make([]int, s.Size())
			n := s.CopyTo(dst)

			Expect(n).To(Equal(10))
			Expect(dst).To(ContainElements(0, 1, 2, 3, 4, 5, 6, 7, 8, 9))
		})
	})

	Describe("Pool", func() {
		It("provides and accepts slices", func() {
			pool := NewPool[int](8)

			ptr := pool.Get(5)
			Expect(*ptr).To(HaveLen(5))
			Expect(cap(*ptr)).To(BeNumerically(">=", 5))

			// Fill with data
			for i := range *ptr {
				(*ptr)[i] = i
			}

			pool.Put(ptr)

			// Get again - should be cleared
			ptr2 := pool.Get(3)
			Expect(*ptr2).To(HaveLen(3))
			// Values should be zeroed after Put
			for _, v := range *ptr2 {
				Expect(v).To(Equal(0))
			}
		})

		It("grows slice when needed", func() {
			pool := NewPool[int](4)

			ptr := pool.Get(10)
			Expect(*ptr).To(HaveLen(10))
			Expect(cap(*ptr)).To(BeNumerically(">=", 10))
		})
	})

	Describe("upgrade behavior", func() {
		It("stays in small mode below threshold", func() {
			s := NewWithThreshold[int](4)
			s.Add(1)
			s.Add(2)
			s.Add(3)
			Expect(s.IsUpgraded()).To(BeFalse())
		})

		It("upgrades at threshold", func() {
			s := NewWithThreshold[int](4)
			s.Add(1)
			s.Add(2)
			s.Add(3)
			s.Add(4)
			Expect(s.IsUpgraded()).To(BeTrue())
		})

		It("preserves all items after upgrade", func() {
			s := NewWithThreshold[int](4)
			for i := 0; i < 10; i++ {
				s.Add(i)
			}

			Expect(s.IsUpgraded()).To(BeTrue())
			Expect(s.Size()).To(Equal(10))

			for i := 0; i < 10; i++ {
				Expect(s.Contains(i)).To(BeTrue())
			}
		})

		It("Delete works after upgrade", func() {
			s := NewWithThreshold[int](4)
			for i := 0; i < 10; i++ {
				s.Add(i)
			}

			Expect(s.Delete(5)).To(BeTrue())
			Expect(s.Size()).To(Equal(9))
			Expect(s.Contains(5)).To(BeFalse())
		})
	})
})

// Benchmarks

func BenchmarkFirstAdd(b *testing.B) {
	thresholds := []int{4, 8, 16, 32}

	for _, threshold := range thresholds {
		b.Run(fmt.Sprintf("hybrid_t%d", threshold), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				s := NewWithThreshold[int](threshold)
				s.Add(1)
			}
		})
	}

	b.Run("xsync", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			s := xsync.NewMapOf[int, struct{}]()
			s.Store(1, struct{}{})
		}
	})

	b.Run("map", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			s := make(map[int]struct{})
			s[1] = struct{}{}
		}
	})
}

func BenchmarkCopyTo(b *testing.B) {
	sizes := []int{1, 3, 10, 32}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("small_%d", size), func(b *testing.B) {
			s := NewWithThreshold[int](size + 10)
			for i := 0; i < size; i++ {
				s.Add(i)
			}
			dst := make([]int, size)
			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				s.CopyTo(dst)
			}
		})

		b.Run(fmt.Sprintf("small_pooled_%d", size), func(b *testing.B) {
			s := NewWithThreshold[int](size + 10)
			for i := 0; i < size; i++ {
				s.Add(i)
			}
			pool := NewPool[int](32)
			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				ptr := pool.Get(size)
				s.CopyTo(*ptr)
				pool.Put(ptr)
			}
		})
	}
}

func BenchmarkRangeXsync(b *testing.B) {
	sizes := []int{32, 64, 128}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("xsync_%d", size), func(b *testing.B) {
			s := NewWithThreshold[int](4) // force upgrade
			for i := 0; i < size; i++ {
				s.Add(i)
			}
			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				s.Range(func(item int) bool {
					return true
				})
			}
		})
	}
}

func BenchmarkContains(b *testing.B) {
	sizes := []int{3, 10, 50}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("hybrid_small_%d", size), func(b *testing.B) {
			s := NewWithThreshold[int](size + 10)
			for i := 0; i < size; i++ {
				s.Add(i)
			}
			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				s.Contains(size / 2)
			}
		})

		b.Run(fmt.Sprintf("hybrid_large_%d", size), func(b *testing.B) {
			s := NewWithThreshold[int](1)
			for i := 0; i < size; i++ {
				s.Add(i)
			}
			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				s.Contains(size / 2)
			}
		})

		b.Run(fmt.Sprintf("xsync_%d", size), func(b *testing.B) {
			s := xsync.NewMapOf[int, struct{}]()
			for i := 0; i < size; i++ {
				s.Store(i, struct{}{})
			}
			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				s.Load(size / 2)
			}
		})

		b.Run(fmt.Sprintf("map_%d", size), func(b *testing.B) {
			m := make(map[int]struct{})
			for i := 0; i < size; i++ {
				m[i] = struct{}{}
			}
			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				_, _ = m[size/2]
			}
		})
	}
}

// BenchmarkAddAtSize measures the cost of Add at various set sizes
func BenchmarkAddAtSize(b *testing.B) {
	sizes := []int{4, 8, 16, 32, 64, 128}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("hybrid_add_at_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				s := NewWithConfig[int](4, size+10)
				for j := 0; j < size; j++ {
					s.Add(j)
				}
				s.Add(size)
			}
		})

		b.Run(fmt.Sprintf("xsync_add_at_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				s := xsync.NewMapOf[int, struct{}]()
				for j := 0; j < size; j++ {
					s.Store(j, struct{}{})
				}
				s.Store(size, struct{}{})
			}
		})

		b.Run(fmt.Sprintf("map_add_at_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				m := make(map[int]struct{})
				for j := 0; j < size; j++ {
					m[j] = struct{}{}
				}
				m[size] = struct{}{}
			}
		})
	}
}

// BenchmarkDeleteAtSize measures the cost of Delete at various set sizes
func BenchmarkDeleteAtSize(b *testing.B) {
	sizes := []int{4, 8, 16, 32, 64, 128}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("hybrid_del_at_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				s := NewWithConfig[int](4, size+10)
				for j := 0; j < size; j++ {
					s.Add(j)
				}
				s.Delete(size / 2)
			}
		})

		b.Run(fmt.Sprintf("xsync_del_at_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				s := xsync.NewMapOf[int, struct{}]()
				for j := 0; j < size; j++ {
					s.Store(j, struct{}{})
				}
				s.Delete(size / 2)
			}
		})

		b.Run(fmt.Sprintf("map_del_at_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				m := make(map[int]struct{})
				for j := 0; j < size; j++ {
					m[j] = struct{}{}
				}
				delete(m, size/2)
			}
		})
	}
}

// BenchmarkSteadyStateAdd measures Add cost on pre-existing set
func BenchmarkSteadyStateAdd(b *testing.B) {
	sizes := []int{8, 16, 32, 64}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("hybrid_%d", size), func(b *testing.B) {
			s := NewWithConfig[int](4, size+10)
			for j := 0; j < size; j++ {
				s.Add(j)
			}
			next := size
			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				s.Add(next)
				s.Delete(next)
				next++
			}
		})

		b.Run(fmt.Sprintf("xsync_%d", size), func(b *testing.B) {
			s := xsync.NewMapOf[int, struct{}]()
			for j := 0; j < size; j++ {
				s.Store(j, struct{}{})
			}
			next := size
			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				s.Store(next, struct{}{})
				s.Delete(next)
				next++
			}
		})

		b.Run(fmt.Sprintf("map_%d", size), func(b *testing.B) {
			m := make(map[int]struct{})
			for j := 0; j < size; j++ {
				m[j] = struct{}{}
			}
			next := size
			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				m[next] = struct{}{}
				delete(m, next)
				next++
			}
		})
	}
}
