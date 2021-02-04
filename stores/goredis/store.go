package goredis

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
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

	// Redis client
	client    redis.UniversalClient
	clientCtx context.Context
}

const (
	// Default prefix used to store session redis
	defaultPrefix = "session:"
	sessionIDLen  = 32
)

// New creates a new Redis store instance.
func New(ctx context.Context, client redis.UniversalClient) *Store {
	return &Store{
		clientCtx:  ctx,
		client:     client,
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

	v, err := s.client.HGet(s.clientCtx, s.prefix+id, key).Result()
	if err == redis.Nil {
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

	v, err := s.client.HMGet(s.clientCtx, s.prefix+id, keys...).Result()
	// If field is not found then return map with fields as nil
	if len(v) == 0 || err == redis.Nil {
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

	res, err := s.client.HGetAll(s.clientCtx, s.prefix+id).Result()
	if res == nil || err == redis.Nil {
		return map[string]interface{}{}, nil
	} else if err != nil {
		return nil, err
	}

	// Convert results to type `map[string]interface{}`
	out := make(map[string]interface{}, len(res))
	for k, v := range res {
		out[k] = v
	}

	return out, nil
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
	args := make([]interface{}, len(vals)*2, len(vals)*2)
	c := 0
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

	pipe := s.client.TxPipeline()
	pipe.HMSet(s.clientCtx, s.prefix+id, args...)
	// Set expiry of key only if 'ttl' is set, this is to
	// ensure that the key remains valid indefinitely like
	// how redis handles it by default
	if s.ttl > 0 {
		pipe.Expire(s.clientCtx, s.prefix+id, s.ttl)
	}

	_, err := pipe.Exec(s.clientCtx)
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

	err := s.client.HDel(s.clientCtx, s.prefix+id, key).Err()
	if err == redis.Nil {
		return simplesessions.ErrFieldNotFound
	}
	return err
}

// Clear clears session in redis.
func (s *Store) Clear(sess *simplesessions.Session, id string) error {
	// Check if valid session
	if !s.isValidSessionID(sess, id) {
		return simplesessions.ErrInvalidSession
	}

	return s.client.Del(s.clientCtx, s.prefix+id).Err()
}

// Int returns redis reply as integer.
func (s *Store) Int(r interface{}, err error) (int, error) {
	if err != nil {
		return 0, err
	}

	switch r := r.(type) {
	case int:
		return r, nil
	case int64:
		x := int(r)
		if int64(x) != r {
			return 0, strconv.ErrRange
		}
		return x, nil
	case []byte:
		n, err := strconv.ParseInt(string(r), 10, 0)
		return int(n), err
	case string:
		n, err := strconv.ParseInt(r, 10, 0)
		return int(n), err
	case nil:
		return 0, redis.Nil
	}

	return 0, fmt.Errorf("simplesssion: unexpected type for Int, got type %T", r)
}

// Int64 returns redis reply as Int64.
func (s *Store) Int64(r interface{}, err error) (int64, error) {
	if err != nil {
		return 0, err
	}

	switch r := r.(type) {
	case int64:
		return r, nil
	case []byte:
		n, err := strconv.ParseInt(string(r), 10, 64)
		return n, err
	case string:
		n, err := strconv.ParseInt(r, 10, 64)
		return n, err
	case nil:
		return 0, redis.Nil
	}

	return 0, fmt.Errorf("simplesssion: unexpected type for Int64, got type %T", r)
}

// UInt64 returns redis reply as UInt64.
func (s *Store) UInt64(r interface{}, err error) (uint64, error) {
	if err != nil {
		return 0, err
	}

	switch r := r.(type) {
	case uint64:
		return r, err
	case int64:
		if r < 0 {
			return 0, fmt.Errorf("simplesssion: unexpected type for Uint64")
		}
		return uint64(r), nil
	case []byte:
		n, err := strconv.ParseUint(string(r), 10, 64)
		return n, err
	case string:
		n, err := strconv.ParseUint(r, 10, 64)
		return n, err
	case nil:
		return 0, redis.Nil
	}
	return 0, fmt.Errorf("simplesssion: unexpected type for Uint64, got type %T", r)
}

// Float64 returns redis reply as Float64.
func (s *Store) Float64(r interface{}, err error) (float64, error) {
	if err != nil {
		return 0, err
	}
	switch r := r.(type) {
	case float64:
		return r, err
	case []byte:
		n, err := strconv.ParseFloat(string(r), 64)
		return n, err
	case string:
		n, err := strconv.ParseFloat(r, 64)
		return n, err
	case nil:
		return 0, redis.Nil
	}
	return 0, fmt.Errorf("simplesssion: unexpected type for Float64, got type %T", r)
}

// String returns redis reply as String.
func (s *Store) String(r interface{}, err error) (string, error) {
	if err != nil {
		return "", err
	}
	switch r := r.(type) {
	case []byte:
		return string(r), nil
	case string:
		return r, nil
	case nil:
		return "", redis.Nil
	}
	return "", fmt.Errorf("simplesssion: unexpected type for String, got type %T", r)
}

// Bytes returns redis reply as Bytes.
func (s *Store) Bytes(r interface{}, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	switch r := r.(type) {
	case []byte:
		return r, nil
	case string:
		return []byte(r), nil
	case nil:
		return nil, redis.Nil
	}
	return nil, fmt.Errorf("simplesssion: unexpected type for Bytes, got type %T", r)
}

// Bool returns redis reply as Bool.
func (s *Store) Bool(r interface{}, err error) (bool, error) {
	if err != nil {
		return false, err
	}
	switch r := r.(type) {
	case bool:
		return r, err
	// Very common in redis to reply int64 with 0 for bool flag.
	case int64:
		return r != 0, nil
	case []byte:
		return strconv.ParseBool(string(r))
	case string:
		return strconv.ParseBool(r)
	case nil:
		return false, redis.Nil
	}
	return false, fmt.Errorf("simplesssion: unexpected type for Bool, got type %T", r)
}
