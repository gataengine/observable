package xset_test

import (
	"github.com/gataengine/observable/internal/xset"
	"slices"
	"testing"
)

type ti interface {
	Get()
}

type impl struct {
	test string
}

func (*impl) Get() {}

func BenchmarkSlice2(b *testing.B) {
	list := []ti{}
	var last ti
	for i := 0; i < 5; i++ {
		list = append(list, new(impl))
		last = list[i]
		//log.Printf("%p\n", last)
	}
	b.ResetTimer()
	for b.Loop() {
		_ = slices.Contains(list, last)
	}
}

func BenchmarkXset2(b *testing.B) {
	xs := xset.NewSet[ti]()
	var last ti
	for i := 0; i < 5; i++ {
		last = &impl{}
		xs.Add(last)
	}
	b.ResetTimer()
	for b.Loop() {
		_ = xs.Contains(last)
	}
}

func BenchmarkSlice5(b *testing.B) {
	list := []ti{}
	var last ti
	for i := 0; i < 5; i++ {
		list = append(list, new(impl))
		last = list[i]
		//log.Printf("%p\n", last)
	}
	b.ResetTimer()
	for b.Loop() {
		_ = slices.Contains(list, last)
	}
}

func BenchmarkXset5(b *testing.B) {
	xs := xset.NewSet[ti]()
	var last ti
	for i := 0; i < 5; i++ {
		last = &impl{}
		xs.Add(last)
	}
	b.ResetTimer()
	for b.Loop() {
		_ = xs.Contains(last)
	}
}

func BenchmarkSlice10(b *testing.B) {
	list := []ti{}
	var last ti
	for i := 0; i < 10; i++ {
		list = append(list, new(impl))
		last = list[i]
		//log.Printf("%p\n", last)
	}
	b.ResetTimer()
	for b.Loop() {
		_ = slices.Contains(list, last)
	}
}

func BenchmarkXset10(b *testing.B) {
	xs := xset.NewSet[ti]()
	var last ti
	for i := 0; i < 10; i++ {
		last = &impl{}
		xs.Add(last)
	}
	b.ResetTimer()
	for b.Loop() {
		_ = xs.Contains(last)
	}
}

func BenchmarkSlice100(b *testing.B) {

	list := []ti{}
	var last ti
	for i := 0; i < 100; i++ {
		list = append(list, new(impl))
		last = list[i]
	}
	b.ResetTimer()
	for b.Loop() {
		_ = slices.Contains(list, last)
	}
}

func BenchmarkXset100(b *testing.B) {
	xs := xset.NewSet[ti]()
	var last ti
	for i := 0; i < 100; i++ {
		last = &impl{}
		xs.Add(last)
	}
	b.ResetTimer()
	for b.Loop() {
		_ = xs.Contains(last)
	}
}
