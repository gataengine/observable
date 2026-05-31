package observable

import "slices"

// SortConfig provides column key and direction for sorting.
type SortConfig struct {
	Column string
	Desc   bool
}

// NewSortedListView creates a sorted projection of the given list.
// Returns a ComputedList that re-sorts when the source or sort config changes.
func NewSortedListView[T any](
	source ROList[T],
	sortConfig *SimpleValue[SortConfig],
	comparators map[string]func(a, b T) int,
) ROList[T] {
	return NewComputedList(func(obs Observer) []ComputedListItem[int64, T] {
		cfg := sortConfig.Get(obs)
		getter := source.Observe(obs)

		items := make([]ComputedListItem[int64, T], getter.Len())
		for i := range items {
			key, val, _ := getter.At(i)
			items[i] = ComputedListItem[int64, T]{Key: key, Value: val}
		}

		cmp, hasCmp := comparators[cfg.Column]
		if cfg.Column != "" && hasCmp {
			slices.SortStableFunc(items, func(a, b ComputedListItem[int64, T]) int {
				result := cmp(a.Value, b.Value)
				if cfg.Desc {
					result = -result
				}
				return result
			})
		}

		return items
	})
}
