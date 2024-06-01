package memory

import (
	"sync"
)

var (
	// Error codes for store errors. This should match the codes
	// defined in the /simplesessions package exactly.
	ErrInvalidSession = &Err{code: 1, msg: "invalid session"}
	ErrNil            = &Err{code: 2, msg: "nil returned"}
	ErrAssertType     = &Err{code: 3, msg: "assertion failed"}
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
func (s *Store) Create(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if session already exists.
	_, ok := s.sessions[id]
	if ok {
		return nil
	}

	s.sessions[id] = make(map[string]interface{})
	return nil
}

// Get gets a field in session
func (s *Store) Get(id, key string) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if session exists before accessing key from it.
	sess, ok := s.sessions[id]
	if !ok {
		return nil, ErrInvalidSession
	}

	val, ok := sess[key]
	if !ok {
		return nil, nil
	}

	return val, nil
}

// GetMulti gets a map for values for multiple keys. If key is not present in session then nil is returned.
func (s *Store) GetMulti(id string, keys ...string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, ok := s.sessions[id]
	if !ok {
		return nil, ErrInvalidSession
	}

	out := make(map[string]interface{})
	for _, k := range keys {
		v, ok := sess[k]
		if !ok {
			out[k] = nil
		} else {
			out[k] = v
		}
	}

	return out, nil
}

// GetAll gets all fields in session
func (s *Store) GetAll(id string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, ok := s.sessions[id]
	if !ok {
		return nil, ErrInvalidSession
	}

	// Copy the map.
	out := make(map[string]interface{})
	for k, v := range sess {
		out[k] = v
	}

	return out, nil
}

// Set sets a value to given session.
func (s *Store) Set(id, key string, val interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.sessions[id]
	if !ok {
		return ErrInvalidSession
	}
	s.sessions[id][key] = val
	return nil
}

// SetMulti sets multiple key value pair to given session.
func (s *Store) SetMulti(id string, data map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.sessions[id]
	if !ok {
		return ErrInvalidSession
	}

	for k, v := range data {
		s.sessions[id][k] = v
	}

	return nil
}

// Delete deletes a key from session.
func (s *Store) Delete(id string, keys ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.sessions[id]
	if !ok {
		return ErrInvalidSession
	}

	for _, k := range keys {
		delete(s.sessions[id], k)
	}

	return nil
}

// Clear empties the session.
func (s *Store) Clear(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.sessions[id]
	if !ok {
		return ErrInvalidSession
	}
	s.sessions[id] = make(map[string]interface{})

	return nil
}

// Destroy deletes the entire session.
func (s *Store) Destroy(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.sessions[id]
	if !ok {
		return ErrInvalidSession
	}
	delete(s.sessions, id)

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
