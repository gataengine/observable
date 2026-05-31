package observable

// GetOrCreate returns v if non-nil, otherwise creates a NonComparableValue[T]
// initialized with the first element of defaultVal (or the zero value of T).
func GetOrCreate[T any](v Value[T], defaultVal ...T) Value[T] {
	if v != nil {
		return v
	}
	var zero T
	if len(defaultVal) > 0 {
		zero = defaultVal[0]
	}
	return SimpleNonComparable(zero)
}
