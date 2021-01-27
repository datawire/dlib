package dhttp

import (
	"context"
	"net/http"
)

func WithTestHook(ctx context.Context, hook func(http.Handler) http.Handler) context.Context {
	return context.WithValue(ctx, testHookContextKey{}, hook)
}
