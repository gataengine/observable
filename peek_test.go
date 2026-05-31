package observable

import "testing"

func TestSimpleValuePeek(t *testing.T) {
	v := Simple(42)
	if v.Peek() != 42 {
		t.Fatalf("expected 42, got %d", v.Peek())
	}
	v.Set(99)
	if v.Peek() != 99 {
		t.Fatalf("expected 99, got %d", v.Peek())
	}
}

func TestStaticPeek(t *testing.T) {
	v := Static("hello")
	if v.Peek() != "hello" {
		t.Fatalf("expected hello, got %s", v.Peek())
	}
}

func TestComputedPeek(t *testing.T) {
	src := Simple(10)
	c := NewComputed(func(obs Observer) int {
		return src.Get(obs) * 2
	})
	_ = c.Get(Noop)
	if c.Peek() != 20 {
		t.Fatalf("expected 20, got %d", c.Peek())
	}
	src.Set(5)
	if c.Peek() != 10 {
		t.Fatalf("expected 10, got %d", c.Peek())
	}
}

func TestMappedPeek(t *testing.T) {
	src := Simple(3)
	m := Mapped[int, string](src, func(v int) string {
		return string(rune('A' + v))
	})
	_ = m.Get(Noop)
	if m.Peek() != "D" {
		t.Fatalf("expected D, got %s", m.Peek())
	}
}

func TestBidiMappedPeek(t *testing.T) {
	src := Simple(5.0)
	b := NewBidiMapped(src, func(f float64) string {
		return "v"
	}, func(s string) (float64, bool) {
		return 0, false
	})
	_ = b.Get(Noop)
	result := b.Peek()
	if result != "v" {
		t.Fatalf("expected v, got %s", result)
	}
	b.Set("cached")
	if b.Peek() != "cached" {
		t.Fatalf("expected cached, got %s", b.Peek())
	}
}

func TestROValueInterfacePeek(t *testing.T) {
	var v ROValue[int] = Simple(1)
	if v.Peek() != 1 {
		t.Fatalf("expected 1, got %d", v.Peek())
	}
}
