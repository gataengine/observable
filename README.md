# Observable

A standalone reactive primitives library for Go with zero-allocation notification paths.

## Status

- This module was extracted from `gataengine/ui/observable`.
- It is the active observable system used by Gata Engine UI code.

## Overview

This package provides reactive values (`SimpleValue`, `List`, `Map`) and computed derivations (`ComputedValue`) that automatically track dependencies and notify observers when values change.

Two subscription mechanisms are available:
- **Registry-based**: Centralized subscription management with zero-allocation notifications
- **Weak pointer-based**: Per-observable subscriptions with automatic cleanup

## Quick Start

```go
// Create observable values
name := observable.Simple("Alice")
age := observable.Simple(30)

// Create an observer
var obs observable.BasicObserver

// Observe values - returns a getter for efficient repeated access
nameGetter := name.Observe(&obs)
ageGetter := age.Observe(&obs)

fmt.Println(nameGetter.Get(), ageGetter.Get())  // "Alice 30"

// Update triggers notification
name.Set("Bob")
if obs.IsUpdated() {
    fmt.Println("Changed!")
    obs.GetAndResetUpdated()  // Reset the flag
}
```

## Core Types

### SimpleValue[T]

A mutable observable value.

```go
v := observable.Simple(42)
v.Set(100)

var obs observable.BasicObserver
getter := v.Observe(&obs)
val := getter.Get()  // 100

// In-place update
v.Update(func(n *int) { *n++ })
```

### List[T]

An observable slice with fine-grained change tracking.

```go
list := observable.NewList[string]()
list.Append("a", "b", "c")
list.Set(1, "B")
list.Remove(0)
```

### Map[K, V]

An observable map.

```go
m := observable.NewMap[string, int]()
m.Set("count", 1)
val, ok := m.Get("count")
m.Delete("count")
```

### ComputedValue[T]

A derived value that automatically tracks dependencies.

```go
a := observable.Simple(1)
b := observable.Simple(2)

sum := observable.NewComputed(func(obs observable.Observer) int {
    return a.Get(obs) + b.Get(obs)  // Dependencies auto-tracked
})

var myObs observable.BasicObserver
getter := sum.Observe(&myObs)
fmt.Println(getter.Get())  // 3

a.Set(10)
// myObs.IsUpdated() is now true
fmt.Println(getter.Get())  // 12
```

## Registry-based Subscriptions

For UI widgets with known lifecycles, use the Registry for zero-allocation notifications.

### Creating a Registry-aware Observer

Implement `RegistryProvider` to enable registry-based subscriptions:

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
```

### Using Registry

```go
reg := observable.NewRegistry()
widget := &Widget{registry: reg}

v := observable.Simple(42)

// When widget observes v, it auto-binds to the registry
getter := v.Observe(widget)

// Later, when widget is destroyed
reg.UnsubscribeAll(widget)
```

### Computed with Registry

`ComputedValue` implements `RegistryProvider`, so computed chains automatically use the registry:

```go
a := observable.Simple(1)

// When computed observes 'a', both bind to the same registry
sum := observable.NewComputed(func(obs observable.Observer) int {
    return a.Get(obs) * 2
})

// Observe with a registry-aware widget
getter := sum.Observe(widget)
```

## Performance

Zero-allocation notification paths in steady state:

| Operation | Observers | Time | Allocations |
|-----------|-----------|------|-------------|
| Notify | 1 | 43ns | 0 |
| Notify | 10 | 57ns | 0 |
| Notify | 100 | 755ns | 0 |

## Architecture

### hybridset

Internal data structure optimized for typical UI patterns:

- **Small mode** (< 32 items): slice + map for fast iteration
- **Large mode** (>= 32 items): upgrades to `xsync.MapOf` for concurrent access

### Pool[T]

Typed slice pool for zero-allocation iteration:

```go
pool := hybridset.NewPool[Observer](32)
ptr := pool.Get(size)
// use *ptr
pool.Put(ptr)
```

### Range/CopyTo Pattern

`Set.Range` returns `useCopyTo bool`:
- **xsync mode**: Iterates directly, returns `false`
- **small mode**: Returns `true` without iterating (use `CopyTo` with pool)

```go
useCopyTo := set.Range(func(item T) bool {
    process(item)
    return true
})
if useCopyTo {
    ptr := pool.Get(set.Size())
    set.CopyTo(*ptr)
    for _, item := range *ptr {
        process(item)
    }
    pool.Put(ptr)
}
```

## Thread Safety

- `Registry`: Safe for concurrent use (uses `sync.RWMutex`)
- `hybridset.Set`: Safe for concurrent use (small mode uses `RWMutex`, large mode uses `xsync.MapOf`)
- `SimpleValue`/`List`/`Map`: Safe for concurrent reads; writes should be synchronized externally

## File Structure

```
.
├── interfaces.go      # Observable, Observer, RegistryProvider interfaces
├── observer.go        # BasicObserver implementation
├── observable.go      # observableBase with dual subscription paths
├── registry.go        # Centralized subscription registry
├── simple.go          # SimpleValue[T] implementation
├── list.go            # List[T] implementation
├── map.go             # Map[K,V] implementation
├── computed.go        # ComputedValue[T] implementation
└── hybridset/
    ├── set.go         # Hybrid set (slice+map / xsync)
    └── pool.go        # Typed slice pool
```
