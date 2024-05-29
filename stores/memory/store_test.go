package memory

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	assert := assert.New(t)
	str := New()
	assert.NotNil(str.sessions)
}

func TestCreate(t *testing.T) {
	var (
		id  = "testid"
		str = New()
	)
	assert.NotContains(t, str.sessions, id)
	err := str.Create(id)
	assert.NoError(t, err)
	assert.NotNil(t, str.sessions, id)

	// Check if existing session is not overwritten on Create.
	val := map[string]interface{}{"foo": "bar"}
	str.sessions["existing_id"] = val
	err = str.Create("existing_id")
	assert.NoError(t, err)
	assert.Equal(t, val, str.sessions["existing_id"])
}

func TestGet(t *testing.T) {
	var (
		id    = "testid"
		field = "somekey"
		value = 100
		str   = New()
	)

	_, err := str.Get("invalidkey", "invalidkey")
	assert.ErrorIs(t, ErrInvalidSession, err)

	str.sessions[id] = make(map[string]interface{})
	str.sessions[id][field] = value

	val, err := str.Get(id, field)
	assert.NoError(t, err)
	assert.Equal(t, val, value)

	val, err = str.Get(id, "invalid")
	assert.NoError(t, err)
	assert.Nil(t, val)
}

func TestGetMulti(t *testing.T) {
	str := New()
	_, err := str.GetMulti("invalidkey", "invalidkey1", "invalidkey2")
	assert.ErrorIs(t, ErrInvalidSession, err)

	var (
		id     = "testid"
		field1 = "somekey"
		value1 = 100
		field2 = "someotherkey"
		value2 = "abc123"
		field3 = "thishouldntbethere"
	)

	// Set a key
	str.sessions[id] = make(map[string]interface{})
	str.sessions[id][field1] = value1
	str.sessions[id][field2] = value2

	vals, err := str.GetMulti(id, field1, field2, field3)
	assert.NoError(t, err)
	assert.Contains(t, vals, field1)
	assert.Equal(t, value1, vals[field1])

	assert.Contains(t, vals, field2)
	assert.Equal(t, value2, vals[field2])

	assert.Contains(t, vals, field3)
	assert.Nil(t, vals[field3])
}

func TestGetAll(t *testing.T) {
	str := New()
	_, err := str.GetAll("invalidkey")
	assert.ErrorIs(t, ErrInvalidSession, err)

	var (
		key    = "testid"
		field1 = "somekey"
		value1 = 100
		field2 = "someotherkey"
		value2 = "abc123"
		field3 = "thishouldntbethere"
	)

	// Set a key
	str.sessions[key] = make(map[string]interface{})
	str.sessions[key][field1] = value1
	str.sessions[key][field2] = value2

	vals, err := str.GetAll(key)
	assert.NoError(t, err)
	assert.Contains(t, vals, field1)
	assert.Contains(t, vals, field2)
	assert.NotContains(t, vals, field3)

	assert.Equal(t, value1, vals[field1])
	assert.Equal(t, value2, vals[field2])
}

func TestSet(t *testing.T) {
	str := New()
	err := str.Set("invalidkey", "key", "val")
	assert.ErrorIs(t, ErrInvalidSession, err)

	// this id is unique across all tests
	var (
		id    = "testid"
		field = "somekey"
		value = 100
	)
	assert.NotContains(t, str.sessions, id)

	str.sessions[id] = map[string]interface{}{
		field: value,
	}
	err = str.Set(id, field, value)
	assert.NoError(t, err)
	assert.Contains(t, str.sessions, id)
	assert.Contains(t, str.sessions[id], field)
	assert.Equal(t, value, str.sessions[id][field])
}

func TestSetMulti(t *testing.T) {
	str := New()
	err := str.SetMulti("invalidkey", map[string]interface{}{"foo": "bar"})
	assert.ErrorIs(t, ErrInvalidSession, err)

	// this id is unique across all tests
	var (
		id     = "testid"
		field1 = "somekey1"
		value1 = 100
		field2 = "somekey2"
		value2 = 100
	)
	str.sessions[id] = map[string]interface{}{}
	err = str.SetMulti(id, map[string]interface{}{
		field1: value1,
		field2: value2,
	})
	assert.NoError(t, err)
	assert.Contains(t, str.sessions, id)
	assert.Contains(t, str.sessions[id], field1)
	assert.Contains(t, str.sessions[id], field2)
	assert.Equal(t, value1, str.sessions[id][field1])
	assert.Equal(t, value2, str.sessions[id][field2])
}

