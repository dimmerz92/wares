package wares

import "net/http"

type ServeMux struct {
	*http.ServeMux
}

func NewServeMux() *ServeMux {
	return &ServeMux{http.NewServeMux()}
}

func (m *ServeMux) Handle(pattern string, handler http.Handler, middleware ...MiddlewareFunc) {
	m.ServeMux.Handle(pattern, Chain(handler, middleware...))
}

func (m *ServeMux) HandleFunc(pattern string, handlerFunc http.HandlerFunc, middleware ...MiddlewareFunc) {
	m.ServeMux.Handle(pattern, Chain(handlerFunc, middleware...))
}
