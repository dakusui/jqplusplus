package utils

func Insert[T any](slice []T, index int, value T) []T {
	if index < 0 || index > len(slice) {
		panic("Index out of range")
	}
	result := append(slice[:index], append([]T{value}, slice[index:]...)...)
	return result
}
