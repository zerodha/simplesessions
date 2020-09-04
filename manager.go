package simplesessions

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const (
	// Default cookie name used to store session.
	defaultCookieName = "session"

	// ContextName is the key used to store session in context passed to acquire method.
	ContextName = "_simple_session"
)

// Manager is a utility to scaffold session and store.
type Manager struct {
	// Store to be used.
	store Store

	// Store basic cookie details.
	opts *Options

	// Callback to get http cookie.
	getCookieCb func(name string, r interface{}) (*http.Cookie, error)

	// Callback to set http cookie.
	setCookieCb func(cookie *http.Cookie, w interface{}) error
}

// Options are available options to configure Manager.
type Options struct {
	// DisableAutoSet skips creation of session cookie in frontend and new session in store if session is not already set.
	DisableAutoSet bool

	// CookieName sets http cookie name. This is also sent as cookie name in `GetCookie` callback.
	CookieName string

	// CookieDomain sets hostname for the cookie. Domain specifies allowed hosts to receive the cookie.
	CookieDomain string

	// CookiePath sets path for the cookie. Path indicates a URL path that must exist in the requested URL in order to send the cookie header.
	CookiePath string

	// IsSecureCookie marks the cookie as secure cookie (only sent in HTTPS).
	IsSecureCookie bool

	// IsHTTPOnlyCookie marks the cookie as http only cookie. JS won't be able to access the cookie so prevents XSS attacks.
	IsHTTPOnlyCookie bool

	// CookieLifeTime sets expiry time for cookie.
	// If expiry time is not specified then cookie is set as session cookie which is cleared on browser close.
	CookieLifetime time.Duration

	// SameSite sets allows you to declare if your cookie should be restricted to a first-party or same-site context.
	SameSite http.SameSite
}

// New creates a new session manager for given options.
func New(opts Options) *Manager {
	m := &Manager{
		opts: &opts,
	}

	// Set default cookie name if not set
	if m.opts.CookieName == "" {
		m.opts.CookieName = defaultCookieName
	}

	// If path not given then set to root path
	if m.opts.CookiePath == "" {
		m.opts.CookiePath = "/"
	}

	return m
}

// UseStore sets the session store to be used.
func (m *Manager) UseStore(str Store) {
	m.store = str
}

// RegisterGetCookie sets a callback to get http cookie from any reader interface which
// is sent on session acquisition using `Acquire` method.
func (m *Manager) RegisterGetCookie(cb func(string, interface{}) (*http.Cookie, error)) {
	m.getCookieCb = cb
}

// RegisterSetCookie sets a callback to set cookie from http writer interface which
// is sent on session acquisition using `Acquire` method.
func (m *Manager) RegisterSetCookie(cb func(*http.Cookie, interface{}) error) {
	m.setCookieCb = cb
}

// Acquire gets a `Session` for current session cookie from store.
// If `Session` is not found on store then it creates a new session and sets on store.
// If 'DisableAutoSet` is set in options then session has to be explicitly created before
// using `Session` for getting or setting.
// `r` and `w` is request and response interfaces which are sent back in GetCookie and SetCookie callbacks respectively.
// In case of net/http `r` will be r`
// Optionally context can be passed around which is used to get already loaded session. This is useful when
// handler is wrapped with multiple middlewares and `Acquire` is already called in any of the middleware.
func (m *Manager) Acquire(r, w interface{}, c context.Context) (*Session, error) {
	// Check if any store is set
	if m.store == nil {
		return nil, fmt.Errorf("session store is not set")
	}

	// Check if callbacks are set
	if m.getCookieCb == nil {
		return nil, fmt.Errorf("callback `GetCookie` not set")
	}

	if m.setCookieCb == nil {
		return nil, fmt.Errorf("callback `SetCookie` not set")
	}

	// If a session was already set in the context by a middleware somewhere, return that.
	if c != nil {
		if v, ok := c.Value(ContextName).(*Session); ok {
			return v, nil
		}
	}

	return NewSession(m, r, w)
}
