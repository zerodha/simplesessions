package goredis

import (
	"context"
	"crypto/rand"
	"sync"
	"time"
	"unicode"

	"github.com/redis/go-redis/v9"
	"github.com/vividvilla/simplesessions/conv"
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

// Create returns a new session id but doesn't stores it in redis since empty hashmap can't be created.
func (s *Store) Create() (string, error) {
	id, err := generateID(sessionIDLen)
	if err != nil {
		return "", err
	}

	return id, err
}

// Get gets a field in hashmap. If field is nill then ErrFieldNotFound is raised
func (s *Store) Get(id, key string) (interface{}, error) {
	if !validateID(id) {
		return nil, ErrInvalidSession
	}

	v, err := s.client.HGet(s.clientCtx, s.prefix+id, key).Result()
	if err == redis.Nil {
		return nil, ErrFieldNotFound
	}

	return v, err
}

// GetMulti gets a map for values for multiple keys. If key is not found then its set as nil.
func (s *Store) GetMulti(id string, keys ...string) (map[string]interface{}, error) {
	if !validateID(id) {
		return nil, ErrInvalidSession
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
func (s *Store) GetAll(id string) (map[string]interface{}, error) {
	if !validateID(id) {
		return nil, ErrInvalidSession
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
func (s *Store) Set(id, key string, val interface{}) error {
	if !validateID(id) {
		return ErrInvalidSession
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

// Commit sets all set values.
func (s *Store) Commit(id string) error {
	if !validateID(id) {
		return ErrInvalidSession
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
func (s *Store) Delete(id string, key string) error {
	if !validateID(id) {
		return ErrInvalidSession
	}

	// Clear temp map for given session id
	s.mu.Lock()
	delete(s.tempSetMap, id)
	s.mu.Unlock()

	err := s.client.HDel(s.clientCtx, s.prefix+id, key).Err()
	if err == redis.Nil {
		return ErrFieldNotFound
	}
	return err
}

// Clear clears session in redis.
func (s *Store) Clear(id string) error {
	if !validateID(id) {
		return ErrInvalidSession
	}

	return s.client.Del(s.clientCtx, s.prefix+id).Err()
}

// Int returns redis reply as integer.
func (s *Store) Int(r interface{}, err error) (int, error) {
	return conv.Int(r, err)
}

// Int64 returns redis reply as Int64.
func (s *Store) Int64(r interface{}, err error) (int64, error) {
	return conv.Int64(r, err)
}

// UInt64 returns redis reply as UInt64.
func (s *Store) UInt64(r interface{}, err error) (uint64, error) {
	return conv.UInt64(r, err)
}

// Float64 returns redis reply as Float64.
func (s *Store) Float64(r interface{}, err error) (float64, error) {
	return conv.Float64(r, err)
}

// String returns redis reply as String.
func (s *Store) String(r interface{}, err error) (string, error) {
	return conv.String(r, err)
}

// Bytes returns redis reply as Bytes.
func (s *Store) Bytes(r interface{}, err error) ([]byte, error) {
	return conv.Bytes(r, err)
}

// Bool returns redis reply as Bool.
func (s *Store) Bool(r interface{}, err error) (bool, error) {
	return conv.Bool(r, err)
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
