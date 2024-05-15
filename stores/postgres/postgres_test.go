package postgres

// For this test to run, set env vars: PG_HOST, PG_PORT, PG_USER, PG_PASSWORD, PG_DB.

import (
	"database/sql"
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
	st        *Store
	db        *sql.DB
	randID, _ = generateID(sessionIDLen)
)

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

func TestCreate(t *testing.T) {
	for n := 0; n < 5; n++ {
		id, err := st.Create()
		assert.NoError(t, err)
		assert.NotEmpty(t, id)
	}
}

func TestSet(t *testing.T) {
	assert.NotEmpty(t, randID)

	id, err := st.Create()
	assert.NoError(t, err)
	assert.NotEmpty(t, id)

	assert.NoError(t, st.Set(id, "num", 123))
	assert.NoError(t, st.Set(id, "str", "hello 123"))
	assert.NoError(t, st.Set(id, "bool", true))

	// Commit invalid session.
	assert.Error(t, st.Commit(randID), ErrInvalidSession)

	// Commit valid session.
	assert.NoError(t, st.Commit(id))

	// Commit without setting.
	assert.Error(t, st.Commit(id))
	assert.Error(t, st.Commit(randID))

	// Get different types.
	v, err := st.Get(id, "num")
	assert.NoError(t, err)
	assert.Equal(t, v, float64(123))

	v, err = st.Get(id, "str")
	assert.NoError(t, err)
	assert.Equal(t, v, "hello 123")

	v, err = st.Get(id, "bool")
	assert.NoError(t, err)
	assert.Equal(t, v, true)

	// Non-existent field.
	_, err = st.Get(id, "xx")
	assert.ErrorIs(t, err, ErrFieldNotFound)

	// Get multiple.
	mp, err := st.GetMulti(id, "num", "str", "bool")
	assert.NoError(t, err)
	assert.Equal(t, mp, map[string]interface{}{
		"str":  "hello 123",
		"num":  float64(123),
		"bool": true,
	})
	mp, err = st.GetMulti(id, "num", "str", "bool", "blah")
	assert.ErrorIs(t, err, ErrFieldNotFound)

	// Add another key in a different commit.
	assert.NoError(t, st.Set(id, "num2", 456))
	assert.NoError(t, st.Commit(id))

	v, err = st.Get(id, "num2")
	assert.NoError(t, err)
	assert.Equal(t, v, float64(456))

	// Delete.
	assert.ErrorIs(t, st.Delete("blah", "num2"), ErrInvalidSession)
	assert.NoError(t, st.Delete(id, "num2"))
	v, err = st.Get(id, "num2")
	v, err = st.Get(id, "num3")
	assert.Error(t, ErrFieldNotFound)

	// Clear.
	assert.ErrorIs(t, st.Clear(randID), ErrInvalidSession)
	assert.NoError(t, st.Clear(id))
	v, err = st.Get(id, "str")
	assert.Error(t, err, ErrFieldNotFound)
}

func TestPrune(t *testing.T) {
	// Create a new session.
	id, err := st.Create()
	assert.NoError(t, err)
	assert.NotEmpty(t, id)

	// Set value.
	assert.NoError(t, st.Set(id, "str", "hello 123"))
	assert.NoError(t, st.Commit(id))

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
	id, err = st.Create()
	assert.NoError(t, err)
	assert.NoError(t, st.Set(id, "str", "hello 123"))
	assert.NoError(t, st.Commit(id))

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
