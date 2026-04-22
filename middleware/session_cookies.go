package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/dimmerz92/wares"
	"github.com/dimmerz92/wares/auth"
	"github.com/dimmerz92/wares/sessions"
)

// CookieConfig holds the configured cookie settings for the session cookie middleware.
type CookieConfig struct {
	CtxKey      string
	Name        string
	Domain      string
	Path        string
	HttpOnly    bool
	Secure      bool
	SameSite    http.SameSite
	Partitioned bool
	ErrHandler  func(w http.ResponseWriter, r *http.Request, err error)
	Skipper     func(r *http.Request) bool
}

// CookieConfig holds the configured cookie settings for the session cookie middleware.
type CookieResponseWriter struct {
	http.ResponseWriter
	config  *CookieConfig
	manager *sessions.SessionManager
	request *http.Request
	session *sessions.Session
	failed  bool
	once    sync.Once
}

// CookieOption represents a functional option for configuring Cookies.
type CookieOption func(*CookieConfig)

// WithContextKey sets the context key the session is mapped to in the request context.
func WithContextKey(key string) CookieOption {
	return func(c *CookieConfig) { c.CtxKey = key }
}

// WithName sets the cookie name.
func WithName(name string) CookieOption {
	return func(c *CookieConfig) { c.Name = name }
}

// WithDomain sets the cookie domain.
func WithDomain(domain string) CookieOption {
	return func(c *CookieConfig) { c.Domain = domain }
}

// WithPath sets the cookie path.
func WithPath(path string) CookieOption {
	return func(c *CookieConfig) { c.Path = path }
}

// WithHttpOnly sets the HttpOnly flag on the cookie.
func WithHttpOnly(b bool) CookieOption {
	return func(c *CookieConfig) { c.HttpOnly = b }
}

// WithSecure sets the Secure flag on the cookie.
func WithSecure(b bool) CookieOption {
	return func(c *CookieConfig) { c.Secure = b }
}

// WithSameSite sets the SameSite attribute on the cookie.
func WithSameSite(s http.SameSite) CookieOption {
	return func(c *CookieConfig) { c.SameSite = s }
}

// WithPartitioned sets the Partitioned attribute on the cookie.
func WithPartitioned(b bool) CookieOption {
	return func(c *CookieConfig) { c.Partitioned = b }
}

// WithErrHandler sets the error handler.
func WithErrHandler(f func(w http.ResponseWriter, r *http.Request, err error)) CookieOption {
	return func(c *CookieConfig) {
		if f != nil {
			c.ErrHandler = f
		}
	}
}

// WithSkipper sets the skipper function for skipping middleware execution.
func WithSkipper(f func(r *http.Request) bool) CookieOption {
	return func(c *CookieConfig) {
		if f != nil {
			c.Skipper = f
		}
	}
}

func SessionCookies(manager *sessions.SessionManager, opts ...CookieOption) wares.MiddlewareFunc {
	config := &CookieConfig{
		CtxKey:   auth.GenerateURLSafeNonce(16),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		ErrHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.Error("session cookies", slog.Any("error", err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		},
		Skipper: func(r *http.Request) bool { _ = r; return false },
	}

	for _, opt := range opts {
		opt(config)
	}

	if config.Name == "" {
		config.Name = fmt.Sprintf("__session_%s", auth.GenerateURLSafeNonce(16))
	}

	if config.Path == "" {
		config.Path = "/"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if config.Skipper(r) {
				next.ServeHTTP(w, r)
				return
			}

			var session *sessions.Session
			switch cookie, err := r.Cookie(config.Name); err {
			case nil:
				var ok bool
				var err error
				session, ok, err = manager.GetSession(r.Context(), cookie.Value)
				if err != nil {
					config.ErrHandler(w, r, err)
					return
				}
				if ok {
					break
				}
				fallthrough

			default:
				session = manager.NewSession()
			}

			r = r.WithContext(context.WithValue(r.Context(), config.CtxKey, session))
			crw := &CookieResponseWriter{
				ResponseWriter: w,
				config:         config,
				manager:        manager,
				request:        r,
				session:        session,
			}

			next.ServeHTTP(crw, r)
		})
	}
}

// WriteHeader runs the session commit logic once to catch and persist any changes before writing the status.
func (crw *CookieResponseWriter) WriteHeader(code int) {
	crw.once.Do(func() {
		err := crw.manager.Commit(crw.request.Context(), crw.session)
		if err != nil {
			crw.config.ErrHandler(crw.ResponseWriter, crw.request, err)
			crw.failed = true
			return
		}
		crw.SetCookie()
	})

	if !crw.failed {
		crw.ResponseWriter.WriteHeader(code)
	}
}

// Write runs the session commit logic once to catch and persist any changes before writing the response.
func (crw *CookieResponseWriter) Write(b []byte) (int, error) {
	crw.once.Do(func() {
		err := crw.manager.Commit(crw.request.Context(), crw.session)
		if err != nil {
			crw.config.ErrHandler(crw.ResponseWriter, crw.request, err)
			crw.failed = true
			return
		}
		crw.SetCookie()
	})

	if crw.failed {
		return 0, fmt.Errorf("session cookie commit failed")
	}

	return crw.ResponseWriter.Write(b)
}

// SetCookie uses the internal session state of the CookieResponseWriter to set the http Cookie on the response.
func (crw *CookieResponseWriter) SetCookie() {
	cookie := &http.Cookie{
		Name:        crw.config.Name,
		Domain:      crw.config.Domain,
		Path:        crw.config.Path,
		HttpOnly:    crw.config.HttpOnly,
		Secure:      crw.config.Secure,
		SameSite:    crw.config.SameSite,
		Partitioned: crw.config.Partitioned,
	}

	switch crw.session.GetStatus() {
	case sessions.StatusDestroy:
		cookie.MaxAge = -1
		cookie.Expires = time.Unix(0, 0)

	default:
		if crw.session.Initial() && crw.session.GetStatus() == sessions.StatusUnchanged {
			return
		}

		cookie.Value = crw.session.ID()
		cookie.Expires = crw.session.Expiry()
		cookie.MaxAge = int(time.Until(crw.session.Expiry()).Seconds())
	}

	http.SetCookie(crw.ResponseWriter, cookie)
}
