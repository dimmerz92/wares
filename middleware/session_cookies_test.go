package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dimmerz92/wares/middleware"
	"github.com/dimmerz92/wares/sessions"
	"github.com/dimmerz92/wares/sessions/memorystore"
)

func TestSessionCookies(t *testing.T) {
	ctxKey := "test"
	cookieName := "test_cookie"
	key := "test_key"
	value := 1234

	store := memorystore.NewMemoryStore()
	manager := sessions.NewSessionManager(sessions.WithStore(store))
	defer store.Stop()

	middleware := middleware.SessionCookies(
		manager,
		middleware.WithName(cookieName),
		middleware.WithContextKey(ctxKey),
	)

	t.Run("session not used, no cookie", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)

		middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(w, r)

		if len(w.Result().Cookies()) > 0 {
			t.Error("expected no cookies")
		}
	})

	var cookie *http.Cookie

	t.Run("session used, cookie exists", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)

		middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, ok := sessions.GetSession(r.Context(), ctxKey)
			if !ok {
				t.Fatal("failed to retrieve session from context")
			}
			session.SetValue(key, value)
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(w, r)

		if len(w.Result().Cookies()) == 0 {
			t.Fatal("expected cookie")
		}

		if name := w.Result().Cookies()[0].Name; name != cookieName {
			t.Fatalf("expected cookie %s, got %s", cookieName, name)
		}

		cookie = w.Result().Cookies()[0]
	})

	t.Run("session cookie exists", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(cookie)

		middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, ok := sessions.GetSession(r.Context(), ctxKey)
			if !ok {
				t.Fatal("failed to retrieve session from context")
			}
			got, ok := session.GetValue(key).(int)
			if !ok {
				t.Fatal("expected value")
			}
			if got != value {
				t.Fatalf("expected %d, got %d", value, got)
			}
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(w, r)
	})

	t.Run("session destroyed, cookie invalidated, removed from store", func(t *testing.T) {
		sessionId := cookie.Value
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(cookie)

		middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, ok := sessions.GetSession(r.Context(), ctxKey)
			if !ok {
				t.Fatal("failed to retrieve session from context")
			}
			got, ok := session.GetValue(key).(int)
			if !ok {
				t.Fatal("expected value")
			}
			if got != value {
				t.Fatalf("expected %d, got %d", value, got)
			}
			session.Destroy()
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(w, r)

		_, ok, err := store.Get(t.Context(), sessionId)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Error("expected session to be deleted")
		}

		cookie := w.Result().Cookies()[0]
		if cookie.Value != "" || !cookie.Expires.Equal(time.Unix(0, 0).UTC()) || cookie.MaxAge >= 0 {
			t.Fatalf("expected cookie to be revoked: %#v", cookie)
		}
	})
}
