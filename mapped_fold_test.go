package observable

import (
	"testing"
)

func TestMappedStaticFolding_StaticSource(t *testing.T) {
	src := Static(10)
	result := Mapped(src, func(v int) string {
		return "v=" + itoa(v)
	})
	if result.Get(Noop) != "v=10" {
		t.Fatalf("expected v=10, got %s", result.Get(Noop))
	}
	if _, ok := result.(staticValue[string]); !ok {
		t.Fatal("expected staticValue when source is static")
	}
}

func TestMappedStaticFolding_DynamicSource(t *testing.T) {
	src := Simple(10)
	result := Mapped(src, func(v int) string {
		return "v=" + itoa(v)
	})
	if result.Get(Noop) != "v=10" {
		t.Fatalf("expected v=10, got %s", result.Get(Noop))
	}
	if _, ok := result.(staticValue[string]); ok {
		t.Fatal("expected MappedValue when source is dynamic")
	}
}
