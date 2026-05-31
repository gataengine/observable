package observable

// staticProbe detects whether any non-static observable attempted subscription.
// Static.Get() never calls maybeAddObserver, so GetObserver() is never invoked.
// Non-static observables call maybeAddObserver → GetObserver(), setting subscribed.
type staticProbe struct {
	subscribed bool
}

func (p *staticProbe) GetObserver() *observerState {
	p.subscribed = true
	return nil
}

func (p *staticProbe) MarkUpdated() {}

// isProbe is a marker method so ComputedValue.Get can detect probe mode and skip computation.
func (p *staticProbe) isProbe() {}
