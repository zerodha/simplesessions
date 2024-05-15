package postgres

// For this test to run, set env vars: PG_HOST, PG_PORT, PG_USER, PG_PASSWORD, PG_DB.

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

var (
	st        *Store
	randID, _ = generateID(sessionIDLen)
)

func init() {
	if os.Getenv("PG_HOST") == "" {
		fmt.Println("WARNING: Skiping DB test as database config isn't set in env vars.")
		os.Exit(0)
	}

	p := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("PG_HOST"), os.Getenv("PG_PORT"), os.Getenv("PG_USER"), os.Getenv("PG_PASSWORD"), os.Getenv("PG_DB"))
	db, err := sql.Open("postgres", p)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	st, err = New(Opt{}, db)
	if err != nil {
		log.Fatal(err)
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
