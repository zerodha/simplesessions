package memory

import (
	"crypto/rand"
	"sync"
	"unicode"
)

const (
	sessionIDLen = 32
)

var (
	// Error codes for store errors. This should match the codes
	// defined in the /simplesessions package exactly.
	ErrInvalidSession = &Err{code: 1, msg: "invalid session"}
	ErrFieldNotFound  = &Err{code: 2, msg: "field not found"}
	ErrAssertType     = &Err{code: 3, msg: "assertion failed"}
	ErrNil            = &Err{code: 4, msg: "nil returned"}
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

// Store represents in-memory session store
type Store struct {
	// map to store all sessions and its values
	sessions map[string]map[string]interface{}

	mu sync.RWMutex
}

// New creates a new in-memory store instance
func New() *Store {
	return &Store{
		sessions: make(map[string]map[string]interface{}),
	}
}

// Create creates a new session id and returns it. This doesn't create the session in
// sessions map since memory can be saved by not storing empty sessions and system
// can not be stressed by just creating new sessions
func (s *Store) Create() (string, error) {
	id, err := generateID(sessionIDLen)
	if err != nil {
		return "", err
	}

	return id, err
}

// Get gets a field in session
func (s *Store) Get(id, key string) (interface{}, error) {
	if !validateID(id) {
		return nil, ErrInvalidSession
	}

	var val interface{}
	s.mu.RLock()
	// Check if session exists before accessing key from it
	v, ok := s.sessions[id]
	if ok && v != nil {
		val, ok = s.sessions[id][key]
	}
	s.mu.RUnlock()

	// If session doesn't exist or field doesn't exist then send field not found error
	// since we don't add session to sessions map on session create
	if !ok || v == nil {
		return nil, ErrFieldNotFound
	}

	return val, nil
}

// GetMulti gets a map for values for multiple keys. If key is not present in session then nil is returned.
func (s *Store) GetMulti(id string, keys ...string) (map[string]interface{}, error) {
	if !validateID(id) {
		return nil, ErrInvalidSession
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	sVals, ok := s.sessions[id]
	// If session not set then send a map with value for all keys is nil
	if sVals == nil || !ok {
		sVals = make(map[string]interface{})
	}

	res := make(map[string]interface{})
	for _, k := range keys {
		v, ok := sVals[k]
		if !ok {
			res[k] = nil
		} else {
			res[k] = v
		}
	}

	return res, nil
}

// GetAll gets all fields in session
func (s *Store) GetAll(id string) (map[string]interface{}, error) {
	if !validateID(id) {
		return nil, ErrInvalidSession
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	vals := s.sessions[id]

	return vals, nil
}

// Set sets a value to given session but stored only on commit
func (s *Store) Set(id, key string, val interface{}) error {
	if !validateID(id) {
		return ErrInvalidSession
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// If session is not set previously then create empty map
	_, ok := s.sessions[id]
	if !ok {
		s.sessions[id] = make(map[string]interface{})
	}

	s.sessions[id][key] = val

	return nil
}

// Commit does nothing here since Set sets the value.
func (s *Store) Commit(id string) error {
	return nil
}

// Delete deletes a key from session.
func (s *Store) Delete(id string, key string) error {
	if !validateID(id) {
		return ErrInvalidSession
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.sessions[id]
	if ok && s.sessions[id] != nil {
		_, ok = s.sessions[id][key]
		if ok {
			delete(s.sessions[id], key)
		}
	}

	return nil
}

// Clear clears session in redis.
func (s *Store) Clear(id string) error {
	if !validateID(id) {
		return ErrInvalidSession
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.sessions[id]
	if ok {
		delete(s.sessions, id)
	}

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

func validateID(id string) bool {
	if len(id) != sessionIDLen {
		return false
	}

	for _, r := range id {
		if !unicode.IsDigit(r) && !unicode.IsLetter(r) {
			return false
		}
	}

	return true
}

// generateID generates a random alpha-num session ID.
func generateID(n int) (string, error) {
	const dict = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	for k, v := range bytes {
		bytes[k] = dict[v%byte(len(dict))]
	}

	return string(bytes), nil
}