func TestDelete(t *testing.T) {
	// Test should only set in internal map and not in redis
	str := New()
	err := str.Delete("invalidkey", "key")
	assert.ErrorIs(t, ErrInvalidSession, err)

	// this key is unique across all tests
	var (
		key    = "8dIHy6S2uBuKaNnTUszB2180898ikGY1"
		field1 = "somefield1"
		field2 = "somefield2"
	)
	str.sessions[key] = make(map[string]interface{})
	str.sessions[key][field1] = 10
	str.sessions[key][field2] = 10

	err = str.Delete(key, field1)
	assert.NoError(t, err)
	assert.Contains(t, str.sessions[key], field2)
	assert.NotContains(t, str.sessions[key], field1)
}

func TestClear(t *testing.T) {
	// Test should only set in internal map and not in redis
	str := New()
	err := str.Clear("invalidkey")
	assert.ErrorIs(t, ErrInvalidSession, err)

	// this id is unique across all tests
	id := "test_id"
	str.sessions[id] = make(map[string]interface{})

	err = str.Clear(id)
	assert.NoError(t, err)
	assert.NotContains(t, str.sessions, id)
}

func TestInt(t *testing.T) {
	str := New()

	var want int = 10
	v, err := str.Int(want, nil)
	assert.Nil(t, err)
	assert.Equal(t, v, want)

	testError := errors.New("test error")
	v, err = str.Int(want, testError)
	assert.Equal(t, v, 0)
	assert.ErrorIs(t, testError, err)

	_, err = str.Int("string", nil)
	assert.ErrorIs(t, ErrAssertType, err)
}

func TestInt64(t *testing.T) {
	str := New()

	var want int64 = 10
	v, err := str.Int64(want, nil)
	assert.Nil(t, err)
	assert.Equal(t, v, want)

	testError := errors.New("test error")
	v, err = str.Int64(want, testError)
	assert.Equal(t, v, int64(0))
	assert.ErrorIs(t, testError, err)

	_, err = str.Int64("string", nil)
	assert.ErrorIs(t, ErrAssertType, err)
}

func TestUInt64(t *testing.T) {
	str := New()

	var want uint64 = 10
	v, err := str.UInt64(want, nil)
	assert.Nil(t, err)
	assert.Equal(t, v, want)

	testError := errors.New("test error")
	v, err = str.UInt64(want, testError)
	assert.Equal(t, v, uint64(0))
	assert.ErrorIs(t, testError, err)

	_, err = str.UInt64("string", nil)
	assert.ErrorIs(t, ErrAssertType, err)
}

func TestFloat64(t *testing.T) {
	str := New()

	var want float64 = 10
	v, err := str.Float64(want, nil)
	assert.Nil(t, err)
	assert.Equal(t, v, want)

	testError := errors.New("test error")
	v, err = str.Float64(want, testError)
	assert.Equal(t, v, float64(0))
	assert.ErrorIs(t, testError, err)

	_, err = str.Float64("string", nil)
	assert.ErrorIs(t, ErrAssertType, err)
}

func TestString(t *testing.T) {
	str := New()

	var want = "string"
	v, err := str.String(want, nil)
	assert.Nil(t, err)
	assert.Equal(t, v, want)

	testError := errors.New("test error")
	v, err = str.String(want, testError)
	assert.Equal(t, v, "")
	assert.ErrorIs(t, testError, err)

	_, err = str.String(123, nil)
	assert.Error(t, ErrAssertType, err)
}

func TestBytes(t *testing.T) {
	str := New()

	var want = []byte("a")
	v, err := str.Bytes(want, nil)
	assert.Nil(t, err)
	assert.Equal(t, v, want)

	testError := errors.New("test error")
	v, err = str.Bytes(want, testError)
	assert.Equal(t, v, []byte(nil))
	assert.ErrorIs(t, testError, err)

	_, err = str.Bytes("string", nil)
	assert.ErrorIs(t, ErrAssertType, err)
}

func TestBool(t *testing.T) {
	str := New()

	var want = true
	v, err := str.Bool(want, nil)
	assert.Nil(t, err)
	assert.Equal(t, v, want)

	testError := errors.New("test error")
	v, err = str.Bool(want, testError)
	assert.Equal(t, v, false)
	assert.ErrorIs(t, testError, err)

	_, err = str.Bool("string", nil)
	assert.Error(t, ErrAssertType, err)
}

func TestError(t *testing.T) {
	err := Err{
		code: 1,
		msg:  "test",
	}
	assert.Equal(t, 1, err.Code())
	assert.Equal(t, "test", err.Error())
}
