package observable

import (
	"sync/atomic"
)

// observerState is the concrete observer implementation.
type observerState struct {
	clean    atomic.Bool // inverted from updated to be correct without init
	OnChange func()
}

func (b *observerState) MarkUpdated() {
	b.clean.Store(false)
	if b.OnChange != nil {
		b.OnChange()
	}
}

// Noop can be passed to Get when the caller wants the subscribed-read API
// without creating a subscription.
var Noop Observer = &noopObserver{}

type noopObserver struct {
	observerState
}

func (n *noopObserver) GetObserver() *observerState { return nil }
func (n *noopObserver) MarkUpdated()                {}

// BasicObserver is a minimal Observer. Embed it or allocate it directly to know
// whether a subscribed source changed.
type BasicObserver struct {
	observerState
}

func (b *BasicObserver) GetObserver() *observerState {
	return &b.observerState
}

func (b *BasicObserver) IsUpdated() bool {
	return !b.clean.Load()
}

func (b *BasicObserver) GetAndResetUpdated() bool {
	return !b.clean.Swap(true)
}
