package redisstore

import (
	"errors"
	"sync"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/zerodhatech/simplesessions"
)

// RedisStore represents redis session store for simple sessions.
// Each session is stored as redis hashmap.
type RedisStore struct {
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
)

// New creates a new in-memory store instance
func New(pool *redis.Pool) *RedisStore {
	return &RedisStore{
		pool:       pool,
		prefix:     defaultPrefix,
		tempSetMap: make(map[string]map[string]interface{}),
	}
}

// SetPrefix sets session id prefix in backend
func (s *RedisStore) SetPrefix(val string) {
	s.prefix = val
}

// SetTTL sets TTL for session in redis.
func (s *RedisStore) SetTTL(d time.Duration) {
	s.ttl = d
}

// IsValid checks if the session is set for the id.
func (s *RedisStore) IsValid(sess *simplesessions.Session, id string) (bool, error) {
	// Validate session is valid generate string or not
	return sess.IsValidRandomString(id), nil
}

// Create returns a new session id.
func (s *RedisStore) Create(sess *simplesessions.Session) (string, error) {
	id, err := sess.GenerateRandomString(32)
	if err != nil {
		return "", err
	}

	return id, err
}

// Get gets a field in hashmap.
func (s *RedisStore) Get(sess *simplesessions.Session, id, key string) (interface{}, error) {
	conn := s.pool.Get()
	defer conn.Close()

	v, err := conn.Do("HGET", s.prefix+id, key)
	if v == nil || err == redis.ErrNil {
		return nil, simplesessions.ErrFieldNotFound
	}

	return v, err
}

// GetAll gets all fields from hashmap.
func (s *RedisStore) GetAll(sess *simplesessions.Session, id string) (map[string]interface{}, error) {
	conn := s.pool.Get()
	defer conn.Close()

	val, err := s.interfaceMap(conn.Do("HGETALL", s.prefix+id))
	if err != nil || err == redis.ErrNil {
		return nil, simplesessions.ErrFieldNotFound
	}

	return val, nil
}

// Set sets a value to given session but stored only on commit
func (s *RedisStore) Set(sess *simplesessions.Session, id, key string, val interface{}) error {
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
func (s *RedisStore) Commit(sess *simplesessions.Session, id string) error {
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
	if err != nil {
		return err
	}

	return sess.WriteCookie(id)
}

// Clear clears session in redis
func (s *RedisStore) Clear(sess *simplesessions.Session, id string) error {
	conn := s.pool.Get()
	defer conn.Close()

	_, err := conn.Do("DEL", s.prefix+id)
	if err != nil {
		return err
	}

	return sess.WriteCookie("")
}

// interfaceMap is a helper method which converts HGETALL reply to map of string interface
func (s *RedisStore) interfaceMap(result interface{}, err error) (map[string]interface{}, error) {
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
func (s *RedisStore) Int(r interface{}, err error) (int, error) {
	return redis.Int(r, err)
}

// Int64 returns redis reply as Int64.
func (s *RedisStore) Int64(r interface{}, err error) (int64, error) {
	return redis.Int64(r, err)
}

// UInt64 returns redis reply as UInt64.
func (s *RedisStore) UInt64(r interface{}, err error) (uint64, error) {
	return redis.Uint64(r, err)
}

// Float64 returns redis reply as Float64.
func (s *RedisStore) Float64(r interface{}, err error) (float64, error) {
	return redis.Float64(r, err)
}

// String returns redis reply as String.
func (s *RedisStore) String(r interface{}, err error) (string, error) {
	return redis.String(r, err)
}

// Bytes returns redis reply as Bytes.
func (s *RedisStore) Bytes(r interface{}, err error) ([]byte, error) {
	return redis.Bytes(r, err)
}

// Bool returns redis reply as Bool.
func (s *RedisStore) Bool(r interface{}, err error) (bool, error) {
	return redis.Bool(r, err)
}
