# Observable Independent Library Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `github.com/gataengine/observable` present as an independent Go reactive primitives library through README, Go docs, and benchmark coverage while preserving the public API and all existing benchmark scenarios.

**Architecture:** Keep the runtime implementation and public names unchanged. Add benchmark coverage around standalone primitives without removing or renaming any existing benchmark groups, then rewrite docs around the library's stable concepts: values, observers, computed derivations, lists, maps, registry lifecycle management, and weak-pointer standalone subscriptions.

**Tech Stack:** Go 1.26 module, standard `testing` benchmarks, package-level Go documentation comments, Markdown README.

---

### Task 1: Preserve Existing Benchmark Surface And Add Primitive Benchmarks

**Files:**
- Modify: `benchmark_test.go`

- [ ] **Step 1: Capture existing benchmark names before editing**

Run:

```sh
go test -run '^$' -bench=. -list '^Benchmark' ./...
```

Expected: command exits successfully and lists existing benchmarks, including `BenchmarkUIScenario`.

- [ ] **Step 2: Add primitive value, list, and map benchmarks**

In `benchmark_test.go`, keep all existing benchmark functions. Add the following benchmark functions after `BenchmarkGetSet` and before `BenchmarkComputedChain`:

```go
// =============================================================================
// Primitive Read/Write Benchmarks
// =============================================================================

func BenchmarkValuePrimitive(b *testing.B) {
	b.Run("peek", func(b *testing.B) {
		v := Simple(42)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = v.Peek()
		}
	})

	b.Run("registry_get", func(b *testing.B) {
		reg := NewRegistry()
		obs := &benchRegistryObserver{registry: reg}
		v := Simple(42)
		v.Get(obs)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = v.Get(obs)
		}
	})

	b.Run("weak_pointer_get", func(b *testing.B) {
		obs := &BasicObserver{}
		v := Simple(42)
		v.Get(obs)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = v.Get(obs)
		}
	})

	b.Run("registry_observe_getter", func(b *testing.B) {
		reg := NewRegistry()
		obs := &benchRegistryObserver{registry: reg}
		v := Simple(42)
		getter := v.Observe(obs)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = getter.Get()
		}
	})

	b.Run("weak_pointer_observe_getter", func(b *testing.B) {
		obs := &BasicObserver{}
		v := Simple(42)
		getter := v.Observe(obs)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = getter.Get()
		}
	})
}

func BenchmarkListPrimitive(b *testing.B) {
	b.Run("registry_getter_at", func(b *testing.B) {
		reg := NewRegistry()
		obs := &benchRegistryObserver{registry: reg}
		list := NewList[int]()
		for i := range 100 {
			list.Add(i)
		}
		getter := list.Observe(obs)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_, _, _ = getter.At(50)
		}
	})

	b.Run("weak_pointer_getter_at", func(b *testing.B) {
		obs := &BasicObserver{}
		list := NewList[int]()
		for i := range 100 {
			list.Add(i)
		}
		getter := list.Observe(obs)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_, _, _ = getter.At(50)
		}
	})

	b.Run("registry_set", func(b *testing.B) {
		reg := NewRegistry()
		obs := &benchRegistryObserver{registry: reg}
		list := NewList[int]()
		for i := range 100 {
			list.Add(i)
		}
		list.Observe(obs)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			_ = list.Set(50, i)
		}
	})

	b.Run("weak_pointer_set", func(b *testing.B) {
		obs := &BasicObserver{}
		list := NewList[int]()
		for i := range 100 {
			list.Add(i)
		}
		list.Observe(obs)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			_ = list.Set(50, i)
		}
	})
}

func BenchmarkMapPrimitive(b *testing.B) {
	b.Run("registry_getter_get", func(b *testing.B) {
		reg := NewRegistry()
		obs := &benchRegistryObserver{registry: reg}
		m := NewMap[string, int]()
		for i := range 100 {
			m.Set("key-"+itoa(i), i)
		}
		getter := m.Observe(obs)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_, _ = getter.Get("key-50")
		}
	})

	b.Run("weak_pointer_getter_get", func(b *testing.B) {
		obs := &BasicObserver{}
		m := NewMap[string, int]()
		for i := range 100 {
			m.Set("key-"+itoa(i), i)
		}
		getter := m.Observe(obs)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_, _ = getter.Get("key-50")
		}
	})

	b.Run("registry_set", func(b *testing.B) {
		reg := NewRegistry()
		obs := &benchRegistryObserver{registry: reg}
		m := NewMap[string, int]()
		for i := range 100 {
			m.Set("key-"+itoa(i), i)
		}
		m.Observe(obs)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			m.Set("key-50", i)
		}
	})

	b.Run("weak_pointer_set", func(b *testing.B) {
		obs := &BasicObserver{}
		m := NewMap[string, int]()
		for i := range 100 {
			m.Set("key-"+itoa(i), i)
		}
		m.Observe(obs)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			m.Set("key-50", i)
		}
	})
}
```

