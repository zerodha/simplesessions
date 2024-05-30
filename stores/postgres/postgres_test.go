package postgres

// For this test to run, set env vars: PG_HOST, PG_PORT, PG_USER, PG_PASSWORD, PG_DB.

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

const testTable = "sessions"

var (
	st *Store
	db *sql.DB
)

func generateID() (string, error) {
	const dict = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	for k, v := range bytes {
		bytes[k] = dict[v%byte(len(dict))]
	}

	return string(bytes), nil
}

func init() {
	if os.Getenv("PG_HOST") == "" {
		fmt.Println("WARNING: Skiping DB test as database config isn't set in env vars.")
		os.Exit(0)
	}

	p := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("PG_HOST"), os.Getenv("PG_PORT"), os.Getenv("PG_USER"), os.Getenv("PG_PASSWORD"), os.Getenv("PG_DB"))
	if d, err := sql.Open("postgres", p); err != nil {
		log.Fatal(err)
	} else {
		db = d
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	if s, err := New(Opt{TTL: time.Second * 2, Table: testTable}, db); err != nil {
		log.Fatal(err)
	} else {
		st = s
	}
}

func TestNew(t *testing.T) {
	s1, err := New(Opt{}, db)
	assert.Nil(t, err)
	assert.Equal(t, s1.opt.Table, "sessions")
	assert.Equal(t, s1.opt.TTL, time.Hour*24)

	_, err = New(Opt{Table: "unknown"}, db)
	assert.Error(t, err)
}

func TestCreate(t *testing.T) {
	id, _ := generateID()
	err := st.Create(id)
	assert.NoError(t, err)

	var data []byte
	err = db.QueryRow(fmt.Sprintf("SELECT data FROM %s WHERE id=$1", testTable), id).Scan(&data)
	assert.NoError(t, err)
	assert.Equal(t, []byte("{}"), data)
}

