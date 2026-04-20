package wares_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dimmerz92/wares"
)

func TestServeMux(t *testing.T) {
	h := func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("h")) }
	m := func(value string) wares.MiddlewareFunc {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(value))
				next.ServeHTTP(w, r)
			})
		}
	}

	tests := []struct {
		name       string
		middleware []wares.MiddlewareFunc
		expected   string
	}{
		{name: "nil middleware", expected: "h"},
		{name: "empty middleware", middleware: []wares.MiddlewareFunc{}, expected: "h"},
		{name: "one middleware", middleware: []wares.MiddlewareFunc{m("1|")}, expected: "1|h"},
		{name: "three middleware", middleware: []wares.MiddlewareFunc{m("1|"), m("2|"), m("3|")}, expected: "1|2|3|h"},
		{name: "inverted middleware", middleware: []wares.MiddlewareFunc{m("3|"), m("2|"), m("1|")}, expected: "3|2|1|h"},
	}

	t.Run("Handle", func(t *testing.T) {
		for _, test := range tests {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)

			mux := wares.NewServeMux()
			mux.Handle("/", http.HandlerFunc(h), test.middleware...)
			mux.ServeHTTP(w, r)

			if got := w.Body.String(); got != test.expected {
				t.Errorf("expected %s, got %s", test.expected, got)
			}
		}
	})

	t.Run("HandleFunc", func(t *testing.T) {
		for _, test := range tests {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)

			mux := wares.NewServeMux()
			mux.HandleFunc("/", h, test.middleware...)
			mux.ServeHTTP(w, r)

			if got := w.Body.String(); got != test.expected {
				t.Errorf("expected %s, got %s", test.expected, got)
			}
		}
	})
}