- [ ] **Step 3: Verify benchmark names still include existing scenarios**

Run:

```sh
go test -run '^$' -bench=. -list '^Benchmark' ./...
```

Expected: command exits successfully. Output includes all pre-existing benchmark groups and the added `BenchmarkValuePrimitive`, `BenchmarkListPrimitive`, and `BenchmarkMapPrimitive` groups. `BenchmarkUIScenario` is still listed.

- [ ] **Step 4: Run package tests after benchmark additions**

Run:

```sh
go test ./...
```

Expected: all package tests pass.

- [ ] **Step 5: Commit benchmark additions**

Run:

```sh
git add benchmark_test.go
git commit -m "bench: add observable primitive benchmarks"
```

Expected: commit succeeds and includes only `benchmark_test.go`.

---

### Task 2: Clean Up Exported Go Documentation

**Files:**
- Modify: `interfaces.go`
- Modify: `observer.go`
- Modify: `registry.go`
- Modify: `simple.go`
- Modify: `comparable.go`
- Modify: `computed.go`
- Modify: `computed_list.go`
- Modify: `list.go`
- Modify: `map.go`
- Modify: `mapped.go`
- Modify: `bidi_mapped.go`
- Modify: `static.go`

- [ ] **Step 1: Update interface comments**

In `interfaces.go`, replace the exported interface comments with library-facing wording. Keep type and method names unchanged:

```go
// Observer receives change notifications from values it has subscribed to.
//
// Implementations usually embed BasicObserver. Passing an Observer to Get or
// Observe subscribes it to the source, unless the observer is Noop.
type Observer interface {
	MarkUpdated()
	GetObserver() *observerState
}

// Observable is implemented by values that can remove a previously subscribed observer.
type Observable interface {
	RemoveObserver(obs Observer)
}

// RegistryProvider lets an observer route subscriptions through a Registry.
//
// The registry path is useful for owners with explicit lifecycles, such as UI
// widgets or request-scoped views. CurrentObserver returns the observer identity
// stored in the registry; for computed values this is the computed value itself.
type RegistryProvider interface {
	ObservableRegistry() *Registry
	CurrentObserver() Observer
}

// DependentObservable is both an observable source and an observer of upstream sources.
//
// Registry uses this marker shape to recursively clean up computed and mapped
// values when their downstream observers are unsubscribed.
type DependentObservable interface {
	Observable
	Observer
}

// ROValue is a read-only observable value.
//
// Get subscribes the observer and returns the current value. Peek returns the
// current value without subscribing. Observe subscribes once and returns a
// getter for repeated reads.
type ROValue[T any] interface {
	Get(obs Observer) T
	Peek() T
	Observe(obs Observer) ValueGetter[T]
	RemoveObserver(obs Observer)
}

// Value is a mutable observable value.
type Value[T any] interface {
	ROValue[T]
	Set(T)
	Update(func(*T))
	MaybeUpdate(func(*T) bool)
}

// ValueGetter reads a subscribed value without re-subscribing on every access.
type ValueGetter[T any] interface {
	Get() T
}

// ROList is a read-only observable list with stable item keys.
//
// Len, At, and All subscribe the observer. PeekLen, PeekAt, and PeekAll read
// without subscribing.
type ROList[T any] interface {
	Observe(obs Observer) ListGetter[T]
	Len(obs Observer) int
	At(obs Observer, index int) (key int64, value T, ok bool)
	All(obs Observer) iter.Seq2[int64, T]
	PeekLen() int
	PeekAt(index int) (key int64, value T, ok bool)
	PeekAll() iter.Seq2[int64, T]
	RemoveObserver(obs Observer)
}

// ListGetter reads a subscribed list without re-subscribing on every access.
type ListGetter[T any] interface {
	Len() int
	At(index int) (key int64, value T, ok bool)
	Keys() []int64
	Values() []T
	All() iter.Seq2[int64, T]
	IndexOf(key int64) int
	ValueByKey(key int64) (value T, ok bool)
}

// ROMap is a read-only observable map.
//
// Get and All subscribe the observer. Peek, PeekLen, and PeekAll read without
// subscribing.
type ROMap[K comparable, V any] interface {
	Observe(obs Observer) *MapGetter[K, V]
	Get(obs Observer, key K) (V, bool)
	All(obs Observer) iter.Seq2[K, V]
	Peek(key K) (V, bool)
	PeekLen() int
	PeekAll() iter.Seq2[K, V]
	RemoveObserver(obs Observer)
}
```

