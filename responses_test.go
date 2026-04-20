package wares_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dimmerz92/wares"
)

func TestHTML(t *testing.T) {
	html := "<html><body>Test</body></html>"

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	wares.HTML(w, r, http.StatusTeapot, html)

	if got := w.Body.String(); got != html {
		t.Errorf("expected body %s, got %s", html, got)
	}

	if got := w.Result().StatusCode; got != http.StatusTeapot {
		t.Errorf("expected status %d, got %d", http.StatusTeapot, got)
	}

	if got := w.Header().Get("Content-Type"); got != "text/html" {
		t.Errorf("expected text/html header, got %s", got)
	}
}

func TestJSON(t *testing.T) {
	type testData struct {
		Text  string
		Int   int
		Float float64
		Bool  bool
	}

	tests := []struct {
		name   string
		status int
		data   any
	}{
		{
			name:   "struct",
			status: http.StatusOK,
			data:   testData{Text: "test", Int: 2, Float: 2.2, Bool: true},
		},
		{
			name:   "map",
			status: http.StatusTeapot,
			data:   map[string]any{"text": "test", "int": 2, "float": 2.2, "bool": true},
		},
		{
			name:   "slice",
			status: http.StatusCreated,
			data:   []string{"foo", "bar", "baz"},
		},
		{
			name:   "string",
			status: http.StatusAccepted,
			data:   "test",
		},
		{
			name:   "number",
			status: http.StatusBadRequest,
			data:   80085,
		},
		{
			name:   "nil",
			status: http.StatusUnprocessableEntity,
			data:   nil,
		},
		{
			name:   "map struct",
			status: http.StatusOK,
			data:   map[string]any{"test": testData{Text: "test", Int: 2, Float: 2.2, Bool: true}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)

			wares.JSON(w, r, test.status, test.data)

			if got := w.Result().StatusCode; got != test.status {
				t.Errorf("expected status %d, got %d", test.status, got)
			}

			if got := w.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("expected application/json header, got %s", got)
			}

			expected, err := json.Marshal(test.data)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			if got := w.Body.String(); got != string(expected) {
				t.Errorf("expected body %s, got %s", string(expected), got)
			}
		})
	}
}

type Renderer struct {
}

func (r *Renderer) Render(ctx context.Context, buf io.Writer) error {
	_ = ctx
	buf.Write([]byte("<html><body>Test</body></html>"))
	return nil
}

func TestRenderTempl(t *testing.T) {
	html := "<html><body>Test</body></html>"
	tpl := &Renderer{}

	tests := []struct {
		name     string
		tpls     []wares.Renderable
		expected string
	}{
		{name: "no templates"},
		{name: "one template", tpls: []wares.Renderable{tpl}, expected: html},
		{name: "three templates", tpls: []wares.Renderable{tpl, tpl, tpl}, expected: html + html + html},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)

			wares.Render(w, r, http.StatusTeapot, test.tpls...)

			if got := w.Body.String(); got != test.expected {
				t.Errorf("expected body %s, got %s", test.expected, got)
			}

			if got := w.Result().StatusCode; got != http.StatusTeapot {
				t.Errorf("expected status %d, got %d", http.StatusTeapot, got)
			}

			if got := w.Header().Get("Content-Type"); got != "text/html" {
				t.Errorf("expected text/html header, got %s", got)
			}
		})
	}
}

func TestRedirect(t *testing.T) {
	t.Run("standard redirect", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		route := "/redirect"
		status := http.StatusSeeOther

		wares.Redirect(w, r, status, route)

		if got := w.Result().StatusCode; got != status {
			t.Errorf("expected status %d, got %d", status, got)
		}

		if got := w.Header().Get("Location"); got != route {
			t.Errorf("expected header %s, got %s", route, got)
		}

		if w.Header().Get("Hx-Redirect") != "" {
			t.Error("did not expected hx-redirect header")
		}
	})

	t.Run("htmx redirect", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Hx-Request", "true")
		route := "/redirect"
		status := http.StatusSeeOther

		wares.Redirect(w, r, status, route)

		if got := w.Result().StatusCode; got != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, got)
		}

		if got := w.Header().Get("Hx-Redirect"); got != route {
			t.Errorf("expected header %s, got %s", route, got)
		}

		if w.Header().Get("Location") != "" {
			t.Error("did not expected location header")
		}
	})
}
