package observable

import (
	"strconv"
	"testing"
)

func floatFormat(v float64) string {
	return strconv.FormatFloat(v, 'f', 3, 64)
}

func floatParse(s string) (float64, bool) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func TestBidiMapped_GetReturnsForwardWhenNoCache(t *testing.T) {
	source := Simple(56.3)
	bidi := NewBidiMapped(source, floatFormat, floatParse)

	result := bidi.Get(Noop)
	if result != "56.300" {
		t.Fatalf("expected \"56.300\", got %q", result)
	}
}

func TestBidiMapped_SetCachesAndPropagatesOnSuccess(t *testing.T) {
	source := Simple(0.0)
	bidi := NewBidiMapped(source, floatFormat, floatParse)

	bidi.Set("42.5")

	// Source should be updated.
	if source.Get(Noop) != 42.5 {
		t.Fatalf("expected source=42.5, got %f", source.Get(Noop))
	}

	// Get should return cached value, not forward(source).
	if bidi.Get(Noop) != "42.5" {
		t.Fatalf("expected cached \"42.5\", got %q", bidi.Get(Noop))
	}

	// Valid should be true.
	if !bidi.Valid().Get(Noop) {
		t.Fatal("expected valid=true after successful parse")
	}
}

func TestBidiMapped_SetCachesButDoesNotPropagateOnFailure(t *testing.T) {
	source := Simple(56.0)
	bidi := NewBidiMapped(source, floatFormat, floatParse)

	bidi.Set("abc")

	// Source should be unchanged.
	if source.Get(Noop) != 56.0 {
		t.Fatalf("expected source=56.0, got %f", source.Get(Noop))
	}

	// Get should return the cached value.
	if bidi.Get(Noop) != "abc" {
		t.Fatalf("expected cached \"abc\", got %q", bidi.Get(Noop))
	}

	// Valid should be false.
	if bidi.Valid().Get(Noop) {
		t.Fatal("expected valid=false after failed parse")
	}
}

func TestBidiMapped_ExternalSourceChangeClearsCache(t *testing.T) {
	reg := NewRegistry()
	source := Simple(56.0)
	bidi := NewBidiMapped(source, floatFormat, floatParse)

	// Subscribe through registry so source change notifications work.
	obs := &testRegistryObserver{reg: reg}
	bidi.Get(obs)

	// Set a cached value (fails to parse).
	bidi.Set("abc")
	if bidi.Get(obs) != "abc" {
		t.Fatalf("expected cached \"abc\", got %q", bidi.Get(obs))
	}

	// External change to source.
	source.Set(72.0)

	// Cache should be cleared, Get returns forward(72).
	result := bidi.Get(obs)
	if result != "72.000" {
		t.Fatalf("expected \"72.000\" after external source change, got %q", result)
	}

	// Valid should reset to true.
	if !bidi.Valid().Get(Noop) {
		t.Fatal("expected valid=true after external source change")
	}
}

func TestBidiMapped_SelfWriteDoesNotClearCache(t *testing.T) {
	reg := NewRegistry()
	source := Simple(0.0)
	bidi := NewBidiMapped(source, floatFormat, floatParse)

	obs := &testRegistryObserver{reg: reg}
	bidi.Get(obs)

	// Set a valid value — this writes to source, but cache should persist.
	bidi.Set("42.5")

	// Cache should still be "42.5", not reformatted to "42.500".
	if bidi.Get(obs) != "42.5" {
		t.Fatalf("expected cached \"42.5\", got %q", bidi.Get(obs))
	}
}

func TestBidiMapped_ClearCacheResetsToForward(t *testing.T) {
	source := Simple(56.3)
	bidi := NewBidiMapped(source, floatFormat, floatParse)

	// Set a cached value.
	bidi.Set("56.3")
	if bidi.Get(Noop) != "56.3" {
		t.Fatalf("expected cached \"56.3\", got %q", bidi.Get(Noop))
	}

	// Clear cache.
	bidi.ClearCache()

	// Get should return formatted value.
	if bidi.Get(Noop) != "56.300" {
		t.Fatalf("expected \"56.300\" after ClearCache, got %q", bidi.Get(Noop))
	}
}

func TestBidiMapped_ClearCacheResetsValidOnInvalid(t *testing.T) {
	source := Simple(56.0)
	bidi := NewBidiMapped(source, floatFormat, floatParse)

	// Set invalid.
	bidi.Set("abc")
	if bidi.Valid().Get(Noop) {
		t.Fatal("expected valid=false after invalid Set")
	}

	// ClearCache resets.
	bidi.ClearCache()
	if !bidi.Valid().Get(Noop) {
		t.Fatal("expected valid=true after ClearCache")
	}
}

func TestBidiMapped_ValidTransitions(t *testing.T) {
	source := Simple(0.0)
	bidi := NewBidiMapped(source, floatFormat, floatParse)

	// Initial: valid.
	if !bidi.Valid().Get(Noop) {
		t.Fatal("expected valid=true initially")
	}

	// Failed parse: invalid.
	bidi.Set("abc")
	if bidi.Valid().Get(Noop) {
		t.Fatal("expected valid=false after failed parse")
	}

	// Successful parse: valid again.
	bidi.Set("42")
	if !bidi.Valid().Get(Noop) {
		t.Fatal("expected valid=true after successful parse")
	}

	// Failed parse again.
	bidi.Set("-")
	if bidi.Valid().Get(Noop) {
		t.Fatal("expected valid=false after second failed parse")
	}
}

func TestBidiMapped_ObserveReturnsGetter(t *testing.T) {
	source := Simple(10.0)
	bidi := NewBidiMapped(source, floatFormat, floatParse)

	reg := NewRegistry()
	obs := &testRegistryObserver{reg: reg}
	getter := bidi.Observe(obs)

	if getter.Get() != "10.000" {
		t.Fatalf("expected \"10.000\", got %q", getter.Get())
	}

	bidi.Set("20")
	if getter.Get() != "20" {
		t.Fatalf("expected cached \"20\", got %q", getter.Get())
	}

	bidi.ClearCache()
	if getter.Get() != "20.000" {
		t.Fatalf("expected \"20.000\" after ClearCache, got %q", getter.Get())
	}
}

func TestBidiMapped_SourceSameValueKeepsCache(t *testing.T) {
	source := Simple(56.0)
	bidi := NewBidiMapped(source, floatFormat, floatParse)

	// Set an invalid value. Source stays at 56.
	bidi.Set("56.")

	// Setting source to the same value shouldn't trigger a change notification,
	// but even if it does, our selfWrite guard handles it.
	// The cache should remain.
	if bidi.Get(Noop) != "56." {
		t.Fatalf("expected cached \"56.\", got %q", bidi.Get(Noop))
	}
}