- [ ] **Step 2: Update observer and registry comments**

In `observer.go`, make `Noop` and `BasicObserver` comments describe public usage:

```go
// Noop can be passed to Get when a caller wants a subscribed-read API without
// creating a subscription.
var Noop Observer = &noopObserver{}
```

```go
// BasicObserver is a minimal Observer implementation.
//
// Embed it in application types or allocate it directly when a caller only
// needs to know whether any subscribed source has changed.
type BasicObserver struct {
	observerState
}
```

In `registry.go`, update the exported comments:

```go
// Registry owns subscriptions for observers with explicit lifecycles.
//
// Observers that implement RegistryProvider use this path automatically. A
// registry can unsubscribe an observer from all sources at once, drain dirty
// observers, and cascade cleanup through computed or mapped values that no
// longer have downstream observers.
type Registry struct {
```

```go
// NewRegistry creates an empty subscription registry.
func NewRegistry() *Registry {
```

```go
// Subscribe records that obs depends on o.
func (r *Registry) Subscribe(obs Observer, o Observable) {
```

```go
// NotifyObservable marks every observer subscribed to o as updated.
func (r *Registry) NotifyObservable(o Observable) {
```

```go
// MarkDirty marks obs as updated inside the registry.
func (r *Registry) MarkDirty(obs Observer) {
```

```go
// HasDirty reports whether the registry has updated observers waiting to drain.
func (r *Registry) HasDirty() bool {
```

```go
// DrainDirty returns all updated observers and clears the dirty set.
func (r *Registry) DrainDirty() []Observer {
```

```go
// Unsubscribe removes one observer/source subscription.
func (r *Registry) Unsubscribe(obs Observer, o Observable) {
```

```go
// UnsubscribeAll removes every subscription owned by obs.
//
// If this leaves a DependentObservable without downstream observers, the
// registry recursively removes that dependent observable from its upstream
// sources.
func (r *Registry) UnsubscribeAll(obs Observer) {
```

```go
// UnsubscribeObservable removes o and all observer subscriptions pointing to it.
func (r *Registry) UnsubscribeObservable(o Observable) {
```

- [ ] **Step 3: Update value and computed comments**

In `comparable.go`, `simple.go`, `computed.go`, `mapped.go`, `bidi_mapped.go`, `computed_list.go`, and `static.go`, update exported comments to use these terms exactly where applicable:

```go
// Get returns the current value and subscribes obs.
```

```go
// Peek returns the current value without subscribing an observer.
```

```go
// Observe subscribes obs and returns a getter for repeated reads.
```

```go
// ObservableRegistry implements RegistryProvider.
```

```go
// CurrentObserver implements RegistryProvider.
```

Also update constructor comments to stand alone:

```go
// Simple creates an observable value for comparable values.
```

```go
// SimpleNonComparable creates an observable value for values that cannot be compared.
```

```go
// Static returns a Value that always reads as v and never notifies observers.
```

```go
// NewComputed creates a read-only observable value derived from other observables.
```

```go
// NewCachedComputed creates a computed value with a separate dependency binding phase.
```

```go
// Mapped creates a read-only observable value derived from source.
```

```go
// NewBidiMapped creates a two-way mapped value backed by source.
```

```go
// NewComputedList creates a read-only observable list derived from other observables.
```

- [ ] **Step 4: Update list and map comments**

In `list.go`, keep stable-key semantics but remove widget-only framing:

```go
// ListItem is an item in an observable list.
//
// Key is stable across moves and swaps so callers can diff list identity
// separately from list position.
type ListItem[T any] struct {
```

```go
// List is a mutable observable list with stable item keys.
//
// Add, Insert, Set, and Replace assign new keys. Move and Swap preserve keys.
// List is safe for concurrent use.
type List[T any] struct {
```

In `map.go`, make the map documentation general:

```go
// Map is a mutable observable map.
//
// Key additions, replacements, deletions, merges, and clears notify observers.
// Map is safe for concurrent use.
type Map[K comparable, V any] struct {
```

- [ ] **Step 5: Run tests after comment cleanup**

Run:

```sh
go test ./...
```

Expected: all tests pass.

- [ ] **Step 6: Commit Go documentation cleanup**

Run:

```sh
git add interfaces.go observer.go registry.go simple.go comparable.go computed.go computed_list.go list.go map.go mapped.go bidi_mapped.go static.go
git commit -m "docs: polish observable package comments"
```

