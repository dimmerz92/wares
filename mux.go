package wares

import (
	"fmt"
	"net/http"
	"strings"
)

// ServeMux is a thin wrapper over the http.ServeMux that extends it with grouping and middleware functionality.
type ServeMux struct {
	*http.ServeMux
	wrapped http.Handler
}

// NewServeMux returns a new instance of a ServeMux.
func NewServeMux() *ServeMux {
	mux := http.NewServeMux()
	return &ServeMux{
		ServeMux: mux,
		wrapped:  mux,
	}
}

// Use registers middleware with the ServeMux in the given stack order.
func (m *ServeMux) Use(middleware ...MiddlewareFunc) {
	m.wrapped = Chain(m.wrapped, middleware...)
}

// Group returns a new ServeMux scoped to the given pattern.
func (m *ServeMux) Group(pattern string, middleware ...MiddlewareFunc) *ServeMux {
	var (
		method string
		route  string
	)

	switch parts := strings.Split(pattern, " "); len(parts) {
	case 2:
		method = parts[0]
		route = parts[1]

	case 1:
		route = parts[0]

	default:
		panic(fmt.Sprintf("invalid pattern: %s", pattern))
	}

	if !strings.HasSuffix(route, "/") {
		route += "/"
	}

	mux := NewServeMux()
	m.ServeMux.Handle(
		strings.Trim(method+" "+route, " "),
		http.StripPrefix(strings.TrimSuffix(route, "/"), Chain(mux, middleware...)),
	)
	return mux
}

// Handle registers the handler for the given pattern.
// Optional middleware are applied in the given stack order.
// If the given pattern conflicts with one that is already registered or if the pattern is invalid, Handle panics.
//
// See [ServeMux](https://pkg.go.dev/net/http#ServeMux) for details on valid patterns and conflict rules.
func (m *ServeMux) Handle(pattern string, handler http.Handler, middleware ...MiddlewareFunc) {
	m.ServeMux.Handle(pattern, Chain(handler, middleware...))
}

// HandleFunc registers the handler function for the given pattern.
// Optional middleware are applied in the given stack order.
// If the given pattern conflicts with one that is already registered or if the pattern is invalid, HandleFunc panics.
//
// See [ServeMux](https://pkg.go.dev/net/http#ServeMux) for details on valid patterns and conflict rules.
func (m *ServeMux) HandleFunc(pattern string, handlerFunc http.HandlerFunc, middleware ...MiddlewareFunc) {
	m.ServeMux.Handle(pattern, Chain(handlerFunc, middleware...))
}

// ServeHTTP dispatches the request to the handler whose pattern most closely matches the request URL.
func (m *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.wrapped.ServeHTTP(w, r)
}
