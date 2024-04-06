package lib

func IfNil[T any](value *T, fallback T) T {
	if value == nil {
		return fallback
	}
	return *value
}
