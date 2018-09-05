package securecookie

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vividvilla/simplesessions"
)

var (
	secretKey = []byte("0dIHy6S2uBuKaNnTUszB218L898ikGYA")
	blockKey  = []byte("0dIHy6S2uBuKaNnTUszB218L898ikGYA")
)

func TestNew(t *testing.T) {
	assert := assert.New(t)
	str := New(secretKey, blockKey)

	assert.NotNil(str.sc)
	assert.NotNil(str.tempSetMap)
}

func TestIsValid(t *testing.T) {
	assert := assert.New(t)
	str := New(secretKey, blockKey)
	sess := &simplesessions.Session{}

	assert.False(str.IsValid(sess, ""))

	encoded, err := str.encode(make(map[string]interface{}))
	assert.Nil(err)
	assert.True(str.IsValid(sess, encoded))
}

func TestCreate(t *testing.T) {
	assert := assert.New(t)
	str := New(secretKey, blockKey)
	sess := &simplesessions.Session{}

	id, err := str.Create(sess)
	assert.Nil(err)
	assert.True(str.IsValid(sess, id))
}

func TestGetInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New(secretKey, blockKey)
	sess := &simplesessions.Session{}

	val, err := str.Get(sess, "invalidkey", "invalidkey")
	assert.Nil(val)
	assert.Error(err, simplesessions.ErrInvalidSession.Error())
}

func TestGet(t *testing.T) {
	assert := assert.New(t)
	field := "somekey"
	value := 100

	// Set a key
	str := New(secretKey, blockKey)
	sess := &simplesessions.Session{}

	m := make(map[string]interface{})
	m[field] = value
	cv, err := str.encode(m)
	assert.Nil(err)

	val, err := str.Get(sess, cv, field)
	assert.NoError(err)
	assert.Equal(val, value)
}

func TestGetFieldNotFoundError(t *testing.T) {
	assert := assert.New(t)
	field := "someotherkey"

	// Set a key
	str := New(secretKey, blockKey)
	sess := &simplesessions.Session{}

	m := make(map[string]interface{})
	cv, err := str.encode(m)
	assert.Nil(err)

	_, err = str.Get(sess, cv, field)
	assert.Error(simplesessions.ErrFieldNotFound)
}

func TestGetMultiInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New(secretKey, blockKey)
	sess := &simplesessions.Session{}

	val, err := str.GetMulti(sess, "invalidkey", "invalidkey")
	assert.Nil(val)
	assert.Error(err, simplesessions.ErrInvalidSession.Error())
}

func TestGetMultiFieldEmptySession(t *testing.T) {
	assert := assert.New(t)
	str := New(secretKey, blockKey)
	sess := &simplesessions.Session{}

	m := make(map[string]interface{})
	cv, err := str.encode(m)
	assert.Nil(err)

	_, err = str.GetMulti(sess, cv)
	assert.Nil(err)
}

func TestGetMulti(t *testing.T) {
	assert := assert.New(t)
	field1 := "somekey"
	value1 := 100
	field2 := "someotherkey"
	value2 := "abc123"
	field3 := "thishouldntbethere"
	value3 := 100.10

	str := New(secretKey, blockKey)
	sess := &simplesessions.Session{}

	// Set a key
	m := make(map[string]interface{})
	m[field1] = value1
	m[field2] = value2
	m[field3] = value3
	cv, err := str.encode(m)
	assert.Nil(err)

	vals, err := str.GetMulti(sess, cv, field1, field2)
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
	str := New(secretKey, blockKey)
	sess := &simplesessions.Session{}

	val, err := str.GetAll(sess, "invalidkey")
	assert.Nil(val)
	assert.Error(err, simplesessions.ErrInvalidSession.Error())
}

func TestGetAll(t *testing.T) {
	assert := assert.New(t)
	field1 := "somekey"
	value1 := 100
	field2 := "someotherkey"
	value2 := "abc123"

	str := New(secretKey, blockKey)
	sess := &simplesessions.Session{}

	// Set a key
	m := make(map[string]interface{})
	m[field1] = value1
	m[field2] = value2
	cv, err := str.encode(m)
	assert.Nil(err)

	vals, err := str.GetAll(sess, cv)
	assert.NoError(err)
	assert.Contains(vals, field1)
	assert.Contains(vals, field2)

	assert.NoError(err)
	assert.Equal(vals[field1], value1)

	assert.NoError(err)
	assert.Equal(vals[field2], value2)
}

func TestSet(t *testing.T) {
	// Test should only set in internal map and not in redis
	assert := assert.New(t)
	str := New(secretKey, blockKey)
	sess := &simplesessions.Session{}

	// this key is unique across all tests
	field := "somekey"
	value := 100

	m := make(map[string]interface{})
	cv, err := str.encode(m)
	assert.Nil(err)

	err = str.Set(sess, cv, field, value)
	assert.NoError(err)
	assert.Contains(str.tempSetMap, cv)
	assert.Contains(str.tempSetMap[cv], field)
	assert.Equal(str.tempSetMap[cv][field], value)
}

func TestCommitInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New(secretKey, blockKey)
	sess := &simplesessions.Session{}

	err := str.Commit(sess, "invalid")
	assert.Error(err, simplesessions.ErrInvalidSession.Error())
}

func TestEmptyCommit(t *testing.T) {
	assert := assert.New(t)
	str := New(secretKey, blockKey)
	sess := &simplesessions.Session{}

	m := make(map[string]interface{})
	cv, err := str.encode(m)
	assert.Nil(err)

	err = str.Commit(sess, cv)
	assert.NoError(err)
}

func TestCommit(t *testing.T) {
	assert := assert.New(t)
	str := New(secretKey, blockKey)
	sessMan := simplesessions.New(simplesessions.Options{})
	sessMan.UseStore(str)

	var receivedCookieValue string
	sessMan.RegisterSetCookie(func(cookie *http.Cookie, w interface{}) error {
		receivedCookieValue = cookie.Value
		return nil
	})

	sessMan.RegisterGetCookie(func(name string, r interface{}) (*http.Cookie, error) {
		return nil, http.ErrNoCookie
	})

	sess, err := simplesessions.NewSession(sessMan, nil, nil)
	assert.Nil(err)

	// this key is unique across all tests
	field := "somekey"
	value := 100

	m := make(map[string]interface{})
	cv, err := str.encode(m)
	assert.Nil(err)

	err = str.Set(sess, cv, field, value)
	assert.NoError(err)
	assert.Equal(len(str.tempSetMap), 1)

	err = str.Commit(sess, cv)
	assert.NoError(err)
	assert.Equal(len(str.tempSetMap), 0)

	decoded, err := str.decode(receivedCookieValue)
	assert.NoError(err)
	assert.Contains(decoded, field)
	assert.Equal(decoded[field], value)
}

func TestDeleteInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New(secretKey, blockKey)
	sess := &simplesessions.Session{}

	err := str.Delete(sess, "invalidkey", "somekey")
	assert.Error(err, simplesessions.ErrInvalidSession.Error())
}

func TestDelete(t *testing.T) {
	assert := assert.New(t)
	str := New(secretKey, blockKey)
	sessMan := simplesessions.New(simplesessions.Options{})
	sessMan.UseStore(str)

	var receivedCookieValue string
	sessMan.RegisterSetCookie(func(cookie *http.Cookie, w interface{}) error {
		receivedCookieValue = cookie.Value
		return nil
	})

	sessMan.RegisterGetCookie(func(name string, r interface{}) (*http.Cookie, error) {
		return nil, http.ErrNoCookie
	})

	sess, err := simplesessions.NewSession(sessMan, nil, nil)
	assert.Nil(err)

	// this key is unique across all tests
	field := "somekey"
	value := 100

	m := make(map[string]interface{})
	m[field] = value
	cv, err := str.encode(m)
	assert.Nil(err)

	err = str.Delete(sess, cv, field)
	assert.NoError(err)
	assert.Equal(len(str.tempSetMap), 0)

	decoded, err := str.decode(receivedCookieValue)
	assert.NoError(err)
	assert.NotContains(decoded, field)
}

func TestClear(t *testing.T) {
	assert := assert.New(t)
	str := New(secretKey, blockKey)
	sessMan := simplesessions.New(simplesessions.Options{})
	sessMan.UseStore(str)

	var receivedCookieValue string
	sessMan.RegisterSetCookie(func(cookie *http.Cookie, w interface{}) error {
		receivedCookieValue = cookie.Value
		return nil
	})

	sessMan.RegisterGetCookie(func(name string, r interface{}) (*http.Cookie, error) {
		return nil, http.ErrNoCookie
	})

	sess, err := simplesessions.NewSession(sessMan, nil, nil)
	assert.Nil(err)

	// this key is unique across all tests
	field := "somekey"
	value := 100

	m := make(map[string]interface{})
	m[field] = value
	cv, err := str.encode(m)
	assert.Nil(err)

	err = str.Clear(sess, cv)
	assert.NoError(err)
	assert.Equal(len(str.tempSetMap), 0)

	decoded, err := str.decode(receivedCookieValue)
	assert.NoError(err)
	assert.NotContains(decoded, field)
}

func TestInt(t *testing.T) {
	assert := assert.New(t)
	str := New(secretKey, blockKey)

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
	str := New(secretKey, blockKey)

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
	str := New(secretKey, blockKey)

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
	str := New(secretKey, blockKey)

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
	str := New(secretKey, blockKey)

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
	str := New(secretKey, blockKey)

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
	str := New(secretKey, blockKey)

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