func TestAll(t *testing.T) {
	id, _ := generateID()

	err := st.Create(id)
	assert.NoError(t, err)

	assert.NoError(t, st.Set(id, "num", 123))
	assert.NoError(t, st.Set(id, "float", 12.3))
	assert.NoError(t, st.Set(id, "str", "hello 123"))
	assert.NoError(t, st.Set(id, "bool", true))

	// Get different types.
	v, err := st.Get(id, "num")
	assert.NoError(t, err)
	assert.Equal(t, v, float64(123))

	{
		v, err := st.Int(st.Get(id, "num"))
		assert.NoError(t, err)
		assert.Equal(t, v, int(123))

		_, err = st.Int("xxx", nil)
		assert.ErrorIs(t, err, ErrAssertType)

		cErr := errors.New("type error")
		_, err = st.Int("xxx", cErr)
		assert.ErrorIs(t, err, cErr)
	}

	{
		v, err := st.Int64(st.Get(id, "num"))
		assert.NoError(t, err)
		assert.Equal(t, v, int64(123))

		_, err = st.Int64("xxx", nil)
		assert.ErrorIs(t, err, ErrAssertType)

		cErr := errors.New("type error")
		_, err = st.Int64("xxx", cErr)
		assert.ErrorIs(t, err, cErr)
	}

	{
		v, err := st.UInt64(st.Get(id, "num"))
		assert.NoError(t, err)
		assert.Equal(t, v, uint64(123))

		_, err = st.UInt64("xxx", nil)
		assert.ErrorIs(t, err, ErrAssertType)

		cErr := errors.New("type error")
		_, err = st.UInt64("xxx", cErr)
		assert.ErrorIs(t, err, cErr)
	}

	{
		v, err := st.Float64(st.Get(id, "float"))
		assert.NoError(t, err)
		assert.Equal(t, v, float64(12.3))

		_, err = st.Float64("xxx", nil)
		assert.ErrorIs(t, err, ErrAssertType)

		cErr := errors.New("type error")
		_, err = st.Float64("xxx", cErr)
		assert.ErrorIs(t, err, cErr)
	}

	{
		v, err := st.String(st.Get(id, "str"))
		assert.NoError(t, err)
		assert.Equal(t, v, "hello 123")

		_, err = st.String(1, nil)
		assert.ErrorIs(t, err, ErrAssertType)

		cErr := errors.New("type error")
		_, err = st.String("xxx", cErr)
		assert.ErrorIs(t, err, cErr)
	}

	{
		v, err := st.Bytes(st.Get(id, "str"))
		assert.NoError(t, err)
		assert.Equal(t, v, []byte("hello 123"))

		_, err = st.Bytes(1, nil)
		assert.ErrorIs(t, err, ErrAssertType)

		cErr := errors.New("type error")
		_, err = st.Bytes("xxx", cErr)
		assert.ErrorIs(t, err, cErr)
	}

	{
		v, err := st.Bool(st.Get(id, "bool"))
		assert.NoError(t, err)
		assert.Equal(t, v, true)

		_, err = st.Bool("xxx", nil)
		assert.ErrorIs(t, err, ErrAssertType)

		cErr := errors.New("type error")
		_, err = st.Bool("xxx", cErr)
		assert.ErrorIs(t, err, cErr)
	}

	{
		v, err := st.Get(id, "str")
		assert.NoError(t, err)
		assert.Equal(t, v, "hello 123")
	}

	{
		v, err := st.Get(id, "bool")
		assert.NoError(t, err)
		assert.Equal(t, v, true)
	}

	// Non-existent field.
	v, err = st.Get(id, "xx")
	assert.Nil(t, v)
	assert.Nil(t, err)

	// Get multiple.
	mp, err := st.GetMulti(id, "num", "str", "bool")
	assert.NoError(t, err)
	assert.Equal(t, mp, map[string]interface{}{
		"str":  "hello 123",
		"num":  float64(123),
		"bool": true,
	})
	mp, err = st.GetMulti(id, "num", "str", "bool", "blah")
	assert.Nil(t, mp["blah"])
	assert.Nil(t, err)

	// Add another key in a different commit.
	assert.NoError(t, st.Set(id, "num2", 456))

	assert.NoError(t, st.SetMulti(id, map[string]interface{}{
		"num10": 1,
		"num11": 2,
	}))

	v, err = st.Get(id, "num2")
	assert.NoError(t, err)
	assert.Equal(t, v, float64(456))

	v, err = st.Get(id, "num10")
	assert.NoError(t, err)
	assert.Equal(t, v, float64(1))

	v, err = st.Get(id, "num11")
	assert.NoError(t, err)
	assert.Equal(t, v, float64(2))

	// Delete.
	assert.ErrorIs(t, st.Delete("blah", "num2"), ErrInvalidSession)
	assert.NoError(t, st.Delete(id, "num2"))
	v, err = st.Get(id, "num2")
	assert.Nil(t, v)
	assert.Nil(t, err)
	v, err = st.Get(id, "num3")
	assert.Nil(t, v)
	assert.Nil(t, err)

	// Clear.
	assert.ErrorIs(t, st.Clear("unknow_id"), ErrInvalidSession)
	assert.NoError(t, st.Clear(id))
	v, err = st.Get(id, "str")
	assert.Nil(t, v)
	assert.Nil(t, err)

	// Destroy.
	assert.ErrorIs(t, st.Destroy("unknow_id"), ErrInvalidSession)
	assert.NoError(t, st.Destroy(id))
	_, err = st.Get(id, "str")
	assert.ErrorIs(t, err, ErrInvalidSession)
}

func TestPrune(t *testing.T) {
	id, _ := generateID()

	// Create a new session.
	err := st.Create(id)
	assert.NoError(t, err)

	// Set value.
	assert.NoError(t, st.Set(id, "str", "hello 123"))

	// Get value and verify.
	v, err := st.Get(id, "str")
	assert.NoError(t, err)
	assert.Equal(t, v, "hello 123")

	// Wait until the 2 sec TTL expires and run prune.
	time.Sleep(time.Second * 3)

	// Session shouldn't be returned.
	_, err = st.Get(id, "str")
	assert.ErrorIs(t, err, ErrInvalidSession)

	// Create one more session and immediately run prune. Except for this,
	// all previous sessions should be gone.
	id, _ = generateID()
	err = st.Create(id)
	assert.NoError(t, err)
	assert.NoError(t, st.Set(id, "str", "hello 123"))

	// Run prune. All previously created sessions should be gone.
	assert.NoError(t, st.Prune())

	var num int
	err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", testTable)).Scan(&num)
	assert.NoError(t, err)
	assert.Equal(t, num, 1)

	// The last created session shouldn't have been pruned.
	v, err = st.Get(id, "str")
	assert.NoError(t, err)
	assert.Equal(t, v, "hello 123")

}

func TestError(t *testing.T) {
	err := Err{
		code: 1,
		msg:  "test",
	}
	assert.Equal(t, 1, err.Code())
	assert.Equal(t, "test", err.Error())
}
