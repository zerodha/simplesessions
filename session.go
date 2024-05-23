package simplesessions

import (
	"errors"
	"net/http"
	"sync"
	"time"
)

// Session provides the object to get, set, or clear session data.
type Session struct {
	// Map to store session data, loaded using `CacheAll` method.
	// All `Get` methods checks here before fetching from the store.
	cache    map[string]interface{}
	cacheMux sync.RWMutex

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

// getCacheAll returns a copy of cached map.
func (s *Session) getCacheAll() map[string]interface{} {
	s.cacheMux.RLock()
	defer s.cacheMux.RUnlock()

	if s.cache == nil {
		return nil
	}

	out := map[string]interface{}{}
	for k, v := range s.cache {
		out[k] = v
	}

	return out
}

// getCacheAll returns a map of values for the given list of keys.
// If key doesn't exist then ErrFieldNotFound is returned.
func (s *Session) getCache(key ...string) map[string]interface{} {
	s.cacheMux.RLock()
	defer s.cacheMux.RUnlock()

	if s.cache == nil {
		return nil
	}

	out := map[string]interface{}{}
	for _, k := range key {
		v, ok := s.cache[k]
		if ok {
			out[k] = v
		} else {
			out[k] = ErrFieldNotFound
		}
	}

	return out
}

// setCache sets a cache for given kv pairs.
func (s *Session) setCache(data map[string]interface{}) {
	s.cacheMux.Lock()
	defer s.cacheMux.Unlock()

	// If cacheAll() is not called the don't maintain cache.
	if s.cache == nil {
		return
	}

	for k, v := range data {
		s.cache[k] = v
	}
}

// deleteCache sets a cache for given kv pairs.
func (s *Session) deleteCache(key ...string) {
	s.cacheMux.Lock()
	defer s.cacheMux.Unlock()

	// If cacheAll() is not called the don't maintain cache.
	if s.cache == nil {
		return
	}

	for _, k := range key {
		delete(s.cache, k)
	}
}

// CacheAll loads session values into memory for quick access.
// Ideal for centralized session fetching, e.g., in middleware.
// Subsequent Get/GetMulti calls return cached values, avoiding store access.
// Use ResetCache() to ensure GetAll/Get/GetMulti fetches from the store.
func (s *Session) CacheAll() error {
	all, err := s.manager.store.GetAll(s.id)
	if err != nil {
		return err
	}

	s.cacheMux.Lock()
	defer s.cacheMux.Unlock()
	s.cache = map[string]interface{}{}
	for k, v := range all {
		s.cache[k] = v
	}

	return nil
}

// ResetCache clears loaded values, ensuring subsequent Get, GetAll, and GetMulti calls fetch from the store.
func (s *Session) ResetCache() {
	s.cacheMux.Lock()
	defer s.cacheMux.Unlock()
	s.cache = nil
}

// GetAll gets all the fields for the given session id.
func (s *Session) GetAll() (map[string]interface{}, error) {
	// Try to get the values from cache.
	c := s.getCacheAll()
	if c != nil {
		return c, nil
	}

	// Get the values from store.
	out, err := s.manager.store.GetAll(s.id)
	return out, errAs(err)
}

// GetMulti retrieves values for multiple session fields.
// If a field is not found in the store then its returned as nil.
func (s *Session) GetMulti(key ...string) (map[string]interface{}, error) {
	// Try to get the values from cache.
	c := s.getCache(key...)
	if c != nil {
		return c, nil
	}

	out, err := s.manager.store.GetMulti(s.id, key...)
	return out, errAs(err)
}

// Get retrieves a value for the given key in the session.
// If the session is already loaded, it returns the value from the existing map.
// Otherwise, it fetches the value from the store.
func (s *Session) Get(key string) (interface{}, error) {
	// Try to get the values from cache.
	c := s.getCache(key)
	if c != nil {
		err, ok := c[key].(error)
		if ok {
			return nil, err
		} else {
			return c[key], nil
		}
	}

	// Fetch from store if not found in the map.
	out, err := s.manager.store.Get(s.id, key)
	return out, errAs(err)
}

// Set assigns a value to the given key in the session.
func (s *Session) Set(key string, val interface{}) error {
	err := s.manager.store.Set(s.id, key, val)
	if err == nil {
		s.setCache(map[string]interface{}{
			key: val,
		})
	}
	return errAs(err)
}

// SetMulti assigns multiple values to the session.
func (s *Session) SetMulti(data map[string]interface{}) error {
	err := s.manager.store.SetMulti(s.id, data)
	if err == nil {
		s.setCache(data)
	}
	return errAs(err)
}

// Delete deletes a field from session.
func (s *Session) Delete(key ...string) error {
	err := s.manager.store.Delete(s.id, key...)
	if err == nil {
		s.deleteCache(key...)
	}
	return errAs(err)
}

// Clear clears session data from store and clears the cookie.
func (s *Session) Clear() error {
	err := s.manager.store.Clear(s.id)
	if err != nil {
		return errAs(err)
	} else {
		s.ResetCache()
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
