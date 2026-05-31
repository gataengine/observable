package observable

import (
	"strings"
	"testing"
)

func newTestSortedView(items []string, col string, desc bool) (*List[string], ROList[string], *SimpleValue[SortConfig]) {
	list := NewList[string]()
	list.Add(items...)

	sortConfig := Simple(SortConfig{Column: col, Desc: desc})

	comparators := map[string]func(a, b string) int{
		"alpha": strings.Compare,
	}

	sv := NewSortedListView(list, sortConfig, comparators)
	return list, sv, sortConfig
}

func collectValues[T any](sv ROList[T]) []T {
	var vals []T
	for _, v := range sv.PeekAll() {
		vals = append(vals, v)
	}
	return vals
}

func TestSortedListView_SortsByColumn(t *testing.T) {
	_, sv, _ := newTestSortedView([]string{"cherry", "apple", "banana"}, "alpha", false)

	vals := collectValues(sv)
	if len(vals) != 3 || vals[0] != "apple" || vals[1] != "banana" || vals[2] != "cherry" {
		t.Errorf("expected sorted [apple banana cherry], got %v", vals)
	}
}

func TestSortedListView_DescOrder(t *testing.T) {
	_, sv, _ := newTestSortedView([]string{"cherry", "apple", "banana"}, "alpha", true)

	vals := collectValues(sv)
	if len(vals) != 3 || vals[0] != "cherry" || vals[1] != "banana" || vals[2] != "apple" {
		t.Errorf("expected desc sorted [cherry banana apple], got %v", vals)
	}
}

func TestSortedListView_EmptySortPreservesOrder(t *testing.T) {
	_, sv, _ := newTestSortedView([]string{"cherry", "apple", "banana"}, "", false)

	vals := collectValues(sv)
	if len(vals) != 3 || vals[0] != "cherry" || vals[1] != "apple" || vals[2] != "banana" {
		t.Errorf("expected source order [cherry apple banana], got %v", vals)
	}
}

func TestSortedListView_ResortsOnSourceMutation(t *testing.T) {
	list, sv, _ := newTestSortedView([]string{"cherry", "apple"}, "alpha", false)

	list.Add("banana")

	vals := collectValues(sv)
	if len(vals) != 3 || vals[0] != "apple" || vals[1] != "banana" || vals[2] != "cherry" {
		t.Errorf("expected [apple banana cherry] after add, got %v", vals)
	}
}

func TestSortedListView_ResortsOnSortChange(t *testing.T) {
	_, sv, sortConfig := newTestSortedView([]string{"cherry", "apple", "banana"}, "", false)

	vals := collectValues(sv)
	if vals[0] != "cherry" {
		t.Errorf("expected source order first, got %v", vals)
	}

	sortConfig.Set(SortConfig{Column: "alpha", Desc: false})

	vals = collectValues(sv)
	if vals[0] != "apple" {
		t.Errorf("expected sorted after config change, got %v", vals)
	}
}

func TestSortedListView_Observable(t *testing.T) {
	list, sv, _ := newTestSortedView([]string{"cherry", "apple"}, "alpha", false)

	var obs BasicObserver
	_ = sv.Observe(&obs)
	obs.GetAndResetUpdated()

	list.Add("banana")
	if !obs.IsUpdated() {
		t.Error("expected observer notified after source mutation")
	}
}
