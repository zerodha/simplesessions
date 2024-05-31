package simplesessions

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"time"
	"unicode"
)

// Context name type.
type ctxNameType string

const (
	// Default cookie name used to store session.
	defaultCookieName = "session"

	// Default cookie path.
	defaultCookiePath = "/"

	// default sessionID length.
	defaultSessIDLength = 32

	// ContextName is the key used to store session in context passed to acquire method.
	ContextName ctxNameType = "_simple_session"
)

// Manager handles the storage and management of HTTP cookies.
type Manager struct {
	// Store to be used.
	store Store

	// Store basic cookie details.
	opts *Options

	// Hook to get http cookie.
	getCookieHook func(name string, r interface{}) (*http.Cookie, error)

	// Hook to set http cookie.
	setCookieHook func(cookie *http.Cookie, w interface{}) error

	// generate cookie ID.
	generateID func() (string, error)

	// validate cookie ID.
	validateID func(string) bool
}

// Options to configure manager and cookie.
type Options struct {
	// If enabled, Acquire() will always create and return a new session if one doesn't already exist.
	// If disabled then new session can only be created using NewSession() method.
	EnableAutoCreate bool

	// Cookie ID length. Defaults to alphanumeric 32 characters.
	// Might not be applicable to some stores like SecureCookie.
	// Also not applicable if custom generateID and validateID is set.
	SessionIDLength int

	// Cookie options.
	Cookie CookieOptions
}

type CookieOptions struct {
	// Name sets http cookie name. This is also sent as cookie name in `GetCookie` callback.
	Name string

	// Domain sets hostname for the cookie. Domain specifies allowed hosts to receive the cookie.
	Domain string

	// Path sets path for the cookie. Path indicates a URL path that must exist in the requested URL in order to send the cookie header.
	Path string

	// IsSecure marks the cookie as secure cookie (only sent in HTTPS).
	IsSecure bool

	// IsHTTPOnly marks the cookie as http only cookie. JS won't be able to access the cookie so prevents XSS attacks.
	IsHTTPOnly bool

	// SameSite sets allows you to declare if your cookie should be restricted to a first-party or same-site context.
	SameSite http.SameSite

	// Expires sets absolute expiration date and time for the cookie.
	// If both Expires and MaxAge are sent then MaxAge takes precedence over Expires.
	// Cookies without a Max-age or Expires attribute – are deleted when the current session ends
	// and some browsers use session restoring when restarting. This can cause session cookies to last indefinitely.
	Expires time.Time

	// Sets the cookie's expiration in seconds from the current time, internally its rounder off to nearest seconds.
	// If both Expires and MaxAge are sent then MaxAge takes precedence over Expires.
	// Cookies without a Max-age or Expires attribute – are deleted when the current session ends
	// and some browsers use session restoring when restarting. This can cause session cookies to last indefinitely.
	MaxAge time.Duration
}

// New creates a new session manager for given options.
func New(opts Options) *Manager {
	m := &Manager{
		opts: &opts,
	}

	// Set default cookie name if not set
	if m.opts.Cookie.Name == "" {
		m.opts.Cookie.Name = defaultCookieName
	}

	// If path not given then set to root path
	if m.opts.Cookie.Path == "" {
		m.opts.Cookie.Path = defaultCookiePath
	}

	if m.opts.SessionIDLength == 0 {
		m.opts.SessionIDLength = defaultSessIDLength
	}

	// Assign default set and validate generate ID.
	m.generateID = m.defaultGenerateID
	m.validateID = m.defaultValidateID

	return m
}

// UseStore sets the session store to be used.
func (m *Manager) UseStore(str Store) {
	m.store = str
}

// SetCookieHooks cane be used to get and set HTTP cookie for the session.
//
// getCookie hook takes session ID and reader interface and returns http.Cookie and error.
// In a HTTP request context reader interface will be the http request object and
// it should obtain http.Cookie from the request object for the given cookie ID.
//
// setCookie hook takes http.Cookie object and a writer interface and returns error.
// In a HTTP request context the write interface will be the http request object and
// it should write http request with the incoming cookie.
func (m *Manager) SetCookieHooks(getCookie func(string, interface{}) (*http.Cookie, error), setCookie func(*http.Cookie, interface{}) error) {
	m.getCookieHook = getCookie
	m.setCookieHook = setCookie
}

