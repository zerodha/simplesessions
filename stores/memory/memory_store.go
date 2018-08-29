package memorystore

import (
	"sync"

	"github.com/zerodhatech/simplesessions"
)

const (
	sessionIDLen = 32
)

// MemoryStore represents in-memory session store
type MemoryStore struct {
	// map to store all sessions and its values
	sessions map[string]map[string]interface{}

	mu sync.RWMutex
}

// New creates a new in-memory store instance
func New() *MemoryStore {
	return &MemoryStore{
		sessions: make(map[string]map[string]interface{}),
	}
}

// isValidSessionID checks is the given session id is valid.
func (s *MemoryStore) isValidSessionID(sess *simplesessions.Session, id string) bool {
	return len(id) == sessionIDLen && sess.IsValidRandomString(id)
}

// IsValid checks if the session is set for the id
func (s *MemoryStore) IsValid(sess *simplesessions.Session, id string) (bool, error) {
	return s.isValidSessionID(sess, id), nil
}

// Create creates a new session id and returns it. This doesn't create the session in
// sessions map since memory can be saved by not storing empty sessions and system
// can not be stressed by just creating new sessions
func (s *MemoryStore) Create(sess *simplesessions.Session) (string, error) {
	id, err := sess.GenerateRandomString(sessionIDLen)
	if err != nil {
		return "", err
	}

	return id, err
}

// Get gets a field in session
func (s *MemoryStore) Get(sess *simplesessions.Session, id, key string) (interface{}, error) {
	if !s.isValidSessionID(sess, id) {
		return nil, simplesessions.ErrInvalidSession
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
		return nil, simplesessions.ErrFieldNotFound
	}

	return val, nil
}

// GetMulti gets a map for values for multiple keys. If key is not present in session then nil is returned.
func (s *MemoryStore) GetMulti(sess *simplesessions.Session, id string, keys ...string) (map[string]interface{}, error) {
	// Check if valid session
	if !s.isValidSessionID(sess, id) {
		return nil, simplesessions.ErrInvalidSession
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
func (s *MemoryStore) GetAll(sess *simplesessions.Session, id string) (map[string]interface{}, error) {
	// Check if valid session
	if !s.isValidSessionID(sess, id) {
		return nil, simplesessions.ErrInvalidSession
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	vals := s.sessions[id]

	return vals, nil
}

// Set sets a value to given session but stored only on commit
func (s *MemoryStore) Set(sess *simplesessions.Session, id, key string, val interface{}) error {
	// Check if valid session
	if !s.isValidSessionID(sess, id) {
		return simplesessions.ErrInvalidSession
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
func (s *MemoryStore) Commit(sess *simplesessions.Session, id string) error {
	return nil
}

// Clear clears session in redis
func (s *MemoryStore) Clear(sess *simplesessions.Session, id string) error {
	// Check if valid session
	if !s.isValidSessionID(sess, id) {
		return simplesessions.ErrInvalidSession
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
func (s *MemoryStore) Int(r interface{}, err error) (int, error) {
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
func (s *MemoryStore) Int64(r interface{}, err error) (int64, error) {
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
func (s *MemoryStore) UInt64(r interface{}, err error) (uint64, error) {
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
func (s *MemoryStore) Float64(r interface{}, err error) (float64, error) {
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
func (s *MemoryStore) String(r interface{}, err error) (string, error) {
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
func (s *MemoryStore) Bytes(r interface{}, err error) ([]byte, error) {
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
func (s *MemoryStore) Bool(r interface{}, err error) (bool, error) {
	if err != nil {
		return false, err
	}

	v, ok := r.(bool)
	if !ok {
		err = simplesessions.ErrAssertType
	}

	return v, err
}
