package securecookie

import (
	"fmt"
	"sync"

	"github.com/gorilla/securecookie"
)

const (
	defaultCookieName = "session"
)

var (
	// Error codes for store errors. This should match the codes
	// defined in the /simplesessions package exactly.
	ErrInvalidSession = &Err{code: 1, msg: "invalid session"}
	ErrAssertType     = &Err{code: 2, msg: "assertion failed"}
	ErrNil            = &Err{code: 3, msg: "nil returned"}
)

type Err struct {
	code int
	msg  string
}

func (e *Err) Error() string {
	return e.msg
}

func (e *Err) Code() int {
	return e.code
}

// Store represents secure cookie session store
type Store struct {
	// Temp map to store values before commit.
	tempSetMap map[string]map[string]interface{}
	mu         sync.RWMutex

	sc         *securecookie.SecureCookie
	cookieName string
}

// New creates a new secure cookie store instance. Gorilla/securecookie is used to encode and
// encrypt cookie.
// The secretKey is required, used to authenticate the cookie value using HMAC.
// It is recommended to use a key with 32 or 64 bytes.
// The blockKey is optional, used to encrypt the cookie value -- set it to nil to not use encryption.
// If set, the length must correspond to the block size of the encryption algorithm.
// For AES, used by default, valid lengths are 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256.
func New(secretKey []byte, blockKey []byte) *Store {
	return &Store{
		cookieName: defaultCookieName,
		sc:         securecookie.New(secretKey, blockKey),
		tempSetMap: make(map[string]map[string]interface{}),
	}
}

// encode and encrypt given interface
func (s *Store) encode(val interface{}) (string, error) {
	return s.sc.Encode(s.cookieName, val)
}

// decode encoded value to map
func (s *Store) decode(cookieVal string) (map[string]interface{}, error) {
	val := make(map[string]interface{})
	err := s.sc.Decode(s.cookieName, cookieVal, &val)
	return val, err
}

// SetCookieName sets the cookie name for securecookie
func (s *Store) SetCookieName(cookieName string) {
	s.cookieName = cookieName
}

// IsValid checks if the given cookie value is valid.
func (s *Store) IsValid(cv string) bool {
	if _, err := s.decode(cv); err != nil {
		return false
	}
	return true
}

// Create creates a new secure cookie session with empty map.
// Once called, Flush() should be called to retrieve the updated.
func (s *Store) Create(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tempSetMap[id] = make(map[string]interface{})
	return nil
}

// Get returns a field value from session
func (s *Store) Get(cv, key string) (interface{}, error) {
	// Decode cookie value
	vals, err := s.decode(cv)
	if err != nil {
		return nil, ErrInvalidSession
	}

	// Get given field
	val, ok := vals[key]
	if !ok {
		return nil, nil
	}

	return val, nil
}

// GetMulti returns values for multiple fields in session.
// If a field is not present then nil is returned.
func (s *Store) GetMulti(cv string, keys ...string) (map[string]interface{}, error) {
	// Decode cookie value
	vals, err := s.decode(cv)
	if err != nil {
		return nil, ErrInvalidSession
	}

	// Get all given fields
	var (
		ok  bool
		res = make(map[string]interface{})
	)
	for _, k := range keys {
		res[k], ok = vals[k]
		if !ok {
			res[k] = nil
		}
	}

	return res, nil
}

// GetAll returns all field for given session.
func (s *Store) GetAll(cv string) (map[string]interface{}, error) {
	vals, err := s.decode(cv)
	if err != nil {
		return nil, ErrInvalidSession
	}

	return vals, nil
}

// Set sets a field in session but not saved untill commit is called.
// Flush() should be called to retrieve the updated, unflushed values
// and written to the cookie externally.
func (s *Store) Set(cv, key string, val interface{}) error {
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

// SetMulti sets given map of kv pairs to session. Flush() should be
// called to retrieve the updated, unflushed values and written to the cookie
// externally.
func (s *Store) SetMulti(cv string, vals map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create session map if doesn't exist
	if _, ok := s.tempSetMap[cv]; !ok {
		s.tempSetMap[cv] = make(map[string]interface{})
	}

	for k, v := range vals {
		s.tempSetMap[cv][k] = v
	}

	return nil
}

// Flush flushes the 'set' buffer and returns encoded secure cookie value ready to be saved.
// This value should be written to the cookie externally.
// This can be used with simplessions.Session.WriteCookie.
// val, _ := str.Flush(cookieVal)
// sess.WriteCookie(val)
func (s *Store) Flush(cv string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	vals, ok := s.tempSetMap[cv]
	if !ok {
		return "", fmt.Errorf("nothing to flush")
	}

	delete(s.tempSetMap, cv)

	encoded, err := s.encode(vals)
	return encoded, err
}

// Delete deletes a field from session. Once called, Flush() should be
// called to retrieve the updated, unflushed values and written to the cookie
// externally.
func (s *Store) Delete(cv, key string) error {
	// Decode current cookie
	vals, err := s.decode(cv)
	if err != nil {
		return ErrInvalidSession
	}

	// Delete given key in current values.
	delete(vals, key)

	// Create session map if doesn't exist.
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tempSetMap[cv]; !ok {
		s.tempSetMap[cv] = make(map[string]interface{})
	}

	for k, v := range vals {
		s.tempSetMap[cv][k] = v
	}

	// After this, Flush() should be called to obtain the updated encoded
	// values to be written to the cookie externally.
	return nil
}

// Clear clears the session. Once called, Flush() should be
// called to retrieve the updated, unflushed values and written to the cookie
// externally.
func (s *Store) Clear(cv string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tempSetMap[cv] = make(map[string]interface{})
	return nil
}

// Int is a helper method to type assert as integer
func (s *Store) Int(r interface{}, err error) (int, error) {
	if err != nil {
		return 0, err
	}

	v, ok := r.(int)
	if !ok {
		err = ErrAssertType
	}

	return v, err
}

// Int64 is a helper method to type assert as Int64
func (s *Store) Int64(r interface{}, err error) (int64, error) {
	if err != nil {
		return 0, err
	}

	v, ok := r.(int64)
	if !ok {
		err = ErrAssertType
	}

	return v, err
}

// UInt64 is a helper method to type assert as UInt64
func (s *Store) UInt64(r interface{}, err error) (uint64, error) {
	if err != nil {
		return 0, err
	}

	v, ok := r.(uint64)
	if !ok {
		err = ErrAssertType
	}

	return v, err
}

// Float64 is a helper method to type assert as Float64
func (s *Store) Float64(r interface{}, err error) (float64, error) {
	if err != nil {
		return 0, err
	}

	v, ok := r.(float64)
	if !ok {
		err = ErrAssertType
	}

	return v, err
}

// String is a helper method to type assert as String
func (s *Store) String(r interface{}, err error) (string, error) {
	if err != nil {
		return "", err
	}

	v, ok := r.(string)
	if !ok {
		err = ErrAssertType
	}

	return v, err
}

// Bytes is a helper method to type assert as Bytes
func (s *Store) Bytes(r interface{}, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}

	v, ok := r.([]byte)
	if !ok {
		err = ErrAssertType
	}

	return v, err
}

// Bool is a helper method to type assert as Bool
func (s *Store) Bool(r interface{}, err error) (bool, error) {
	if err != nil {
		return false, err
	}

	v, ok := r.(bool)
	if !ok {
		err = ErrAssertType
	}

	return v, err
}
