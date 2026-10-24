package dto

import "context"

type ContextKey string

const (
	ContextPrefixStackqlRequest string = "stackql/request/"
)

const (
	ContextKeyCreationDate ContextKey = "stackql/request/created_date"
)

func ExtractStackqlRequestContextValue[T any](ctx context.Context, key ContextKey) (T, bool) {
	val := ctx.Value(key)
	if val == nil {
		var zero T
		return zero, false
	}
	typedVal, ok := val.(T)
	return typedVal, ok
}
