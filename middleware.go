package wares

import "net/http"

type MiddlewareFunc func(next http.Handler) http.Handler

// Chain applies the given middlewares to the handler in the specified stack order.
func Chain(handler http.Handler, middleware ...MiddlewareFunc) http.Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}
	return handler
}
