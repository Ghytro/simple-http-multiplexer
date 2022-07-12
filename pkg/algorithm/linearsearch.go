package algorithm

func Find[T comparable](x []T, value T) int {
	for i, el := range x {
		if el == value {
			return i
		}
	}
	return -1
}

type FindIfCallback[T any] func(elem T) bool

func FindIf[T any](x []T, c FindIfCallback[T]) int {
	for i, el := range x {
		if c(el) {
			return i
		}
	}
	return -1
}
