# Observable Independent Library Cleanup Design

## Goal

Make `github.com/gataengine/observable` read and behave like an independent Go library while preserving the current public API and existing benchmark scenarios.

The work is documentation and benchmark focused. It does not rename public interfaces, change behavior, or remove existing benchmarks.

## Decisions

- Keep existing interface names, including `ROValue`, `ROList`, `ROMap`, `RegistryProvider`, and `DependentObservable`.
- Rewrite the README as standalone library documentation.
- Keep a short provenance note that the library was originally developed for Gata Engine UI.
- Mention MobX only as inspiration, not as a feature comparison or compatibility claim.
- Keep concise performance and architecture documentation in the README.
- Remove README emphasis on file layout and low-level internal implementation details.
- Update exported Go comments across the package so `go doc` reads like public library documentation.
- Preserve all existing benchmark scenarios and names. New scenarios may be added, but existing scenarios must not be removed or renamed.

## README Design

The README should present `observable` as a standalone reactive primitives library for Go.

It should include:

- Package identity and install command.
- Brief provenance note: originally developed for Gata Engine UI.
- Brief inspiration note: inspired by MobX-style observable/computed dependency tracking, implemented as idiomatic Go primitives.
- Quick start with `Simple`, `BasicObserver`, `Observe`, `Get`, `Set`, and update detection.
- Core concepts:
  - observable values
  - observers
  - computed values
  - observable lists and maps
  - `Peek` versus subscribed reads
  - `Observe` getters for repeated reads
- Subscription modes:
  - weak-pointer standalone subscriptions
  - optional `Registry` lifecycle management
- Concise performance and architecture notes:
  - registry path
  - weak-pointer path
  - computed dependency tracking
  - cleanup behavior
- Benchmark command and a curated benchmark table from a fresh local run, including Go version and machine context.
- Thread-safety notes based on the current implementation.

The README should avoid:

- Presenting the module as an extracted internal package.
- Deep `hybridset` or pool mechanics.
- File-structure walkthroughs.
- MobX comparison tables.

## Go Doc Design

Update exported comments without changing names or behavior.

The comments should consistently explain:

- `Get(obs)` reads and subscribes the observer.
- `Peek()` reads without subscribing.
- `Observe(obs)` subscribes once and returns a getter for repeated reads.
- `Registry` is optional lifecycle-owned subscription management.
- weak-pointer subscriptions are the default standalone path.
- computed and mapped values can subscribe to upstream observables and notify downstream observers.

Likely files:

- `interfaces.go`
- `observer.go`
- `registry.go`
- `simple.go`
- `comparable.go`
- `computed.go`
- `computed_list.go`
- `list.go`
- `map.go`
- `mapped.go`
- `bidi_mapped.go`
- `static.go`

## Benchmark Design

Keep all existing benchmark groups and scenario names, including `BenchmarkUIScenario`.

Add primitive-focused coverage where useful, with both registry and weak-pointer paths when subscriptions are involved:

- value read/write/observe
- notification fanout for 1, 10, and 100 observers
- computed recomputation
- list mutation/read
- map mutation/read
- allocation-sensitive benchmarks using `b.ReportAllocs()`

README benchmark reporting should show a concise curated subset:

- primitive baseline results
- registry versus weak-pointer results where meaningful
- at least one UI scenario result, preserving the library's original performance context

## Testing And Verification

Run:

```sh
go test ./...
go test -bench=. -benchmem
```

Use the benchmark output to populate the README table. If the environment cannot run benchmarks reliably, document that limitation instead of inventing numbers.

## Out Of Scope

- Public API renames.
- Behavior changes.
- Removing or renaming existing benchmark scenarios.
- Reworking internals solely for documentation polish.
- A MobX compatibility layer or detailed MobX comparison.
