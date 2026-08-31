package compare

func Default[T comparable](value T, defaultValue T) T {
	if IsZero(value) {
		return defaultValue
	}
	return value
}

func IsZero[T comparable](v T) bool {
	var z T
	return v == z
}
