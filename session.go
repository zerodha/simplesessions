package simplesessions

import (
	"errors"
	"net/http"
	"time"
)

// Session is utility for get, set or clear session.
type Session struct {
	// Map to store session data which can be loaded using `Load` method.
	// Get session method check if the field is available here before getting from store directly.
	values map[string]interface{}

	// Session manager.
	manager *Manager

	// Current http cookie. This is passed down to `SetCookie` callback.
	cookie *http.Cookie

	// HTTP reader and writer interfaces which are passed on to
	// `GetCookie`` and `SetCookie`` callback respectively.
	reader interface{}
	writer interface{}

	// Track if session is set in store or not
	// used to throw and error is autoSet is not enabled and user
	// explicitly didn't create new session in store.
	isSet bool
}

var (
	// ErrInvalidSession is raised when session is tried to access before setting it or its not set in store.
	// Handle this and create new session.
	// Store code = 1
	ErrInvalidSession = errors.New("simplesession: invalid session")

	// ErrFieldNotFound is raised when given key is not found in store
	// Store code = 2
	ErrFieldNotFound = errors.New("simplesession: session field not found in store")

	// ErrAssertType is raised when type assertion fails
	// Store code = 3
	ErrAssertType = errors.New("simplesession: invalid type assertion")

	// ErrNil is raised when returned value is nil.
	// Store code = 4
	ErrNil = errors.New("simplesession: nil returned")
)

type errCode interface {
	Code() int
}

// NewSession creates a new session. Reads cookie info from `GetCookie“ callback
// and validate the session with current store. If cookie not set then it creates
// new session and calls `SetCookie“ callback. If `DisableAutoSet` is set then it
// skips new session creation and should be manually done using `Create` method.
// If a cookie is found but its invalid in store then `ErrInvalidSession` error is returned.
func NewSession(m *Manager, r, w interface{}) (*Session, error) {
	var (
		err  error
		sess = &Session{
			manager: m,
			reader:  r,
			writer:  w,
			values:  make(map[string]interface{}),
		}
	)

	// Get existing http session cookie
	sess.cookie, err = m.getCookieCb(m.opts.CookieName, r)

	// Create new session
	if err == http.ErrNoCookie {
		// Skip creating new cookie in store. User has to manually create before doing Get or Set.
		if m.opts.DisableAutoSet {
			return sess, nil
		}

		// Create new cookie in store and write to front
		// Store also calls `WriteCookie`` to write to http interface
		cv, err := m.store.Create()
		if err != nil {
			return nil, errAs(err)
		}

		// Write cookie
		if err := sess.WriteCookie(cv); err != nil {
			return nil, err
		}

		// Set isSet flag
		sess.isSet = true
	} else if err != nil {
		return nil, err
	}

	// Set isSet flag
	sess.isSet = true

	return sess, nil
}

// WriteCookie updates the cookie and calls `SetCookie` callback.
// This method can also be used by store to update cookie whenever the cookie value changes.
func (s *Session) WriteCookie(cv string) error {
	s.cookie = &http.Cookie{
		Value:    cv,
		Name:     s.manager.opts.CookieName,
		Domain:   s.manager.opts.CookieDomain,
		Path:     s.manager.opts.CookiePath,
		Secure:   s.manager.opts.IsSecureCookie,
		HttpOnly: s.manager.opts.IsHTTPOnlyCookie,
		SameSite: s.manager.opts.SameSite,
	}

	// Set cookie expiry
	if s.manager.opts.CookieLifetime != 0 {
		s.cookie.Expires = time.Now().Add(s.manager.opts.CookieLifetime)
	}

	// Call `SetCookie` callback to write cookie to response
	return s.manager.setCookieCb(s.cookie, s.writer)
}

// clearCookie sets expiry of the cookie to one day before to clear it.
func (s *Session) clearCookie() error {
	s.cookie = &http.Cookie{
		Name:  s.manager.opts.CookieName,
		Value: "",
		// Set expiry to previous date to clear it from browser
		Expires: time.Now().AddDate(0, 0, -1),
	}

	// Call `SetCookie` callback to write cookie to response
	return s.manager.setCookieCb(s.cookie, s.writer)
}

// Create a new session. This is implicit when option `DisableAutoSet` is false
// else session has to be manually created before setting or getting values.
func (s *Session) Create() error {
	// Create new cookie in store and write to front.
	cv, err := s.manager.store.Create()
	if err != nil {
		return errAs(err)
	}

	// Write cookie
	if err := s.WriteCookie(cv); err != nil {
		return err
	}

	// Set isSet flag
	s.isSet = true

	return nil
}

// ID returns the acquired session ID. If cookie is not set then empty string is returned.
func (s *Session) ID() string {
	if s.cookie != nil {
		return s.cookie.Value
	}
	return ""
}

// LoadValues loads the session values in memory.
// Get session field tries to get value from memory before hitting store.
func (s *Session) LoadValues() error {
	var err error
	s.values, err = s.GetAll()
	return err
}

// ResetValues reset the loaded values using `LoadValues` method.ResetValues
// Subsequent Get, GetAll and GetMulti
func (s *Session) ResetValues() {
	s.values = make(map[string]interface{})
}