Expected: commit succeeds and includes only Go comment changes.

---

### Task 3: Rewrite README As Independent Library Documentation

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Replace README content**

Replace `README.md` with this structure and prose. Task 4 adds the measured benchmark table after a local benchmark run.

```markdown
# Observable

`observable` is a small reactive primitives library for Go. It provides observable values, computed derivations, observable lists, observable maps, and explicit subscription lifecycle management for applications that need predictable change propagation.

The API is inspired by MobX-style observable and computed dependency tracking, implemented as idiomatic Go primitives.

Originally developed for Gata Engine UI, this module is distributed as the standalone package `github.com/gataengine/observable`.

## Install

```sh
go get github.com/gataengine/observable
```

## Quick Start

```go
package main

import (
	"fmt"

	"github.com/gataengine/observable"
)

func main() {
	name := observable.Simple("Alice")

	var obs observable.BasicObserver
	getter := name.Observe(&obs)

	fmt.Println(getter.Get())

	name.Set("Bob")
	if obs.GetAndResetUpdated() {
		fmt.Println(getter.Get())
	}
}
```

## Concepts

Observable values expose subscribed reads and unsubscribed reads:

- `Get(obs)` returns the current value and subscribes `obs`.
- `Peek()` returns the current value without subscribing.
- `Observe(obs)` subscribes once and returns a getter for repeated reads.

Observers are notified when a subscribed source changes. For simple use cases, embed or allocate `BasicObserver` and check `IsUpdated` or `GetAndResetUpdated`.

Computed values derive from other observables. Dependencies are tracked when the compute function reads sources through the observer passed into the function.

```go
a := observable.Simple(1)
b := observable.Simple(2)

sum := observable.NewComputed(func(obs observable.Observer) int {
	return a.Get(obs) + b.Get(obs)
})

var obs observable.BasicObserver
value := sum.Observe(&obs)

fmt.Println(value.Get())
a.Set(10)
fmt.Println(obs.GetAndResetUpdated(), value.Get())
```

## Values

Use `Simple` for comparable values. Setting the same value again does not notify observers.

```go
count := observable.Simple(0)
count.Set(1)
count.Update(func(v *int) {
	*v = *v + 1
})
```

Use `SimpleNonComparable` when the value type cannot be compared or when every `Set` should notify observers.

```go
state := observable.SimpleNonComparable([]string{"ready"})
state.Set([]string{"ready"})
```

## Lists

`List` stores values with stable item keys. Keys let callers distinguish item identity from item position.

```go
items := observable.NewList[string]()
items.Add("a", "b", "c")
items.Set(1, "B")
items.Move(2, 0)

var obs observable.BasicObserver
view := items.Observe(&obs)

for key, value := range view.All() {
	fmt.Println(key, value)
}
```

## Maps

`Map` is an observable map keyed by caller-provided comparable keys.

```go
counts := observable.NewMap[string, int]()
counts.Set("open", 3)
counts.Set("closed", 5)

var obs observable.BasicObserver
view := counts.Observe(&obs)

value, ok := view.Get("open")
fmt.Println(value, ok)
```

## Subscription Modes

By default, observables keep weak references to standalone observers. This is convenient for direct use: create a `BasicObserver`, observe values, and read the updated flag when sources change.

For owners with explicit lifecycles, use `Registry`. Observers that implement `RegistryProvider` route subscriptions through the registry, allowing the owner to unsubscribe from every source in one call.

```go
type Widget struct {
	observable.BasicObserver
	registry *observable.Registry
}

func (w *Widget) ObservableRegistry() *observable.Registry {
	return w.registry
}

func (w *Widget) CurrentObserver() observable.Observer {
	return w
}

reg := observable.NewRegistry()
widget := &Widget{registry: reg}

name := observable.Simple("Alice")
_ = name.Observe(widget)

name.Set("Bob")
dirty := reg.DrainDirty()
fmt.Println(len(dirty))

reg.UnsubscribeAll(widget)
```

Computed and mapped values also use the registry path when they are observed through registry-backed observers. When a computed value no longer has downstream observers, registry cleanup cascades to its upstream subscriptions.

## Performance

The library has two subscription paths:

- Registry-backed subscriptions for lifecycle-owned observers and zero-allocation steady-state notification.
- Weak-pointer subscriptions for standalone observers.

Computed values cache their last result and recompute when dependencies mark them updated. Lists and maps notify observers on structural mutation.

Run benchmarks locally:

```sh
go test -bench=. -benchmem
```

Benchmark results below are from a local run and should be read as machine-specific.

## Thread Safety

`Registry`, `List`, and `Map` synchronize their internal state for concurrent use.

`SimpleValue` and `NonComparableValue` are lightweight value containers. If multiple goroutines write to the same value, synchronize those writes externally.

Iterator methods such as `All` and `PeekAll` hold a read lock while the iterator is consumed.
```

