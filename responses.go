package wares

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
)

var buffers = sync.Pool{New: func() any { return new(bytes.Buffer) }}

// HTML writes the given html string to the response with the given status and apporpriate header.
func HTML(w http.ResponseWriter, r *http.Request, status int, html string) error {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(status)
	_, err := w.Write([]byte(html))

	return err
}

// JSON encodes the given data and writes it to the response with the given status and appropriate header.
func JSON(w http.ResponseWriter, r *http.Request, status int, data any) error {
	encoded, err := json.Marshal(data)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(encoded)

	return err
}

type Renderable interface {
	Render(ctx context.Context, buf io.Writer) error
}

// RenderTempl writes any number of Renderables to the response with the given status.
func Render(w http.ResponseWriter, r *http.Request, status int, tpls ...Renderable) error {
	buf := buffers.Get().(*bytes.Buffer)
	defer buf.Reset()

	for _, tpl := range tpls {
		err := tpl.Render(r.Context(), buf)
		if err != nil {
			return err
		}
	}

	return HTML(w, r, status, buf.String())
}

// IsHTMX returns true if the request originated from HTMX, otherwise false.
func IsHTMX(r *http.Request) bool {
	return r.Header.Get("Hx-Request") == "true"
}

// Redirect is a HTMX aware redirect.
//   - On a non-HTMX initiated request, a standard http.Redirect is performed.
//   - On a HTMX initiated request, the Hx-Redirect header is set and the status 200 OK is used.
//
// Reason for status override: https://github.com/bigskysoftware/htmx/issues/2052#issuecomment-1979805051
func Redirect(w http.ResponseWriter, r *http.Request, status int, route string) {
	if IsHTMX(r) {
		w.Header().Set("Hx-Redirect", route)
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, route, status)
}
