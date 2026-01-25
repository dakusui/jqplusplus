package internal

import (
	"fmt"
	"sort"
)

func Reverse[T any](s []T) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func Insert[T any](slice []T, index int, value T) []T {
	if index < 0 || index > len(slice) {
		panic("Index out of range")
	}
	result := append(slice[:index], append([]T{value}, slice[index:]...)...)
	return result
}

func Filter[T any](in []T, pred func(T) bool) []T {
	if len(in) == 0 {
		fmt.Printf("Filter:   in is Empty\n")
	} else {
		fmt.Printf("Filter:   in is NOT Empty\n")
	}

	out := make([]T, 0, len(in))
	for _, v := range in {
		if pred(v) {
			out = append(out, v)
		}
	}
	return out
}

func Map[T any, R any](in []T, f func(T) R) []R {
	out := make([]R, len(in))
	for i, v := range in {
		out[i] = f(v)
	}
	return out
}

func DistinctBy[T any, K comparable](in []T, key func(T) K) []T {
	seen := make(map[K]struct{}, len(in))
	out := make([]T, 0, len(in))

	for _, v := range in {
		k := key(v)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, v)
	}
	return out
}

type Ordered interface {
	int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | uintptr | float32 | float64 | string
}

func Sort[T any](in []T, comparator func(a, b T) bool) []T {

	out := make([]T, len(in))
	copy(out, in)
	sort.Slice(out, func(i, j int) bool {
		return comparator(out[i], out[j])
	})
	return out
}

func DropLast[T any](in []T) []T {
	if len(in) == 0 {
		return nil
	}
	out := make([]T, len(in)-1)
	copy(out, in[:len(in)-1])
	return out
}

// ToAnySlice converts a []T into a []any by copying elements.
func ToAnySlice[T any](xs []T) []any {
	out := make([]any, len(xs))
	for i, v := range xs {
		out[i] = v
	}
	return out
}

/*
PASSING:
=== RUN   TestLoadAndResolveInheritances_RefToStringFromInsideArray
Filter:   in is NOT Empty
Paths: paths is NOT empty
Filter:   in is NOT Empty
Paths: paths is NOT empty
Filter:   in is NOT Empty
Paths: paths is NOT empty
Filter:   in is NOT Empty
paths is NOT empty!!!!!!!!!!!!!!!!!!!!!!!!!!!
Paths: paths is NOT empty
Filter:   in is NOT Empty
paths is empty!!!!!!!!!!!!!!!!!!!!!!!!!!!
--- PASS: TestLoadAndResolveInheritances_RefToStringFromInsideArray (0.00s)
PASS
*/

/*
FAILING:
=== RUN   TestLoadAndResolveInheritances_RefToStringFromInsideArray
Filter:   in is NOT Empty
Paths: paths is NOT empty
Filter:   in is NOT Empty
Paths: paths is NOT empty
Filter:   in is NOT Empty
Paths: paths is NOT empty
Filter:   in is NOT Empty
paths is empty!!!!!!!!!!!!!!!!!!!!!!!!!!!

*/
