package postgres

/*
CREATE TABLE sessions (
    id TEXT NOT NULL PRIMARY KEY,
    data jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL
);
CREATE INDEX idx_sessions ON sessions (id, created_at);
*/

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
	"unicode"

	_ "github.com/lib/pq"
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

type queries struct {
	create *sql.Stmt
	get    *sql.Stmt
	update *sql.Stmt
	delete *sql.Stmt
	clear  *sql.Stmt
}

// Store represents redis session store for simple sessions.
// Each session is stored as redis hashmap.
type Store struct {
	db  *sql.DB
	opt Opt
	q   *queries

	commitID string
	tx       *sql.Tx
	stmt     *sql.Stmt
	sync.Mutex
}

type Opt struct {
	Table string        `json:"table"`
	TTL   time.Duration `json:"ttl"`

	// Delete expired (TTL) rows from the table at this interval.
	// This runs concurrently on a separate goroutine.
	CleanInterval time.Duration `json:"clean_interval"`
}

const (
	sessionIDLen = 32
)

// New creates a new Postgres store instance.
func New(opt Opt, db *sql.DB) (*Store, error) {
	if opt.Table == "" {
		opt.Table = "sessions"
	}
	if opt.TTL.Seconds() < 1 {
		opt.TTL = time.Hour * 24
	}
	if opt.CleanInterval.Seconds() < 1 {
		opt.CleanInterval = time.Hour * 1
	}

	st := &Store{
		db:  db,
		opt: opt,
	}

	// Prepare and keep the queries.
	q, err := st.prepareQueries()
	if err != nil {
		return nil, err
	}
	st.q = q

	return st, nil
}

// Create creates a new session and returns the ID.
func (s *Store) Create() (string, error) {
	id, err := generateID(sessionIDLen)
	if err != nil {
		return "", err
	}

	if _, err := s.q.create.Exec(id); err != nil {
		return "", err
	}
	return id, nil
}

// Get returns a single session field's value.
func (s *Store) Get(id, key string) (interface{}, error) {
	if !validateID(id) {
		return nil, ErrInvalidSession
	}

	// Scan the whole JSON map out so that it can be unmarshalled,
	// preserving the types.
	var b []byte
	if err := s.q.get.QueryRow(id, s.opt.TTL.Seconds()).Scan(&b); err != nil {
		return nil, err
	}

	var mp map[string]interface{}
	if err := json.Unmarshal(b, &mp); err != nil {
		return nil, err
	}

	v, ok := mp[key]
	if !ok {
		return nil, ErrFieldNotFound
	}

	return v, nil
}

// GetMulti gets a map for values for multiple keys. If a key doesn't exist, it returns ErrFieldNotFound.
func (s *Store) GetMulti(id string, keys ...string) (map[string]interface{}, error) {
	if !validateID(id) {
		return nil, ErrInvalidSession
	}

	vals, err := s.GetAll(id)
	if err != nil {
		return nil, err
	}

	out := make(map[string]interface{}, len(keys))
	for _, k := range keys {
		v, ok := vals[k]
		if !ok {
			return nil, ErrFieldNotFound
		}
		out[k] = v
	}

	return out, err
}

// GetAll returns the map of all keys in the session.
func (s *Store) GetAll(id string, keys ...string) (map[string]interface{}, error) {
	if !validateID(id) {
		return nil, ErrInvalidSession
	}

	var b []byte
	err := s.q.get.QueryRow(id, s.opt.TTL.Seconds()).Scan(&b)
	if err != nil {
		return nil, err
	}

	out := make(map[string]interface{})
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}

	return out, err
}

// Set sets a value to given session but is stored only on commit.
func (s *Store) Set(id, key string, val interface{}) (err error) {
	if !validateID(id) {
		return ErrInvalidSession
	}

	b, err := json.Marshal(map[string]interface{}{key: val})
	if err != nil {
		return err
	}

	s.Lock()
	defer func() {
		if err == nil {
			s.Unlock()
			return
		}

		if s.tx != nil {
			s.tx.Rollback()
			s.tx = nil
		}
		s.stmt = nil

		s.Unlock()
	}()

	// If a transaction isn't set, set it.
	if s.tx == nil {
		tx, err := s.db.Begin()
		if err != nil {
			return err
		}

		// Prepare the statement for executing SQL commands
		s.tx = tx
		s.stmt = tx.Stmt(s.q.update)
	}

	// Execute the query in the batch to be committed later.
	res, err := s.stmt.Exec(id, json.RawMessage(b))
	if err != nil {
		return err
	}
	num, err := res.RowsAffected()
	if err != nil {
		return err
	}

	// No row was updated. The session didn't exist.
	if num == 0 {
		return ErrInvalidSession
	}

	s.commitID = id
	return err
}

// Commit sets all set values
func (s *Store) Commit(id string) error {
	if !validateID(id) {
		return ErrInvalidSession
	}

	s.Lock()
	if s.commitID != id {
		s.Unlock()
		return ErrInvalidSession
	}

	defer func() {
		if s.stmt != nil {
			s.stmt.Close()
		}
		s.tx = nil
		s.stmt = nil
		s.Unlock()
	}()

	if s.tx == nil {
		return errors.New("nothing to commit")
	}
	if s.commitID != id {
		return ErrInvalidSession
	}

	return s.tx.Commit()
}

// Delete deletes a key from redis session hashmap.
func (s *Store) Delete(id string, key string) error {
	if !validateID(id) {
		return ErrInvalidSession
	}

	res, err := s.q.delete.Exec(id, key)
	if err != nil {
		return err
	}

	num, err := res.RowsAffected()
	if err != nil {
		return err
	}

	// No row was updated. The session didn't exist.
	if num == 0 {
		return ErrInvalidSession
	}

	return nil
}

// Clear clears session in redis.
func (s *Store) Clear(id string) error {
	if !validateID(id) {
		return ErrInvalidSession
	}

	res, err := s.q.clear.Exec(id)
	if err != nil {
		return err
	}

	num, err := res.RowsAffected()
	if err != nil {
		return err
	}

	// No row was updated. The session didn't exist.
	if num == 0 {
		return ErrInvalidSession
	}

	return nil
}

func (s *Store) prepareQueries() (*queries, error) {
	var (
		q   = &queries{}
		err error
	)

	q.create, err = s.db.Prepare(fmt.Sprintf("INSERT INTO %s (id, data) VALUES($1, '{}'::JSONB)", s.opt.Table))
	if err != nil {
		return nil, err
	}

	q.get, err = s.db.Prepare(fmt.Sprintf("SELECT data FROM %s WHERE id=$1 AND created_at >= NOW() - INTERVAL '1 second' * $2", s.opt.Table))
	if err != nil {
		return nil, err
	}

	q.update, err = s.db.Prepare(fmt.Sprintf("UPDATE %s SET data = data || $2::JSONB WHERE id = $1", s.opt.Table))
	if err != nil {
		return nil, err
	}

	q.delete, err = s.db.Prepare(fmt.Sprintf("UPDATE %s SET data = data - $2 WHERE id=$1", s.opt.Table))
	if err != nil {
		return nil, err
	}

	q.clear, err = s.db.Prepare(fmt.Sprintf("UPDATE %s SET data = '{}'::JSONB WHERE id=$1", s.opt.Table))
	if err != nil {
		return nil, err
	}

	return q, err
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