// SetSessionIDHooks cane be used to generate and validate custom session ID.
// Bydefault alpha-numeric 32bit length session ID is used if its not set.
// - Generating custom session ID, which will be uses as the ID for storing sessions in the backend.
// - Validating custom session ID, which will be used to verify the ID before querying backend.
func (m *Manager) SetSessionIDHooks(generateID func() (string, error), validateID func(string) bool) {
	m.generateID = generateID
	m.validateID = validateID
}

// NewSession creates a new `Session` and updates the cookie with a new session ID,
// replacing any existing session ID if it exists.
func (m *Manager) NewSession(r, w interface{}) (*Session, error) {
	// Check if any store is set
	if m.store == nil {
		return nil, fmt.Errorf("session store not set")
	}

	if m.setCookieHook == nil {
		return nil, fmt.Errorf("`SetCookie` hook not set")
	}

	// Create new cookie in store and write to front.
	// Store also calls `WriteCookie`` to write to http interface.
	id, err := m.generateID()
	if err != nil {
		return nil, errAs(err)
	}

	if err = m.store.Create(id); err != nil {
		return nil, errAs(err)
	}

	var sess = &Session{
		id:      id,
		manager: m,
		reader:  r,
		writer:  w,
		cache:   nil,
	}
	// Write cookie.
	if err := sess.WriteCookie(id); err != nil {
		return nil, err
	}

	return sess, nil
}

// Acquire retrieves a `Session` from the store using the current session cookie.
//
// If session not found and `opt.EnableAutoCreate` is true, a new session is created and returned.
// If session not found and `opt.EnableAutoCreate` is false which is the default, it returns `ErrInvalidSession`.
//
// `r` and `w` are request and response interfaces which is passed back in in GetCookie and SetCookie callbacks.
// Optionally, a context can be passed to get an already loaded session, useful in middleware chains.
func (m *Manager) Acquire(c context.Context, r, w interface{}) (*Session, error) {
	// Check if any store is set
	if m.store == nil {
		return nil, fmt.Errorf("session store not set")
	}

	// Check if callbacks are set
	if m.getCookieHook == nil {
		return nil, fmt.Errorf("`GetCookie` hook not set")
	}

	if m.setCookieHook == nil {
		return nil, fmt.Errorf("`SetCookie` hook not set")
	}

	// If a session was already set in the context by a middleware somewhere, return that.
	if c != nil {
		if v, ok := c.Value(ContextName).(*Session); ok {
			return v, nil
		}
	}

	// Get existing HTTP session cookie.
	// If there's no error and there's a session ID (unvalidated at this point),
	// return a session object.
	ck, err := m.getCookieHook(m.opts.Cookie.Name, r)
	if err == nil && ck != nil && ck.Value != "" {
		return &Session{
			manager: m,
			reader:  r,
			writer:  w,
			id:      ck.Value,
			cache:   nil,
		}, nil
	}

	// If auto-creation is disabled, return an error.
	if !m.opts.EnableAutoCreate {
		return nil, ErrInvalidSession
	}

	return m.NewSession(r, w)
}

// defaultGenerateID generates a random alpha-num session ID.
// This will be the default method to generate cookie ID and
// can override using `SetCookieIDGenerate` method.
func (m *Manager) defaultGenerateID() (string, error) {
	const dict = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	bytes := make([]byte, m.opts.SessionIDLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	for k, v := range bytes {
		bytes[k] = dict[v%byte(len(dict))]
	}

	return string(bytes), nil
}

// defaultValidateID validates the incoming to ID to check
// if its alpha-numeric with configured cookie ID length.
// Can override using `SetCookieIDGenerate` method.
func (m *Manager) defaultValidateID(id string) bool {
	if len(id) != m.opts.SessionIDLength {
		return false
	}

	for _, r := range id {
		if !unicode.IsDigit(r) && !unicode.IsLetter(r) {
			return false
		}
	}

	return true
}
