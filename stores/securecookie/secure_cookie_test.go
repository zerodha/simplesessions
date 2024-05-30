package securecookie

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	secretKey = []byte("0dIHy6S2uBuKaNnTUszB218L898ikGYA")
	blockKey  = []byte("0dIHy6S2uBuKaNnTUszB218L898ikGYA")
)

func TestNew(t *testing.T) {
	str := New(secretKey, blockKey)

	assert.NotNil(t, str.sc)
	assert.NotNil(t, str.tempSetMap)
}

func TestSetCookieName(t *testing.T) {
	str := New(secretKey, blockKey)
	assert.Equal(t, defaultCookieName, str.cookieName)

	str.SetCookieName("csrftoken")
	assert.Equal(t, "csrftoken", str.cookieName)
}

func TestIsValid(t *testing.T) {
	str := New(secretKey, blockKey)
	assert.False(t, str.IsValid(""))

	encoded, err := str.encode(make(map[string]interface{}))
	assert.Nil(t, err)
	assert.True(t, str.IsValid(encoded))
}

func TestCreate(t *testing.T) {
	str := New(secretKey, blockKey)

	err := str.Create("testid")
	assert.Nil(t, err)
	assert.Contains(t, str.tempSetMap, "testid")
	assert.Equal(t, 0, len(str.tempSetMap["testid"]))
}

func TestGet(t *testing.T) {
	str := New(secretKey, blockKey)
	val, err := str.Get("invalidkey", "invalidkey")
	assert.Nil(t, val)
	assert.ErrorIs(t, err, ErrInvalidSession)

	var (
		field = "somekey"
		value = 100
		m     = map[string]interface{}{
			field: value,
		}
	)
	cv, err := str.encode(m)
	assert.Nil(t, err)

	val, err = str.Get(cv, field)
	assert.NoError(t, err)
	assert.Equal(t, val, value)

	val, err = str.Get(cv, "invalid")
	assert.NoError(t, err)
	assert.Equal(t, nil, val)
}

func TestGetMulti(t *testing.T) {
	str := New(secretKey, blockKey)
	val, err := str.GetMulti("invalidkey", "invalidkey")
	assert.Nil(t, val)
	assert.ErrorIs(t, err, ErrInvalidSession)

	var (
		field1 = "somekey"
		value1 = 100
		field2 = "someotherkey"
		value2 = "abc123"
		field3 = "thishouldntbethere"
	)

	// Set a key
	m := map[string]interface{}{
		field1: value1,
		field2: value2,
	}
	cv, err := str.encode(m)
	assert.Nil(t, err)

	vals, err := str.GetMulti(cv, field1, field2, field3)
	assert.NoError(t, err)
	assert.Contains(t, vals, field1)
	assert.Contains(t, vals, field2)
	assert.Contains(t, vals, field3)
	assert.Equal(t, vals[field1], value1)
	assert.Equal(t, vals[field2], value2)
	assert.Equal(t, vals[field3], nil)
}

func TestGetAll(t *testing.T) {
	str := New(secretKey, blockKey)

	val, err := str.GetAll("invalidkey")
	assert.Nil(t, val)
	assert.ErrorIs(t, err, ErrInvalidSession)

	var (
		field1 = "somekey"
		value1 = 100
		field2 = "someotherkey"
		value2 = "abc123"
	)

	// Set a key
	m := map[string]interface{}{
		field1: value1,
		field2: value2,
	}
	cv, err := str.encode(m)
	assert.Nil(t, err)

	vals, err := str.GetAll(cv)
	assert.NoError(t, err)
	assert.Contains(t, vals, field1)
	assert.Contains(t, vals, field2)
	assert.Equal(t, vals[field1], value1)
	assert.Equal(t, vals[field2], value2)
}

func TestSet(t *testing.T) {
	// Test should only set in internal map and not in redis
	var (
		str = New(secretKey, blockKey)

		// this key is unique across all tests
		field = "somekey"
		value = 100
	)
	m := map[string]interface{}{
		field: value,
	}
	cv, err := str.encode(m)
	assert.Nil(t, err)

	err = str.Set(cv, field, value)
	assert.NoError(t, err)
	assert.Contains(t, str.tempSetMap, cv)
	assert.Contains(t, str.tempSetMap[cv], field)
	assert.Equal(t, str.tempSetMap[cv][field], value)
}

func TestSetMulti(t *testing.T) {
	// Test should only set in internal map and not in redis
	var (
		str = New(secretKey, blockKey)

		// this key is unique across all tests
		field1 = "somekey1"
		value1 = 100
		field2 = "somekey2"
		value2 = 10
	)
	m := map[string]interface{}{
		field1: value1,
		field2: value2,
	}
	cv, err := str.encode(m)
	assert.Nil(t, err)

	err = str.SetMulti(cv, m)
	assert.NoError(t, err)
	assert.Contains(t, str.tempSetMap, cv)
	assert.Contains(t, str.tempSetMap[cv], field1)
	assert.Equal(t, str.tempSetMap[cv][field1], value1)
	assert.Contains(t, str.tempSetMap[cv], field2)
	assert.Equal(t, str.tempSetMap[cv][field2], value2)
}

