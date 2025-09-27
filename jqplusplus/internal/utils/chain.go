package utils

type Result[T any] struct {
	V   T
	Err error
}

func Then[A, B any](r Result[A], f func(A) (B, error)) Result[B] {
	if r.Err != nil {
		return Result[B]{Err: r.Err}
	}
	v, err := f(r.V)
	return Result[B]{V: v, Err: err}
}

func Ok[T any](v T) Result[T] { return Result[T]{V: v} }
