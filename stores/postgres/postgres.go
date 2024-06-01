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
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
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

type queries struct {
	create  *sql.Stmt
	get     *sql.Stmt
	update  *sql.Stmt
	delete  *sql.Stmt
	clear   *sql.Stmt
	prune   *sql.Stmt
	destroy *sql.Stmt
}

// Store represents redis session store for simple sessions.
// Each session is stored as redis hashmap.
type Store struct {
	db  *sql.DB
	opt Opt
	q   *queries
}

type Opt struct {
	Table string        `json:"table"`
	TTL   time.Duration `json:"ttl"`

	// Delete expired (TTL) rows from the table at this interval.
	// This runs concurrently on a separate goroutine.
	CleanInterval time.Duration `json:"clean_interval"`
}

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
func (s *Store) Create(id string) error {
	_, err := s.q.create.Exec(id)
	return err
}

// Get returns a single session field's value.
func (s *Store) Get(id, key string) (interface{}, error) {
	vals, err := s.GetAll(id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrInvalidSession
		}
		return nil, err
	}

	v, ok := vals[key]
	if !ok {
		return nil, nil
	}

	return v, nil
}

// GetMulti gets a map for values for multiple keys. If a key doesn't exist, it returns ErrFieldNotFound.
func (s *Store) GetMulti(id string, keys ...string) (map[string]interface{}, error) {
	vals, err := s.GetAll(id)
	if err != nil {
		return nil, err
	}

	out := make(map[string]interface{}, len(keys))
	for _, k := range keys {
		v, ok := vals[k]
		if !ok {
			return nil, nil
		}
		out[k] = v
	}

	return out, err
}

// GetAll returns the map of all keys in the session.
func (s *Store) GetAll(id string) (map[string]interface{}, error) {
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
	b, err := json.Marshal(map[string]interface{}{key: val})
	if err != nil {
		return err
	}

	// Execute the query in the batch to be committed later.
	res, err := s.q.update.Exec(id, json.RawMessage(b))
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

	return err
}

// Set sets a value to given session but is stored only on commit.
func (s *Store) SetMulti(id string, data map[string]interface{}) (err error) {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Execute the query in the batch to be committed later.
	res, err := s.q.update.Exec(id, json.RawMessage(b))
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

	return err
}

// Delete deletes a key from redis session hashmap.
func (s *Store) Delete(id string, keys ...string) error {
	res, err := s.q.delete.Exec(id, pq.Array(keys))
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

// Destroy deletes the entire session from backend.
func (s *Store) Destroy(id string) error {
	res, err := s.q.destroy.Exec(id)
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

// Int is a helper method to type assert as integer.
func (s *Store) Int(r interface{}, err error) (int, error) {
	if err != nil {
		return 0, err
	}

	v, ok := r.(float64)
	if !ok {
		return 0, ErrAssertType
	}

	return int(v), err
}

// Int64 is a helper method to type assert as Int64
func (s *Store) Int64(r interface{}, err error) (int64, error) {
	if err != nil {
		return 0, err
	}

	v, ok := r.(float64)
	if !ok {
		return 0, ErrAssertType
	}

	return int64(v), err
}

// UInt64 is a helper method to type assert as UInt64
func (s *Store) UInt64(r interface{}, err error) (uint64, error) {
	if err != nil {
		return 0, err
	}

	v, ok := r.(float64)
	if !ok {
		return 0, ErrAssertType
	}

	return uint64(v), err
}

// Float64 is a helper method to type assert as Float64
func (s *Store) Float64(r interface{}, err error) (float64, error) {
	if err != nil {
		return 0, err
	}

	v, ok := r.(float64)
	if !ok {
		return 0, ErrAssertType
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
		return "", ErrAssertType
	}

	return v, err
}

// Bytes is a helper method to type assert as Bytes
func (s *Store) Bytes(r interface{}, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}

	v, ok := r.(string)
	if !ok {
		return nil, ErrAssertType
	}

	return []byte(v), err
}

// Bool is a helper method to type assert as Bool
func (s *Store) Bool(r interface{}, err error) (bool, error) {
	if err != nil {
		return false, err
	}

	v, ok := r.(bool)
	if !ok {
		return false, ErrAssertType
	}

	return v, nil
}

// Prune deletes rows that have exceeded the TTL. This should be run externally periodically (ideally as a separate goroutine)
// at desired intervals, hourly/daily etc. based on the expected volume of sessions.
func (s *Store) Prune() error {
	_, err := s.q.prune.Exec(s.opt.TTL.Seconds())
	return err
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

	q.delete, err = s.db.Prepare(fmt.Sprintf("UPDATE %s SET data = data #- $2 WHERE id=$1", s.opt.Table))
	if err != nil {
		return nil, err
	}

	q.clear, err = s.db.Prepare(fmt.Sprintf("UPDATE %s SET data = '{}'::JSONB WHERE id=$1", s.opt.Table))
	if err != nil {
		return nil, err
	}

	q.prune, err = s.db.Prepare(fmt.Sprintf("DELETE FROM %s WHERE created_at <= NOW() - INTERVAL '1 second' * $1", s.opt.Table))
	if err != nil {
		return nil, err
	}

	q.destroy, err = s.db.Prepare(fmt.Sprintf("DELETE FROM %s WHERE id=$1", s.opt.Table))
	if err != nil {
		return nil, err
	}

	return q, err
}