- [ ] **Step 2: Confirm README avoids internal extraction framing**

Run:

```sh
rg "extracted|File Structure|hybridset|Pool\\[|MobX.*vs|comparison" README.md
```

Expected: no matches for extraction framing, internal file structure, deep internals, or MobX comparison language. The command exits with status 1 because there are no matches.

- [ ] **Step 3: Confirm required README topics are present**

Run:

```sh
rg "Originally developed|MobX-style|go get github.com/gataengine/observable|Registry|weak references|go test -bench=. -benchmem|Thread Safety" README.md
```

Expected: matches are printed for each required topic.

- [ ] **Step 4: Commit README rewrite before benchmark numbers**

Run:

```sh
git add README.md
git commit -m "docs: rewrite observable readme"
```

Expected: commit succeeds and includes only `README.md`.

---

### Task 4: Run Benchmarks And Populate README Results

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Run tests before collecting benchmark numbers**

Run:

```sh
go test ./...
```

Expected: all tests pass.

- [ ] **Step 2: Capture Go and machine context**

Run:

```sh
go version
uname -sm
```

Expected: first command prints the Go toolchain version. Second command prints kernel and machine architecture. Use these exact values in the README sentence above the benchmark table.

- [ ] **Step 3: Run full benchmark suite**

Run:

```sh
go test -bench=. -benchmem ./...
```

Expected: benchmark command completes successfully. Save the terminal output for selecting README rows.

- [ ] **Step 4: Populate README benchmark table with measured rows**

Add a concise Markdown table under the benchmark context sentence in `README.md`.

Use three columns: `Benchmark`, `Time`, and `Allocations`.

Use the measured `ns/op` value for `Time`. Use the measured `B/op` and `allocs/op` values together for `Allocations`, for example `0 B/op, 0 allocs/op`.

Include these benchmark cases when they appear in the output:

- `BenchmarkValuePrimitive/registry_get`
- `BenchmarkValuePrimitive/weak_pointer_get`
- `BenchmarkValuePrimitive/registry_observe_getter`
- `BenchmarkNotify/registry_10_observers`
- `BenchmarkNotify/weak_pointer_10_observers`
- `BenchmarkComputedChain/registry_depth_3`
- `BenchmarkComputedChain/weak_pointer_depth_3`
- `BenchmarkListPrimitive/registry_set`
- `BenchmarkMapPrimitive/registry_set`
- `BenchmarkUIScenario/registry_widget_with_3_deps`

Also replace the sentence before the table so it names the Go version printed by `go version` and the platform printed by `uname -sm`.

Do not use approximate values. If a benchmark row name differs because of Go's full package output format, use the readable benchmark name without the CPU suffix.

- [ ] **Step 5: Verify README contains measured benchmark values**

Run:

```sh
rg "benchmark instructions|local output|table values" README.md
```

Expected: no matches. The command exits with status 1 because there are no benchmark instructions or temporary values left in README.

- [ ] **Step 6: Commit measured benchmark table**

Run:

```sh
git add README.md
git commit -m "docs: add observable benchmark results"
```

Expected: commit succeeds and includes only `README.md`.

---

### Task 5: Final Verification

**Files:**
- Inspect: `README.md`
- Inspect: `benchmark_test.go`
- Inspect: exported comments in package files

- [ ] **Step 1: Run full tests**

Run:

```sh
go test ./...
```

Expected: all tests pass.

- [ ] **Step 2: Run full benchmarks**

Run:

```sh
go test -bench=. -benchmem ./...
```

Expected: benchmark command completes successfully.

- [ ] **Step 3: Verify existing benchmark scenarios remain**

Run:

```sh
go test -run '^$' -bench=. -list '^Benchmark' ./... | rg "BenchmarkUIScenario|BenchmarkSubscribe|BenchmarkNotify|BenchmarkGetSet|BenchmarkComputedChain|BenchmarkCleanup|BenchmarkMemoryAlloc"
```

Expected: output contains each listed benchmark group.

- [ ] **Step 4: Check README standalone framing**

Run:

```sh
rg "standalone package|Originally developed|MobX-style|weak references|Registry|Thread Safety" README.md
```

Expected: output contains all listed concepts.

- [ ] **Step 5: Check final git state**

Run:

```sh
git status --short
```

Expected: no uncommitted changes in the `observable` repository.
