package securecookiestore

import (
	"sync"

	"github.com/gorilla/securecookie"
	"github.com/zerodhatech/simplesessions"
)

const (
	cookieName = "session"
)

// SecureCookieStore represents secure cookie session store
type SecureCookieStore struct {
	// Temp map to store values before commit.
	tempSetMap map[string]map[string]interface{}
	mu         sync.RWMutex

	sc *securecookie.SecureCookie
}

// New creates a new secure cookie store instance. Gorilla/securecookie is used to encode and
// encrypt cookie.
// The secretKey is required, used to authenticate the cookie value using HMAC.
// It is recommended to use a key with 32 or 64 bytes.
// The blockKey is optional, used to encrypt the cookie value -- set it to nil to not use encryption.
// If set, the length must correspond to the block size of the encryption algorithm.
// For AES, used by default, valid lengths are 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256.
func New(secretKey []byte, blockKey []byte) *SecureCookieStore {
	return &SecureCookieStore{
		sc:         securecookie.New(secretKey, blockKey),
		tempSetMap: make(map[string]map[string]interface{}),
	}
}

// encode and encrypt given interface
func (s *SecureCookieStore) encode(val interface{}) (string, error) {
	return s.sc.Encode(cookieName, val)
}

// decode encoded value to map
func (s *SecureCookieStore) decode(cookieVal string) (map[string]interface{}, error) {
	val := make(map[string]interface{})
	err := s.sc.Decode(cookieName, cookieVal, &val)
	return val, err
}

// IsValid checks if the given cookie value is valid.
func (s *SecureCookieStore) IsValid(sess *simplesessions.Session, cv string) (bool, error) {
	if _, err := s.decode(cv); err != nil {
		return false, nil
	}

	return true, nil
}

// Create creates a new secure cookie session with empty map.
func (s *SecureCookieStore) Create(sess *simplesessions.Session) (string, error) {
	// Create empty cookie
	return s.encode(make(map[string]interface{}))
}

// Get returns a field value from session
func (s *SecureCookieStore) Get(sess *simplesessions.Session, cv, key string) (interface{}, error) {
	// Decode cookie value
	vals, err := s.decode(cv)
	if err != nil {
		return nil, simplesessions.ErrInvalidSession
	}

	// Get given field
	val, ok := vals[key]
	if !ok {
		return nil, simplesessions.ErrFieldNotFound
	}

	return val, nil
}

// GetMulti returns values for multiple fields in session.
// If a field is not present then nil is returned.
func (s *SecureCookieStore) GetMulti(sess *simplesessions.Session, cv string, keys ...string) (map[string]interface{}, error) {
	// Decode cookie value
	vals, err := s.decode(cv)
	if err != nil {
		return nil, simplesessions.ErrInvalidSession
	}

	// Get all given fields
	res := make(map[string]interface{})
	for _, k := range keys {
		res[k], _ = vals[k]
	}

	return res, nil
}

// GetAll returns all field for given session.
func (s *SecureCookieStore) GetAll(sess *simplesessions.Session, cv string) (map[string]interface{}, error) {
	vals, err := s.decode(cv)
	if err != nil {
		return nil, simplesessions.ErrInvalidSession
	}

	return vals, nil
}

// Set sets a field in session but not saved untill commit is called.
func (s *SecureCookieStore) Set(sess *simplesessions.Session, cv, key string, val interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create session map if doesn't exist
	if _, ok := s.tempSetMap[cv]; !ok {
		s.tempSetMap[cv] = make(map[string]interface{})
	}

	// set value to map
	s.tempSetMap[cv][key] = val

	return nil
}

// Commit saves all the field set previously to cookie.
func (s *SecureCookieStore) Commit(sess *simplesessions.Session, cv string) error {
	// Decode current cookie
	vals, err := s.decode(cv)
	if err != nil {
		return simplesessions.ErrInvalidSession
	}

	s.mu.RLock()
	tempVals, ok := s.tempSetMap[cv]
	s.mu.RUnlock()
	if !ok {
		// Nothing to commit
		return nil
	}

	// Assign new fields to current values
	for k, v := range tempVals {
		vals[k] = v
	}

	// Encode new values
	encoded, err := s.encode(vals)
	if err != nil {
		return err
	}

	// Clear temp map for given session id
	s.mu.Lock()
	delete(s.tempSetMap, cv)
	s.mu.Unlock()

	// Write cookie
	return sess.WriteCookie(encoded)
}

// Delete deletes a field from session.
func (s *SecureCookieStore) Delete(sess *simplesessions.Session, cv, key string) error {
	// Decode current cookie
	vals, err := s.decode(cv)
	if err != nil {
		return simplesessions.ErrInvalidSession
	}

	// Delete given key in current values
	delete(vals, key)

	// Encode new values
	encoded, err := s.encode(vals)
	if err != nil {
		return err
	}

	// Clear temp map for given session id
	s.mu.Lock()
	delete(s.tempSetMap, cv)
	s.mu.Unlock()

	// Write new value to cookie
	return sess.WriteCookie(encoded)
}

// Clear clears the session.
func (s *SecureCookieStore) Clear(sess *simplesessions.Session, id string) error {
	encoded, err := s.encode(make(map[string]interface{}))
	if err != nil {
		return err
	}

	// Write new value to cookie
	return sess.WriteCookie(encoded)
}

// Int is a helper method to type assert as integer
func (s *SecureCookieStore) Int(r interface{}, err error) (int, error) {
	if err != nil {
		return 0, err
	}

	v, ok := r.(int)
	if !ok {
		err = simplesessions.ErrAssertType
	}

	return v, err
}

// Int64 is a helper method to type assert as Int64
func (s *SecureCookieStore) Int64(r interface{}, err error) (int64, error) {
	if err != nil {
		return 0, err
	}

	v, ok := r.(int64)
	if !ok {
		err = simplesessions.ErrAssertType
	}

	return v, err
}

// UInt64 is a helper method to type assert as UInt64
func (s *SecureCookieStore) UInt64(r interface{}, err error) (uint64, error) {
	if err != nil {
		return 0, err
	}

	v, ok := r.(uint64)
	if !ok {
		err = simplesessions.ErrAssertType
	}

	return v, err
}

// Float64 is a helper method to type assert as Float64
func (s *SecureCookieStore) Float64(r interface{}, err error) (float64, error) {
	if err != nil {
		return 0, err
	}

	v, ok := r.(float64)
	if !ok {
		err = simplesessions.ErrAssertType
	}

	return v, err
}

// String is a helper method to type assert as String
func (s *SecureCookieStore) String(r interface{}, err error) (string, error) {
	if err != nil {
		return "", err
	}

	v, ok := r.(string)
	if !ok {
		err = simplesessions.ErrAssertType
	}

	return v, err
}

// Bytes is a helper method to type assert as Bytes
func (s *SecureCookieStore) Bytes(r interface{}, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}

	v, ok := r.([]byte)
	if !ok {
		err = simplesessions.ErrAssertType
	}

	return v, err
}

// Bool is a helper method to type assert as Bool
func (s *SecureCookieStore) Bool(r interface{}, err error) (bool, error) {
	if err != nil {
		return false, err
	}

	v, ok := r.(bool)
	if !ok {
		err = simplesessions.ErrAssertType
	}

	return v, err
}
