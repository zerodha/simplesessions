package redis

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
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

// Store represents redis session store for simple sessions.
// Each session is stored as redis hashmap.
type Store struct {
	// Maximum lifetime sessions has to be persisted.
	ttl time.Duration
	// extend TTL on update.
	extendTTL bool

	// Prefix for session id.
	prefix string

	// Redis client
	client    redis.UniversalClient
	clientCtx context.Context
}

const (
	// Default prefix used to store session redis
	defaultPrefix = "session:"
	// Default key used when session is created.
	// Its not possible to have empty map in Redis.
	defaultSessKey = "_ss"
)

// New creates a new Redis store instance.
func New(ctx context.Context, client redis.UniversalClient) *Store {
	return &Store{
		clientCtx: ctx,
		client:    client,
		prefix:    defaultPrefix,
	}
}

// SetPrefix sets session id prefix in backend
func (s *Store) SetPrefix(val string) {
	s.prefix = val
}

// SetTTL sets TTL for session in redis.
// if isExtend is true then ttl is updated on all set/setmulti.
// otherwise its set only on create().
func (s *Store) SetTTL(d time.Duration, extend bool) {
	s.ttl = d
	s.extendTTL = extend
}

// Create returns a new session id but doesn't stores it in redis since empty hashmap can't be created.
func (s *Store) Create(id string) error {
	// Create the session in backend with default session key since
	// Redis doesn't support empty hashmap and its impossible to
	// check if the session exist or not.
	p := s.client.TxPipeline()
	p.HSet(s.clientCtx, s.prefix+id, defaultSessKey, "1")
	if s.ttl > 0 {
		p.Expire(s.clientCtx, s.prefix+id, s.ttl)
	}
	_, err := p.Exec(s.clientCtx)
	return err
}

// Get gets a field in hashmap. If field is nill then ErrFieldNotFound is raised
func (s *Store) Get(id, key string) (interface{}, error) {
	vals, err := s.client.HMGet(s.clientCtx, s.prefix+id, defaultSessKey, key).Result()
	if err != nil {
		return nil, err
	}

	if vals[0] == nil {
		return nil, ErrInvalidSession
	}

	return vals[1], nil
}

// GetMulti gets a map for values for multiple keys. If key is not found then its set as nil.
func (s *Store) GetMulti(id string, keys ...string) (map[string]interface{}, error) {
	allKeys := append([]string{defaultSessKey}, keys...)
	vals, err := s.client.HMGet(s.clientCtx, s.prefix+id, allKeys...).Result()
	if err != nil {
		return nil, err
	}

	if vals[0] == nil {
		return nil, ErrInvalidSession
	}

	// Form a map with returned results
	res := make(map[string]interface{})
	for i, k := range allKeys {
		if k != defaultSessKey {
			res[k] = vals[i]
		}
	}

	return res, err
}

// GetAll gets all fields from hashmap.
func (s *Store) GetAll(id string) (map[string]interface{}, error) {
	vals, err := s.client.HGetAll(s.clientCtx, s.prefix+id).Result()
	if err != nil {
		return nil, err
	}

	// Convert results to type `map[string]interface{}`
	out := make(map[string]interface{})
	for k, v := range vals {
		if k != defaultSessKey {
			out[k] = v
		}
	}

	return out, nil
}

// Set sets a value to given session.
// If session is not present in backend then its still written.
func (s *Store) Set(id, key string, val interface{}) error {
	p := s.client.TxPipeline()
	p.HSet(s.clientCtx, s.prefix+id, key, val)
	p.HSet(s.clientCtx, s.prefix+id, defaultSessKey, "1")

	// Set expiry of key only if 'ttl' is set, this is to
	// ensure that the key remains valid indefinitely like
	// how redis handles it by default
	if s.ttl > 0 && s.extendTTL {
		p.Expire(s.clientCtx, s.prefix+id, s.ttl)
	}

	_, err := p.Exec(s.clientCtx)
	return err
}

// Set sets a value to given session.
func (s *Store) SetMulti(id string, data map[string]interface{}) error {
	// Make slice of arguments to be passed in HGETALL command
	args := []interface{}{defaultSessKey, "1"}
	for k, v := range data {
		args = append(args, k, v)
	}

	p := s.client.TxPipeline()
	p.HMSet(s.clientCtx, s.prefix+id, args...)
	// Set expiry of key only if 'ttl' is set, this is to
	// ensure that the key remains valid indefinitely like
	// how redis handles it by default
	if s.ttl > 0 && s.extendTTL {
		p.Expire(s.clientCtx, s.prefix+id, s.ttl)
	}

	_, err := p.Exec(s.clientCtx)
	return err
}