// GetAll gets all the fields in the session.
func (s *Session) GetAll() (map[string]interface{}, error) {
	// Check if session is set before accessing it
	if !s.isSet {
		return nil, ErrInvalidSession
	}

	// Load value from map if its already loaded
	if len(s.values) > 0 {
		return s.values, nil
	}

	out, err := s.manager.store.GetAll(s.cookie.Value)
	return out, errAs(err)
}

// GetMulti gets a map of values for multiple session keys.
func (s *Session) GetMulti(keys ...string) (map[string]interface{}, error) {
	// Check if session is set before accessing it
	if !s.isSet {
		return nil, ErrInvalidSession
	}

	// Load values from map if its already loaded
	if len(s.values) > 0 {
		vals := make(map[string]interface{})
		for _, k := range keys {
			if v, ok := s.values[k]; ok {
				vals[k] = v
			}
		}

		return vals, nil
	}

	out, err := s.manager.store.GetMulti(s.cookie.Value, keys...)
	return out, errAs(err)
}

// Get gets a value for given key in session.
// If session is already loaded using `Load` then returns values from
// existing map instead of getting it from store.
func (s *Session) Get(key string) (interface{}, error) {
	// Check if session is set before accessing it
	if !s.isSet {
		return nil, ErrInvalidSession
	}

	// Load value from map if its already loaded
	if len(s.values) > 0 {
		if val, ok := s.values[key]; ok {
			return val, nil
		}
	}

	// Get from backend if not found in previous step
	out, err := s.manager.store.Get(s.cookie.Value, key)
	return out, errAs(err)
}

// Set sets a value for given key in session. Its up to store to commit
// all previously set values at once or store it on each set.
func (s *Session) Set(key string, val interface{}) error {
	// Check if session is set before accessing it
	if !s.isSet {
		return ErrInvalidSession
	}

	err := s.manager.store.Set(s.cookie.Value, key, val)
	return errAs(err)
}

// SetMulti sets all values in the session.
// Its up to store to commit all previously
// set values at once or store it on each set.
func (s *Session) SetMulti(values map[string]interface{}) error {
	// Check if session is set before accessing it
	if !s.isSet {
		return ErrInvalidSession
	}

	for k, v := range values {
		if err := s.manager.store.Set(s.cookie.Value, k, v); err != nil {
			return errAs(err)
		}
	}

	return nil
}

// Commit commits all set to store. Its up to store to commit
// all previously set values at once or store it on each set.
func (s *Session) Commit() error {
	// Check if session is set before accessing it
	if !s.isSet {
		return ErrInvalidSession
	}

	if err := s.manager.store.Commit(s.cookie.Value); err != nil {
		return errAs(err)
	}

	return nil
}

// Delete deletes a field from session.
func (s *Session) Delete(key string) error {
	// Check if session is set before accessing it
	if !s.isSet {
		return ErrInvalidSession
	}

	if err := s.manager.store.Delete(s.cookie.Value, key); err != nil {
		return errAs(err)
	}

	return nil
}

// Clear clears session data from store and clears the cookie
func (s *Session) Clear() error {
	// Check if session is set before accessing it
	if !s.isSet {
		return ErrInvalidSession
	}

	if err := s.manager.store.Clear(s.cookie.Value); err != nil {
		return errAs(err)
	}

	return s.clearCookie()
}

// Int is a helper to get values as integer
func (s *Session) Int(r interface{}, err error) (int, error) {
	out, err := s.manager.store.Int(r, err)
	return out, errAs(err)
}

// Int64 is a helper to get values as Int64
func (s *Session) Int64(r interface{}, err error) (int64, error) {
	out, err := s.manager.store.Int64(r, err)
	return out, errAs(err)
}

// UInt64 is a helper to get values as UInt64
func (s *Session) UInt64(r interface{}, err error) (uint64, error) {
	out, err := s.manager.store.UInt64(r, err)
	return out, errAs(err)
}

// Float64 is a helper to get values as Float64
func (s *Session) Float64(r interface{}, err error) (float64, error) {
	out, err := s.manager.store.Float64(r, err)
	return out, errAs(err)
}

// String is a helper to get values as String
func (s *Session) String(r interface{}, err error) (string, error) {
	out, err := s.manager.store.String(r, err)
	return out, errAs(err)
}

// Bytes is a helper to get values as Bytes
func (s *Session) Bytes(r interface{}, err error) ([]byte, error) {
	out, err := s.manager.store.Bytes(r, err)
	return out, errAs(err)
}

// Bool is a helper to get values as Bool
func (s *Session) Bool(r interface{}, err error) (bool, error) {
	out, err := s.manager.store.Bool(r, err)
	return out, errAs(err)
}

// errAs takes an error coming from a store and maps it to an error
// defined in the sessions package based on its code, if it's available at all.
func errAs(err error) error {
	if err == nil {
		return nil
	}

	e, ok := err.(errCode)
	if !ok {
		return err
	}

	switch e.Code() {
	case 1:
		return ErrInvalidSession
	case 2:
		return ErrFieldNotFound
	case 3:
		return ErrAssertType
	case 4:
		return ErrNil
	}

	return err
}
