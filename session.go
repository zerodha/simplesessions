package simplesessions

import (
	"errors"
	"net/http"
	"time"
)

// Session provides the object to get, set, or clear session data.
type Session struct {
	// Map to store session data, loaded using `LoadValues` method.
	// All `Get` methods checks here before fetching from the store.
	values map[string]interface{}

	// Session manager.
	manager *Manager

	// Session ID.
	id string

	// HTTP reader and writer interfaces which are passed on to
	// `GetCookie`` and `SetCookie`` callbacks.
	reader interface{}
	writer interface{}
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

// WriteCookie creates a cookie with the given session ID and parameters,
// then calls the `SetCookie` callback. This can be used to update the cookie externally.
func (s *Session) WriteCookie(id string) error {
	ck := &http.Cookie{
		Value:    id,
		Name:     s.manager.opts.CookieName,
		Domain:   s.manager.opts.CookieDomain,
		Path:     s.manager.opts.CookiePath,
		Secure:   s.manager.opts.IsSecureCookie,
		HttpOnly: s.manager.opts.IsHTTPOnlyCookie,
		SameSite: s.manager.opts.SameSite,
	}

	// Call `SetCookie` callback to write cookie to response
	return s.manager.setCookieCb(ck, s.writer)
}

// clearCookie sets the cookie's expiry to one day prior to clear it.
func (s *Session) clearCookie() error {
	ck := &http.Cookie{
		Name:  s.manager.opts.CookieName,
		Value: "",
		// Set expiry to previous date to clear it from browser
		Expires: time.Now().AddDate(0, 0, -1),
	}

	// Call `SetCookie` callback to write cookie to response
	return s.manager.setCookieCb(ck, s.writer)
}

// ID returns the acquired session ID. If cookie is not set then empty string is returned.
func (s *Session) ID() string {
	return s.id
}

// LoadValues loads session values into memory for quick access.
// Ideal for centralized session fetching, e.g., in middleware.
// Subsequent Get/GetMulti calls return cached values, avoiding store access.
// Use ResetValues() to ensure GetAll/Get/GetMulti fetches from the store.
// Set/SetMulti/Clear do not update the values, so this method must be called again for any changes.
func (s *Session) LoadValues() error {
	var err error
	s.values, err = s.GetAll()
	return err
}

// ResetValues clears loaded values, ensuring subsequent Get, GetAll, and GetMulti calls fetch from the store.
func (s *Session) ResetValues() {
	s.values = make(map[string]interface{})
}

// GetAll gets all the fields for the given session id.
func (s *Session) GetAll() (map[string]interface{}, error) {
	// Load value from map if its already loaded.
	if len(s.values) > 0 {
		return s.values, nil
	}

	out, err := s.manager.store.GetAll(s.id)
	return out, errAs(err)
}

// GetMulti retrieves values for multiple session fields.
// If a field is not found in the store then its returned as nil.
func (s *Session) GetMulti(keys ...string) (map[string]interface{}, error) {
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

	out, err := s.manager.store.GetMulti(s.id, keys...)
	return out, errAs(err)
}

// Get retrieves a value for the given key in the session.
// If the session is already loaded, it returns the value from the existing map.
// Otherwise, it fetches the value from the store.
func (s *Session) Get(key string) (interface{}, error) {
	// Return value from map if already loaded.
	if len(s.values) > 0 {
		if val, ok := s.values[key]; ok {
			return val, nil
		}
	}

	// Fetch from store if not found in the map.
	out, err := s.manager.store.Get(s.id, key)
	return out, errAs(err)
}

// Set assigns a value to the given key in the session.
// The store determines whether to commit all values at once or store them individually.
// Use Commit() method to commit all values if the store doesn't immediately persist them.
func (s *Session) Set(key string, val interface{}) error {
	err := s.manager.store.Set(s.id, key, val)
	return errAs(err)
}

// SetMulti assigns multiple values to the session.
// The store determines whether to commit all values at once or store them individually.
func (s *Session) SetMulti(values map[string]interface{}) error {
	for k, v := range values {
		if err := s.manager.store.Set(s.id, k, v); err != nil {
			return errAs(err)
		}
	}
	return nil
}

// Commit persists all values to the store.
// The store determines whether to commit all values at once or store them individually.
func (s *Session) Commit() error {
	if err := s.manager.store.Commit(s.id); err != nil {
		return errAs(err)
	}
	return nil
}

// Delete deletes a field from session.
func (s *Session) Delete(key string) error {
	if err := s.manager.store.Delete(s.id, key); err != nil {
		return errAs(err)
	}
	return nil
}

// Clear clears session data from store and clears the cookie.
func (s *Session) Clear() error {
	if err := s.manager.store.Clear(s.id); err != nil {
		return errAs(err)
	}
	return s.clearCookie()
}

// Int is a helper to get values as integer.
func (s *Session) Int(r interface{}, err error) (int, error) {
	out, err := s.manager.store.Int(r, err)
	return out, errAs(err)
}

// Int64 is a helper to get values as Int64.
func (s *Session) Int64(r interface{}, err error) (int64, error) {
	out, err := s.manager.store.Int64(r, err)
	return out, errAs(err)
}

// UInt64 is a helper to get values as UInt64.
func (s *Session) UInt64(r interface{}, err error) (uint64, error) {
	out, err := s.manager.store.UInt64(r, err)
	return out, errAs(err)
}

// Float64 is a helper to get values as Float64.
func (s *Session) Float64(r interface{}, err error) (float64, error) {
	out, err := s.manager.store.Float64(r, err)
	return out, errAs(err)
}

// String is a helper to get values as String.
func (s *Session) String(r interface{}, err error) (string, error) {
	out, err := s.manager.store.String(r, err)
	return out, errAs(err)
}

// Bytes is a helper to get values as Bytes.
func (s *Session) Bytes(r interface{}, err error) ([]byte, error) {
	out, err := s.manager.store.Bytes(r, err)
	return out, errAs(err)
}

// Bool is a helper to get values as Bool.
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
