package redis

import (
	"errors"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/vividvilla/simplesessions"
)

// Store represents redis session store for simple sessions.
// Each session is stored as redis hashmap.
type Store struct {
	// Maximum lifetime sessions has to be persisted.
	ttl time.Duration

	// Prefix for session id.
	prefix string

	// Temp map to store values before commit.
	tempSetMap map[string]map[string]interface{}
	mu         sync.RWMutex

	// Redis pool
	pool *redis.Pool
}

const (
	// Default prefix used to store session redis
	defaultPrefix = "session:"
	sessionIDLen  = 32
)

// New creates a new Redis store instance.
func New(pool *redis.Pool) *Store {
	return &Store{
		pool:       pool,
		prefix:     defaultPrefix,
		tempSetMap: make(map[string]map[string]interface{}),
	}
}

// SetPrefix sets session id prefix in backend
func (s *Store) SetPrefix(val string) {
	s.prefix = val
}

// SetTTL sets TTL for session in redis.
func (s *Store) SetTTL(d time.Duration) {
	s.ttl = d
}

// isValidSessionID checks is the given session id is valid.
func (s *Store) isValidSessionID(sess *simplesessions.Session, id string) bool {
	return len(id) == sessionIDLen && sess.IsValidRandomString(id)
}

// IsValid checks if the session is set for the id.
func (s *Store) IsValid(sess *simplesessions.Session, id string) (bool, error) {
	// Validate session is valid generate string or not
	return s.isValidSessionID(sess, id), nil
}

// Create returns a new session id but doesn't stores it in redis since empty hashmap can't be created.
func (s *Store) Create(sess *simplesessions.Session) (string, error) {
	id, err := sess.GenerateRandomString(sessionIDLen)
	if err != nil {
		return "", err
	}

	return id, err
}

// Get gets a field in hashmap. If field is nill then ErrFieldNotFound is raised
func (s *Store) Get(sess *simplesessions.Session, id, key string) (interface{}, error) {
	// Check if valid session
	if !s.isValidSessionID(sess, id) {
		return nil, simplesessions.ErrInvalidSession
	}

	conn := s.pool.Get()
	defer conn.Close()

	v, err := conn.Do("HGET", s.prefix+id, key)
	if v == nil || err == redis.ErrNil {
		return nil, simplesessions.ErrFieldNotFound
	}

	return v, err
}

// GetMulti gets a map for values for multiple keys. If key is not found then its set as nil.
func (s *Store) GetMulti(sess *simplesessions.Session, id string, keys ...string) (map[string]interface{}, error) {
	// Check if valid session
	if !s.isValidSessionID(sess, id) {
		return nil, simplesessions.ErrInvalidSession
	}

	conn := s.pool.Get()
	defer conn.Close()

	// Make list of args for HMGET
	args := make([]interface{}, len(keys)+1)
	args[0] = s.prefix + id
	for i := range keys {
		args[i+1] = keys[i]
	}

	v, err := redis.Values(conn.Do("HMGET", args...))
	// If field is not found then return map with fields as nil
	if len(v) == 0 || err == redis.ErrNil {
		v = make([]interface{}, len(keys))
	}

	// Form a map with returned results
	res := make(map[string]interface{})
	for i, k := range keys {
		res[k] = v[i]
	}

	return res, err
}

// GetAll gets all fields from hashmap.
func (s *Store) GetAll(sess *simplesessions.Session, id string) (map[string]interface{}, error) {
	// Check if valid session
	if !s.isValidSessionID(sess, id) {
		return nil, simplesessions.ErrInvalidSession
	}

	conn := s.pool.Get()
	defer conn.Close()

	return s.interfaceMap(conn.Do("HGETALL", s.prefix+id))
}

// Set sets a value to given session but stored only on commit
func (s *Store) Set(sess *simplesessions.Session, id, key string, val interface{}) error {
	// Check if valid session
	if !s.isValidSessionID(sess, id) {
		return simplesessions.ErrInvalidSession
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Create session map if doesn't exist
	if _, ok := s.tempSetMap[id]; !ok {
		s.tempSetMap[id] = make(map[string]interface{})
	}

	// set value to map
	s.tempSetMap[id][key] = val

	return nil
}

// Commit sets all set values
func (s *Store) Commit(sess *simplesessions.Session, id string) error {
	// Check if valid session
	if !s.isValidSessionID(sess, id) {
		return simplesessions.ErrInvalidSession
	}

	s.mu.RLock()
	vals, ok := s.tempSetMap[id]
	if !ok {
		// Nothing to commit
		s.mu.RUnlock()
		return nil
	}

	// Make slice of arguments to be passed in HGETALL command
	args := make([]interface{}, len(vals)*2+1, len(vals)*2+1)
	args[0] = s.prefix + id

	c := 1
	for k, v := range s.tempSetMap[id] {
		args[c] = k
		args[c+1] = v
		c += 2
	}
	s.mu.RUnlock()

	// Clear temp map for given session id
	s.mu.Lock()
	delete(s.tempSetMap, id)
	s.mu.Unlock()

	// Set to redis
	conn := s.pool.Get()
	defer conn.Close()
	_, err := conn.Do("HMSET", args...)

	return err
}

// Delete deletes a key from redis session hashmap.
func (s *Store) Delete(sess *simplesessions.Session, id string, key string) error {
	// Check if valid session
	if !s.isValidSessionID(sess, id) {
		return simplesessions.ErrInvalidSession
	}

	// Clear temp map for given session id
	s.mu.Lock()
	delete(s.tempSetMap, id)
	s.mu.Unlock()

	conn := s.pool.Get()
	defer conn.Close()

	_, err := conn.Do("HDEL", s.prefix+id, key)
	return err
}

// Clear clears session in redis.
func (s *Store) Clear(sess *simplesessions.Session, id string) error {
	// Check if valid session
	if !s.isValidSessionID(sess, id) {
		return simplesessions.ErrInvalidSession
	}

	conn := s.pool.Get()
	defer conn.Close()

	_, err := conn.Do("DEL", s.prefix+id)
	return err
}

// interfaceMap is a helper method which converts HGETALL reply to map of string interface
func (s *Store) interfaceMap(result interface{}, err error) (map[string]interface{}, error) {
	values, err := redis.Values(result, err)
	if err != nil {
		return nil, err
	}

	if len(values)%2 != 0 {
		return nil, errors.New("redigo: StringMap expects even number of values result")
	}

	m := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].([]byte)
		if !ok {
			return nil, errors.New("redigo: StringMap key not a bulk string value")
		}

		m[string(key)] = values[i+1]
	}

	return m, nil
}

// Int returns redis reply as integer.
func (s *Store) Int(r interface{}, err error) (int, error) {
	return redis.Int(r, err)
}

// Int64 returns redis reply as Int64.
func (s *Store) Int64(r interface{}, err error) (int64, error) {
	return redis.Int64(r, err)
}

// UInt64 returns redis reply as UInt64.
func (s *Store) UInt64(r interface{}, err error) (uint64, error) {
	return redis.Uint64(r, err)
}

// Float64 returns redis reply as Float64.
func (s *Store) Float64(r interface{}, err error) (float64, error) {
	return redis.Float64(r, err)
}

// String returns redis reply as String.
func (s *Store) String(r interface{}, err error) (string, error) {
	return redis.String(r, err)
}

// Bytes returns redis reply as Bytes.
func (s *Store) Bytes(r interface{}, err error) ([]byte, error) {
	return redis.Bytes(r, err)
}

// Bool returns redis reply as Bool.
func (s *Store) Bool(r interface{}, err error) (bool, error) {
	return redis.Bool(r, err)
}
