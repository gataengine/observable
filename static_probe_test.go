package observable

import "testing"

func TestStaticProbe_StaticDoesNotSubscribe(t *testing.T) {
	probe := &staticProbe{}
	s := Static(42)
	_ = s.Get(probe)
	if probe.subscribed {
		t.Fatal("static value should not trigger subscription")
	}
}

func TestStaticProbe_SimpleSubscribes(t *testing.T) {
	probe := &staticProbe{}
	s := Simple(42)
	_ = s.Get(probe)
	if !probe.subscribed {
		t.Fatal("simple value should trigger subscription")
	}
}

func TestStaticProbe_ComputedSubscribes(t *testing.T) {
	probe := &staticProbe{}
	src := Simple(1)
	c := NewComputed(func(obs Observer) int { return src.Get(obs) * 2 })
	_ = c.Get(probe)
	if !probe.subscribed {
		t.Fatal("computed value should trigger subscription")
	}
}