func TestDelete(t *testing.T) {
	str := New(secretKey, blockKey)

	err := str.Delete("invalidkey", "somekey")
	assert.ErrorIs(t, err, ErrInvalidSession)

	m := map[string]interface{}{
		"key1": "val1",
		"key2": "val2",
	}
	cv, err := str.encode(m)
	assert.Nil(t, err)
	assert.NoError(t, str.Delete(cv, "key1"))
	assert.NotContains(t, str.tempSetMap[cv], "key1")
	assert.Contains(t, str.tempSetMap[cv], "key2")
}

func TestClear(t *testing.T) {
	str := New(secretKey, blockKey)
	err := str.Clear("xxx")
	assert.Nil(t, err)
	assert.Equal(t, len(str.tempSetMap["xxx"]), 0)
}

func TestFlush(t *testing.T) {
	str := New(secretKey, blockKey)
	m := map[string]interface{}{
		"key1": "val1",
		"key2": "val2",
	}

	str.tempSetMap["id"] = m
	cv, err := str.Flush("id")
	assert.Nil(t, err)

	vals, err := str.decode(cv)
	assert.Nil(t, err)
	assert.NotContains(t, str.tempSetMap, cv)
	assert.Contains(t, vals, "key1")
	assert.Contains(t, vals, "key2")
	assert.Equal(t, vals["key1"], "val1")
	assert.Equal(t, vals["key2"], "val2")

	_, err = str.Flush("xxx")
	assert.Equal(t, err.Error(), "nothing to flush")
}

func TestInt(t *testing.T) {
	str := New(secretKey, blockKey)

	var want int = 10
	v, err := str.Int(want, nil)
	assert.Nil(t, err)
	assert.Equal(t, v, want)

	testError := errors.New("test error")
	v, err = str.Int(want, testError)
	assert.Equal(t, v, 0)
	assert.ErrorIs(t, err, testError)

	_, err = str.Int("string", nil)
	assert.ErrorIs(t, err, ErrAssertType)
}

func TestInt64(t *testing.T) {
	str := New(secretKey, blockKey)

	var want int64 = 10
	v, err := str.Int64(want, nil)
	assert.Nil(t, err)
	assert.Equal(t, v, want)

	testError := errors.New("test error")
	_, err = str.Int64(want, testError)
	assert.ErrorIs(t, err, testError)

	_, err = str.Int64("string", nil)
	assert.ErrorIs(t, err, ErrAssertType)
}

func TestUInt64(t *testing.T) {
	str := New(secretKey, blockKey)

	var want uint64 = 10
	v, err := str.UInt64(want, nil)
	assert.Nil(t, err)
	assert.Equal(t, v, want)

	testError := errors.New("test error")
	_, err = str.UInt64(want, testError)
	assert.ErrorIs(t, err, testError)

	_, err = str.UInt64("string", nil)
	assert.ErrorIs(t, err, ErrAssertType)
}

func TestFloat64(t *testing.T) {
	str := New(secretKey, blockKey)

	var want float64 = 10
	v, err := str.Float64(want, nil)
	assert.Nil(t, err)
	assert.Equal(t, v, want)

	testError := errors.New("test error")
	_, err = str.Float64(want, testError)
	assert.ErrorIs(t, err, testError)

	_, err = str.Float64("string", nil)
	assert.ErrorIs(t, err, ErrAssertType)
}

func TestString(t *testing.T) {
	str := New(secretKey, blockKey)

	var want = "string"
	v, err := str.String(want, nil)
	assert.Nil(t, err)
	assert.Equal(t, v, want)

	testError := errors.New("test error")
	_, err = str.String(want, testError)
	assert.ErrorIs(t, err, testError)

	_, err = str.String(123, nil)
	assert.ErrorIs(t, err, ErrAssertType)
}

func TestBytes(t *testing.T) {
	str := New(secretKey, blockKey)

	var want = []byte("a")
	v, err := str.Bytes(want, nil)
	assert.Nil(t, err)
	assert.Equal(t, v, want)

	testError := errors.New("test error")
	_, err = str.Bytes(want, testError)
	assert.ErrorIs(t, err, testError)

	_, err = str.Bytes("string", nil)
	assert.ErrorIs(t, err, ErrAssertType)
}

func TestBool(t *testing.T) {
	str := New(secretKey, blockKey)

	var want = true
	v, err := str.Bool(want, nil)
	assert.Nil(t, err)
	assert.Equal(t, v, want)

	testError := errors.New("test error")
	_, err = str.Bool(want, testError)
	assert.ErrorIs(t, err, testError)

	_, err = str.Bool("string", nil)
	assert.ErrorIs(t, err, ErrAssertType)
}

func TestError(t *testing.T) {
	err := Err{
		code: 1,
		msg:  "test",
	}
	assert.Equal(t, 1, err.Code())
	assert.Equal(t, "test", err.Error())
}