// Delete deletes a key from redis session hashmap.
func (s *Store) Delete(id string, key string) error {
	return s.client.HDel(s.clientCtx, s.prefix+id, key).Err()
}

// Clear clears session in redis.
func (s *Store) Clear(id string) error {
	return s.client.Del(s.clientCtx, s.prefix+id).Err()
}

// Int converts interface to integer.
func (s *Store) Int(r interface{}, err error) (int, error) {
	if err != nil {
		return 0, err
	}

	switch r := r.(type) {
	case int:
		return r, nil
	case int64:
		if x := int(r); int64(x) != r {
			return 0, ErrAssertType
		} else {
			return x, nil
		}
	case []byte:
		if n, err := strconv.ParseInt(string(r), 10, 0); err != nil {
			return 0, ErrAssertType
		} else {
			return int(n), nil
		}
	case string:
		if n, err := strconv.ParseInt(r, 10, 0); err != nil {
			return 0, ErrAssertType
		} else {
			return int(n), nil
		}
	case nil:
		return 0, ErrNil
	case error:
		return 0, r
	}

	return 0, ErrAssertType
}

// Int64 converts interface to Int64.
func (s *Store) Int64(r interface{}, err error) (int64, error) {
	if err != nil {
		return 0, err
	}

	switch r := r.(type) {
	case int:
		return int64(r), nil
	case int64:
		return r, nil
	case []byte:
		if n, err := strconv.ParseInt(string(r), 10, 64); err != nil {
			return 0, ErrAssertType
		} else {
			return n, nil
		}
	case string:
		if n, err := strconv.ParseInt(r, 10, 64); err != nil {
			return 0, ErrAssertType
		} else {
			return n, nil
		}
	case nil:
		return 0, ErrNil
	case error:
		return 0, r
	}

	return 0, ErrAssertType
}

// UInt64 converts interface to UInt64.
func (s *Store) UInt64(r interface{}, err error) (uint64, error) {
	if err != nil {
		return 0, err
	}

	switch r := r.(type) {
	case uint64:
		return r, nil
	case int:
		if r < 0 {
			return 0, ErrAssertType
		}
		return uint64(r), nil
	case int64:
		if r < 0 {
			return 0, ErrAssertType
		}
		return uint64(r), nil
	case []byte:
		if n, err := strconv.ParseUint(string(r), 10, 64); err != nil {
			return 0, ErrAssertType
		} else {
			return n, nil
		}
	case string:
		if n, err := strconv.ParseUint(r, 10, 64); err != nil {
			return 0, ErrAssertType
		} else {
			return n, nil
		}
	case nil:
		return 0, ErrNil
	case error:
		return 0, r
	}

	return 0, ErrAssertType
}

// Float64 converts interface to Float64.
func (s *Store) Float64(r interface{}, err error) (float64, error) {
	if err != nil {
		return 0, err
	}
	switch r := r.(type) {
	case float64:
		return r, err
	case []byte:
		if n, err := strconv.ParseFloat(string(r), 64); err != nil {
			return 0, ErrAssertType
		} else {
			return n, nil
		}
	case string:
		if n, err := strconv.ParseFloat(r, 64); err != nil {
			return 0, ErrAssertType
		} else {
			return n, nil
		}
	case nil:
		return 0, ErrNil
	case error:
		return 0, r
	}
	return 0, ErrAssertType
}

// String converts interface to String.
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
		return "", ErrNil
	case error:
		return "", r
	}
	return "", ErrAssertType
}

// Bytes converts interface to Bytes.
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
		return nil, ErrNil
	case error:
		return nil, r
	}
	return nil, ErrAssertType
}

// Bool converts interface to Bool.
func (s *Store) Bool(r interface{}, err error) (bool, error) {
	if err != nil {
		return false, err
	}
	switch r := r.(type) {
	case bool:
		return r, err
	// Very common in redis to reply int64 with 0 for bool flag.
	case int:
		return r != 0, nil
	case int64:
		return r != 0, nil
	case []byte:
		if n, err := strconv.ParseBool(string(r)); err != nil {
			return false, ErrAssertType
		} else {
			return n, nil
		}
	case string:
		if n, err := strconv.ParseBool(r); err != nil {
			return false, ErrAssertType
		} else {
			return n, nil
		}
	case nil:
		return false, ErrNil
	case error:
		return false, r
	}
	return false, ErrAssertType
}
