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

Observable reads are also subscription points:

- `Get(obs)` returns the current value and subscribes `obs` to future changes.
- `Peek()` returns the current value without subscribing.
- `Observe(obs)` subscribes once and returns a getter for repeated reads without resubscribing.

`BasicObserver` is the smallest observer implementation. It records whether any subscribed source changed, exposes `IsUpdated`, and lets callers consume the flag with `GetAndResetUpdated`.

Computed values track dependencies through the observer passed into the compute function. When a dependency changes, the computed value is marked dirty, notifies its observers, and recomputes on the next read.

```go
first := observable.Simple("Ada")
last := observable.Simple("Lovelace")

fullName := observable.NewComputed(func(obs observable.Observer) string {
	return first.Get(obs) + " " + last.Get(obs)
})

var obs observable.BasicObserver
getter := fullName.Observe(&obs)

fmt.Println(getter.Get()) // Ada Lovelace

last.Set("Byron")

if obs.GetAndResetUpdated() {
	fmt.Println(getter.Get()) // Ada Byron
}
```

## Values

Use `Simple` for comparable values. It skips notifications when the new value equals the old value.

```go
count := observable.Simple(1)

var obs observable.BasicObserver
getter := count.Observe(&obs)

count.Update(func(v *int) {
	*v = *v + 1
})

fmt.Println(getter.Get()) // 2
```

Use `SimpleNonComparable` for slices, maps, functions, and other values that cannot be compared with `==`. It notifies observers whenever `Set`, `Update`, or a successful `MaybeUpdate` runs.

```go
items := observable.SimpleNonComparable([]string{"a"})

var obs observable.BasicObserver
getter := items.Observe(&obs)

items.Update(func(v *[]string) {
	*v = append(*v, "b")
})

fmt.Println(getter.Get()) // [a b]
```

## Lists

`NewList` creates an observable list with stable item keys. `Add`, `Set`, and other content-changing mutations notify observers. `Move` preserves the moved item's key.

```go
todos := observable.NewList[string]()
todos.Add("write docs", "ship package")

var obs observable.BasicObserver
view := todos.Observe(&obs)

todos.Set(0, "review docs")
todos.Move(1, 0)

if obs.GetAndResetUpdated() {
	for key, value := range view.All() {
		fmt.Println(key, value)
	}
}
```

## Maps

`NewMap` creates an observable map. `Set`, `Delete`, `Merge`, `Replace`, and `Clear` notify observers.

```go
scores := observable.NewMap[string, int]()
scores.Set("alice", 10)

var obs observable.BasicObserver
view := scores.Observe(&obs)

scores.Set("alice", 11)

if obs.GetAndResetUpdated() {
	if score, ok := view.Get("alice"); ok {
		fmt.Println(score)
	}
}
```

## Subscription Modes

Standalone observers use weak references by default. This keeps simple use cases lightweight: observe values with a `BasicObserver`, then let normal Go ownership decide when the observer disappears.

For objects with explicit lifecycles, use a `Registry`. Any observer that implements `ObservableRegistry` and `CurrentObserver` routes subscriptions through that registry, which can drain dirty observers and unsubscribe all subscriptions owned by an observer.

```go
type Widget struct {
	observable.BasicObserver
	registry *observable.Registry
}

func NewWidget(registry *observable.Registry) *Widget {
	return &Widget{registry: registry}
}

func (w *Widget) ObservableRegistry() *observable.Registry {
	return w.registry
}

func (w *Widget) CurrentObserver() observable.Observer {
	return w
}

func main() {
	registry := observable.NewRegistry()
	widget := NewWidget(registry)
	name := observable.Simple("Alice")

	getter := name.Observe(widget)
	name.Set("Bob")

	for _, dirty := range registry.DrainDirty() {
		if dirty == widget {
			fmt.Println(getter.Get())
		}
	}

	registry.UnsubscribeAll(widget)
}
```

Computed and mapped values also participate in registry cleanup. When a registry-owned observer is unsubscribed, orphaned computed or mapped dependencies are cleaned up recursively.

## Performance

Observable supports both registry-backed subscriptions and standalone weak-pointer subscriptions. Registry-backed observers make lifecycle cleanup explicit and give applications a central dirty queue. Standalone observers keep one-off use cases simple without requiring a registry.

Computed values cache their last result and recompute only after a dependency marks them dirty. List and map mutations notify observers after the internal mutation is complete.

Benchmarks below were measured with `go version go1.26.0 darwin/arm64` on `Darwin arm64`.

| Benchmark | Time | Allocations |
| --- | --- | --- |
| BenchmarkValuePrimitive/registry_get | 76.79 ns/op | 0 B/op, 0 allocs/op |
| BenchmarkValuePrimitive/weak_pointer_get | 32.70 ns/op | 0 B/op, 0 allocs/op |
| BenchmarkValuePrimitive/registry_observe_getter | 2.112 ns/op | 0 B/op, 0 allocs/op |
| BenchmarkNotify/registry_10_observers | 59.28 ns/op | 0 B/op, 0 allocs/op |
| BenchmarkNotify/weak_pointer_10_observers | 303.7 ns/op | 144 B/op, 6 allocs/op |
| BenchmarkComputedChain/registry_depth_3 | 523.1 ns/op | 0 B/op, 0 allocs/op |
| BenchmarkComputedChain/weak_pointer_depth_3 | 404.5 ns/op | 224 B/op, 12 allocs/op |
| BenchmarkListPrimitive/registry_set | 59.00 ns/op | 0 B/op, 0 allocs/op |
| BenchmarkMapPrimitive/registry_set | 60.96 ns/op | 0 B/op, 0 allocs/op |
| BenchmarkUIScenario/registry_widget_with_3_deps | 494.1 ns/op | 28 B/op, 3 allocs/op |

## Thread Safety

`Registry`, `List`, and `Map` synchronize their internal state. `SimpleValue` and `NonComparableValue` writes should be externally synchronized if multiple goroutines write the same value. `All` and `PeekAll` iterators on lists and maps hold a read lock while the iterator is consumed.
