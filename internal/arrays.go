package internal

import (
	"strconv"
	"strings"
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

func DropLast[T any](in []T) []T {
	if len(in) == 0 {
		return nil
	}
	out := make([]T, len(in)-1)
	copy(out, in[:len(in)-1])
	return out
}

func lastElementIsOneOf(v ...string) func(p []any) bool {
	return func(p []any) bool {
		if len(p) == 0 {
			return false
		}
		s, ok := p[len(p)-1].(string)
		if !ok {
			return false
		}
		for _, v := range v {
			if s == v {
				return true
			}
		}
		return false
	}
}

func pathKey(p []any) string {
	var b strings.Builder
	for _, v := range p {
		switch x := v.(type) {
		case string:
			b.WriteString("s:")
			b.WriteString(x)
		case int:
			b.WriteString("i:")
			b.WriteString(strconv.Itoa(x))
		default:
			panic("unsupported type in path")
		}
		b.WriteByte('|')
	}
	return b.String()
}

// ToAnySlice converts a []T into a []any by copying elements.
func ToAnySlice[T any](xs []T) []any {
	out := make([]any, len(xs))
	for i, v := range xs {
		out[i] = v
	}
	return out
}
