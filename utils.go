package wares

// Coalesce returns the first non-zero value for type T if one exists, otherwise the zero value is returned.
func Coalesce[T comparable](values ...T) T {
	var zero T
	for _, value := range values {
		if value != zero {
			return value
		}
	}
	return zero
}

// IIF is an inline if that returns v1 if the condition is true, otherwise v2 is returned.
func IIF[T any](condition bool, v1, v2 T) T {
	if condition {
		return v1
	}
	return v2
}
