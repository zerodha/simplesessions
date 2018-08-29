package memorystore

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zerodhatech/simplesessions"
)

func TestNew(t *testing.T) {
	assert := assert.New(t)
	str := New()
	assert.NotNil(str.sessions)
}

func TestIsValidSessionID(t *testing.T) {
	assert := assert.New(t)
	str := New()
	sess := &simplesessions.Session{}

	// Not valid since length doesn't match
	testString := "abc123"
	assert.NotEqual(len(testString), sessionIDLen)
	assert.False(str.isValidSessionID(sess, testString))

	// Not valid since length is same but not alpha numeric
	invalidTestString := "0dIHy6S2uBuKaNnTUszB218L898ikGY$"
	assert.Equal(len(invalidTestString), sessionIDLen)
	assert.False(str.isValidSessionID(sess, invalidTestString))

	// Valid
	validTestString := "1dIHy6S2uBuKaNnTUszB218L898ikGY1"
	assert.Equal(len(validTestString), sessionIDLen)
	assert.True(str.isValidSessionID(sess, validTestString))
}

func TestIsValid(t *testing.T) {
	assert := assert.New(t)
	str := New()
	sess := &simplesessions.Session{}

	// Not valid since length doesn't match
	testString := "abc123"
	assert.NotEqual(len(testString), sessionIDLen)
	assert.False(str.IsValid(sess, testString))

	// Not valid since length is same but not alpha numeric
	invalidTestString := "2dIHy6S2uBuKaNnTUszB218L898ikGY$"
	assert.Equal(len(invalidTestString), sessionIDLen)
	assert.False(str.IsValid(sess, invalidTestString))

	// Valid
	validTestString := "3dIHy6S2uBuKaNnTUszB218L898ikGY1"
	assert.Equal(len(validTestString), sessionIDLen)
	assert.True(str.IsValid(sess, validTestString))
}

func TestCreate(t *testing.T) {
	assert := assert.New(t)
	str := New()
	sess := &simplesessions.Session{}

	id, err := str.Create(sess)
	assert.Nil(err)
	assert.Equal(len(id), sessionIDLen)
	assert.True(str.IsValid(sess, id))
}

func TestGetInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New()
	sess := &simplesessions.Session{}

	val, err := str.Get(sess, "invalidkey", "invalidkey")
	assert.Nil(val)
	assert.Error(err, simplesessions.ErrInvalidSession.Error())
}

func TestGet(t *testing.T) {
	assert := assert.New(t)
	key := "4dIHy6S2uBuKaNnTUszB218L898ikGY1"
	field := "somekey"
	value := 100

	// Set a key
	str := New()
	sess := &simplesessions.Session{}
	str.sessions[key] = make(map[string]interface{})
	str.sessions[key][field] = value

	val, err := str.Get(sess, key, field)
	assert.NoError(err)
	assert.Equal(val, value)
}

func TestGetFieldNotFoundError(t *testing.T) {
	assert := assert.New(t)
	str := New()
	sess := &simplesessions.Session{}

	key := "10IHy6S2uBuKaNnTUszB218L898ikGY1"
	val, err := str.Get(sess, key, "invalidkey")
	assert.Nil(val)
	assert.Error(err, simplesessions.ErrFieldNotFound.Error())
}

func TestGetMultiInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New()
	sess := &simplesessions.Session{}

	val, err := str.GetMulti(sess, "invalidkey", "invalidkey")
	assert.Nil(val)
	assert.Error(err, simplesessions.ErrInvalidSession.Error())
}

func TestGetMultiFieldEmptySession(t *testing.T) {
	assert := assert.New(t)
	str := New()
	sess := &simplesessions.Session{}

	key := "11IHy6S2uBuKaNnTUszB218L898ikGY1"
	_, err := str.GetMulti(sess, key)
	assert.Nil(err)
}

func TestGetMulti(t *testing.T) {
	assert := assert.New(t)
	key := "5dIHy6S2uBuKaNnTUszB218L898ikGY1"
	field1 := "somekey"
	value1 := 100
	field2 := "someotherkey"
	value2 := "abc123"
	field3 := "thishouldntbethere"
	value3 := 100.10

	str := New()
	sess := &simplesessions.Session{}

	// Set a key
	str.sessions[key] = make(map[string]interface{})
	str.sessions[key][field1] = value1
	str.sessions[key][field2] = value2
	str.sessions[key][field3] = value3

	vals, err := str.GetMulti(sess, key, field1, field2)
	assert.NoError(err)
	assert.Contains(vals, field1)
	assert.Contains(vals, field2)
	assert.NotContains(vals, field3)

	assert.NoError(err)
	assert.Equal(vals[field1], value1)

	assert.NoError(err)
	assert.Equal(vals[field2], value2)
}

func TestGetAllInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New()
	sess := &simplesessions.Session{}

	val, err := str.GetAll(sess, "invalidkey")
	assert.Nil(val)
	assert.Error(err, simplesessions.ErrInvalidSession.Error())
}

func TestGetAll(t *testing.T) {
	assert := assert.New(t)
	key := "6dIHy6S2uBuKaNnTUszB218L898ikGY1"
	field1 := "somekey"
	value1 := 100
	field2 := "someotherkey"
	value2 := "abc123"
	field3 := "thishouldntbethere"
	value3 := 100.10

	str := New()
	sess := &simplesessions.Session{}

	// Set a key
	str.sessions[key] = make(map[string]interface{})
	str.sessions[key][field1] = value1
	str.sessions[key][field2] = value2
	str.sessions[key][field3] = value3

	vals, err := str.GetAll(sess, key)
	assert.NoError(err)
	assert.Contains(vals, field1)
	assert.Contains(vals, field2)
	assert.Contains(vals, field3)

	assert.NoError(err)
	assert.Equal(vals[field1], value1)

	assert.NoError(err)
	assert.Equal(vals[field2], value2)

	assert.NoError(err)
	assert.Equal(vals[field3], value3)
}

func TestSetInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New()
	sess := &simplesessions.Session{}

	err := str.Set(sess, "invalidid", "key", "value")
	assert.Error(err, simplesessions.ErrInvalidSession.Error())
}

func TestSet(t *testing.T) {
	// Test should only set in internal map and not in redis
	assert := assert.New(t)
	str := New()
	sess := &simplesessions.Session{}

	// this key is unique across all tests
	key := "7dIHy6S2uBuKaNnTUszB218L898ikGY9"
	field := "somekey"
	value := 100

	assert.NotContains(str.sessions, key)

	err := str.Set(sess, key, field, value)
	assert.NoError(err)
	assert.Contains(str.sessions, key)
	assert.Contains(str.sessions[key], field)
	assert.Equal(str.sessions[key][field], value)
}

func TestCommit(t *testing.T) {
	assert := assert.New(t)
	str := New()
	sess := &simplesessions.Session{}

	err := str.Commit(sess, "invalidkey")
	assert.Nil(err)
}

func TestDeleteInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New()
	sess := &simplesessions.Session{}

	err := str.Delete(sess, "invalidkey", "somekey")
	assert.Error(err, simplesessions.ErrInvalidSession.Error())
}

func TestDelete(t *testing.T) {
	// Test should only set in internal map and not in redis
	assert := assert.New(t)
	str := New()
	sess := &simplesessions.Session{}

	// this key is unique across all tests
	key := "8dIHy6S2uBuKaNnTUszB2180898ikGY1"
	field1 := "somefield1"
	field2 := "somefield2"
	str.sessions[key] = make(map[string]interface{})
	str.sessions[key][field1] = 10
	str.sessions[key][field2] = 10

	err := str.Delete(sess, key, field1)
	assert.NoError(err)
	assert.Contains(str.sessions[key], field2)
	assert.NotContains(str.sessions[key], field1)
}

func TestClearInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New()
	sess := &simplesessions.Session{}

	err := str.Clear(sess, "invalidkey")
	assert.Error(err, simplesessions.ErrInvalidSession.Error())
}

func TestClear(t *testing.T) {
	// Test should only set in internal map and not in redis
	assert := assert.New(t)
	str := New()
	sess := &simplesessions.Session{}

	// this key is unique across all tests
	key := "8dIHy6S2uBuKaNnTUszB2180898ikGY1"
	str.sessions[key] = make(map[string]interface{})

	err := str.Clear(sess, key)
	assert.NoError(err)
	assert.NotContains(str.sessions, key)
}

func TestInt(t *testing.T) {
	assert := assert.New(t)
	str := New()

	var want int = 10
	v, err := str.Int(want, nil)
	assert.Nil(err)
	assert.Equal(v, want)

	testError := errors.New("test error")
	v, err = str.Int(want, testError)
	assert.Equal(v, 0)
	assert.Error(testError)

	_, err = str.Int("string", nil)
	assert.Error(simplesessions.ErrAssertType)
}

func TestInt64(t *testing.T) {
	assert := assert.New(t)
	str := New()

	var want int64 = 10
	v, err := str.Int64(want, nil)
	assert.Nil(err)
	assert.Equal(v, want)

	testError := errors.New("test error")
	v, err = str.Int64(want, testError)
	assert.Error(testError)

	_, err = str.Int64("string", nil)
	assert.Error(simplesessions.ErrAssertType)
}

func TestUInt64(t *testing.T) {
	assert := assert.New(t)
	str := New()

	var want uint64 = 10
	v, err := str.UInt64(want, nil)
	assert.Nil(err)
	assert.Equal(v, want)

	testError := errors.New("test error")
	v, err = str.UInt64(want, testError)
	assert.Error(testError)

	_, err = str.UInt64("string", nil)
	assert.Error(simplesessions.ErrAssertType)
}

func TestFloat64(t *testing.T) {
	assert := assert.New(t)
	str := New()

	var want float64 = 10
	v, err := str.Float64(want, nil)
	assert.Nil(err)
	assert.Equal(v, want)

	testError := errors.New("test error")
	v, err = str.Float64(want, testError)
	assert.Error(testError)

	_, err = str.Float64("string", nil)
	assert.Error(simplesessions.ErrAssertType)
}

func TestString(t *testing.T) {
	assert := assert.New(t)
	str := New()

	var want = "string"
	v, err := str.String(want, nil)
	assert.Nil(err)
	assert.Equal(v, want)

	testError := errors.New("test error")
	v, err = str.String(want, testError)
	assert.Error(testError)

	_, err = str.String(123, nil)
	assert.Error(simplesessions.ErrAssertType)
}

func TestBytes(t *testing.T) {
	assert := assert.New(t)
	str := New()

	var want = []byte("a")
	v, err := str.Bytes(want, nil)
	assert.Nil(err)
	assert.Equal(v, want)

	testError := errors.New("test error")
	v, err = str.Bytes(want, testError)
	assert.Error(testError)

	_, err = str.Bytes("string", nil)
	assert.Error(simplesessions.ErrAssertType)
}

func TestBool(t *testing.T) {
	assert := assert.New(t)
	str := New()

	var want = true
	v, err := str.Bool(want, nil)
	assert.Nil(err)
	assert.Equal(v, want)

	testError := errors.New("test error")
	v, err = str.Bool(want, testError)
	assert.Error(testError)

	_, err = str.Bool("string", nil)
	assert.Error(simplesessions.ErrAssertType)
}
