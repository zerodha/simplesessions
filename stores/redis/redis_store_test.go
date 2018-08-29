package redisstore

import (
	"errors"
	"testing"
	"time"

	"github.com/zerodhatech/simplesessions"

	"github.com/alicebob/miniredis"
	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
)

var (
	mockRedis *miniredis.Miniredis
)

func init() {
	var err error
	mockRedis, err = miniredis.Run()
	if err != nil {
		panic(err)
	}
}

func getRedisPool() *redis.Pool {
	return &redis.Pool{
		Wait: true,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial(
				"tcp",
				mockRedis.Addr(),
			)

			return c, err
		},
	}
}

func TestNew(t *testing.T) {
	assert := assert.New(t)
	rPool := getRedisPool()
	str := New(rPool)
	assert.Equal(str.prefix, defaultPrefix)
	assert.Equal(str.pool, rPool)
	assert.NotNil(str.tempSetMap)
}

func TestSetPrefix(t *testing.T) {
	assert := assert.New(t)
	str := New(getRedisPool())
	str.SetPrefix("test")
	assert.Equal(str.prefix, "test")
}

func TestSetTTL(t *testing.T) {
	assert := assert.New(t)
	testDur := time.Second * 10
	str := New(getRedisPool())
	str.SetTTL(testDur)
	assert.Equal(str.ttl, testDur)
}

func TestIsValidSessionID(t *testing.T) {
	assert := assert.New(t)
	str := New(getRedisPool())
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
	str := New(getRedisPool())
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
	str := New(getRedisPool())
	sess := &simplesessions.Session{}

	id, err := str.Create(sess)
	assert.Nil(err)
	assert.Equal(len(id), sessionIDLen)
	assert.True(str.IsValid(sess, id))
}

func TestGetInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New(getRedisPool())
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
	redisPool := getRedisPool()

	// Set a key
	conn := redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("HSET", defaultPrefix+key, field, value)
	assert.NoError(err)

	str := New(redisPool)
	sess := &simplesessions.Session{}

	val, err := redis.Int(str.Get(sess, key, field))
	assert.NoError(err)
	assert.Equal(val, value)
}

func TestGetFieldNotFoundError(t *testing.T) {
	assert := assert.New(t)
	str := New(getRedisPool())
	sess := &simplesessions.Session{}

	key := "10IHy6S2uBuKaNnTUszB218L898ikGY1"
	val, err := str.Get(sess, key, "invalidkey")
	assert.Nil(val)
	assert.Error(err, simplesessions.ErrFieldNotFound.Error())
}

func TestGetMultiInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New(getRedisPool())
	sess := &simplesessions.Session{}

	val, err := str.GetMulti(sess, "invalidkey", "invalidkey")
	assert.Nil(val)
	assert.Error(err, simplesessions.ErrInvalidSession.Error())
}

func TestGetMultiFieldEmptySession(t *testing.T) {
	assert := assert.New(t)
	str := New(getRedisPool())
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
	redisPool := getRedisPool()

	// Set a key
	conn := redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("HMSET", defaultPrefix+key, field1, value1, field2, value2, field3, value3)
	assert.NoError(err)

	str := New(redisPool)
	sess := &simplesessions.Session{}

	vals, err := str.GetMulti(sess, key, field1, field2)
	assert.NoError(err)
	assert.Contains(vals, field1)
	assert.Contains(vals, field2)
	assert.NotContains(vals, field3)

	val1, err := redis.Int(vals[field1], nil)
	assert.NoError(err)
	assert.Equal(val1, value1)

	val2, err := redis.String(vals[field2], nil)
	assert.NoError(err)
	assert.Equal(val2, value2)
}

func TestGetAllInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New(getRedisPool())
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
	redisPool := getRedisPool()

	// Set a key
	conn := redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("HMSET", defaultPrefix+key, field1, value1, field2, value2, field3, value3)
	assert.NoError(err)

	str := New(redisPool)
	sess := &simplesessions.Session{}

	vals, err := str.GetAll(sess, key)
	assert.NoError(err)
	assert.Contains(vals, field1)
	assert.Contains(vals, field2)
	assert.Contains(vals, field3)

	val1, err := redis.Int(vals[field1], nil)
	assert.NoError(err)
	assert.Equal(val1, value1)

	val2, err := redis.String(vals[field2], nil)
	assert.NoError(err)
	assert.Equal(val2, value2)

	val3, err := redis.Float64(vals[field3], nil)
	assert.NoError(err)
	assert.Equal(val3, value3)
}

func TestSetInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New(getRedisPool())
	sess := &simplesessions.Session{}

	err := str.Set(sess, "invalidid", "key", "value")
	assert.Error(err, simplesessions.ErrInvalidSession.Error())
}

func TestSet(t *testing.T) {
	// Test should only set in internal map and not in redis
	assert := assert.New(t)
	redisPool := getRedisPool()
	str := New(redisPool)
	sess := &simplesessions.Session{}

	// this key is unique across all tests
	key := "7dIHy6S2uBuKaNnTUszB218L898ikGY9"
	field := "somekey"
	value := 100

	assert.NotNil(str.tempSetMap)
	assert.NotContains(str.tempSetMap, key)

	err := str.Set(sess, key, field, value)
	assert.NoError(err)
	assert.Contains(str.tempSetMap, key)
	assert.Contains(str.tempSetMap[key], field)
	assert.Equal(str.tempSetMap[key][field], value)

	// Check ifs not commited to redis
	conn := redisPool.Get()
	defer conn.Close()
	val, err := conn.Do("TTL", defaultPrefix+key)
	assert.NoError(err)
	// -2 represents key doesn't exist
	assert.Equal(val, int64(-2))
}

func TestCommitInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New(getRedisPool())
	sess := &simplesessions.Session{}

	err := str.Commit(sess, "invalidkey")
	assert.Error(err, simplesessions.ErrInvalidSession.Error())
}

func TestEmptyCommit(t *testing.T) {
	assert := assert.New(t)
	str := New(getRedisPool())
	sess := &simplesessions.Session{}

	err := str.Commit(sess, "15IHy6S2uBuKaNnTUszB2180898ikGY1")
	assert.NoError(err)
}

func TestCommit(t *testing.T) {
	// Test should only set in internal map and not in redis
	assert := assert.New(t)
	redisPool := getRedisPool()
	str := New(redisPool)
	sess := &simplesessions.Session{}

	// this key is unique across all tests
	key := "8dIHy6S2uBuKaNnTUszB2180898ikGY1"
	field1 := "somekey"
	value1 := 100
	field2 := "someotherkey"
	value2 := "abc123"

	err := str.Set(sess, key, field1, value1)
	assert.NoError(err)

	err = str.Set(sess, key, field2, value2)
	assert.NoError(err)

	err = str.Commit(sess, key)
	assert.NoError(err)

	conn := redisPool.Get()
	defer conn.Close()
	vals, err := redis.Values(conn.Do("HGETALL", defaultPrefix+key))
	assert.Equal(2*2, len(vals))
}

func TestClearInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New(getRedisPool())
	sess := &simplesessions.Session{}

	err := str.Clear(sess, "invalidkey")
	assert.Error(err, simplesessions.ErrInvalidSession.Error())
}

func TestClear(t *testing.T) {
	// Test should only set in internal map and not in redis
	assert := assert.New(t)
	redisPool := getRedisPool()
	str := New(redisPool)
	sess := &simplesessions.Session{}

	// this key is unique across all tests
	key := "8dIHy6S2uBuKaNnTUszB2180898ikGY1"
	field1 := "somekey"
	value1 := 100
	field2 := "someotherkey"
	value2 := "abc123"

	conn := redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("HMSET", defaultPrefix+key, field1, value1, field2, value2)
	assert.NoError(err)

	// Check if its set
	val, err := conn.Do("TTL", defaultPrefix+key)
	assert.NoError(err)
	// -2 represents key doesn't exist
	assert.NotEqual(val, int64(-2))

	err = str.Clear(sess, key)
	assert.NoError(err)

	val, err = conn.Do("TTL", defaultPrefix+key)
	assert.NoError(err)
	// -2 represents key doesn't exist
	assert.Equal(val, int64(-2))
}

func TestInterfaceMap(t *testing.T) {
	assert := assert.New(t)
	redisPool := getRedisPool()
	str := New(redisPool)

	// this key is unique across all tests
	key := "8dIHy6S2uBuKaNnTUszB2180898ikGY1"
	field1 := "somekey"
	value1 := 100
	field2 := "someotherkey"
	value2 := "abc123"

	conn := redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("HMSET", defaultPrefix+key, field1, value1, field2, value2)
	assert.NoError(err)

	vals, err := str.interfaceMap(conn.Do("HGETALL", defaultPrefix+key))
	assert.Contains(vals, field1)
	assert.Contains(vals, field2)
}

func TestInterfaceMapWithError(t *testing.T) {
	assert := assert.New(t)
	redisPool := getRedisPool()
	str := New(redisPool)

	testError := errors.New("test error")
	vals, err := str.interfaceMap(nil, testError)
	assert.Nil(vals)
	assert.Error(err, testError.Error())

	valsInfSlice := []interface{}{nil, nil, nil}
	vals, err = str.interfaceMap(valsInfSlice, nil)
	assert.Nil(vals)
	assert.Equal(err.Error(), "redigo: StringMap expects even number of values result")

	valsInfSlice = []interface{}{"abc123", 123}
	vals, err = str.interfaceMap(valsInfSlice, nil)
	assert.Nil(vals)
	assert.Equal(err.Error(), "redigo: StringMap key not a bulk string value")
}

func TestInt(t *testing.T) {
	assert := assert.New(t)
	redisPool := getRedisPool()
	str := New(redisPool)

	field := "somekey"
	value := 100

	conn := redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("SET", field, value)
	assert.NoError(err)

	val, err := str.Int(conn.Do("GET", field))
	assert.NoError(err)
	assert.Equal(value, val)

	testError := errors.New("test error")
	val, err = str.Int(value, testError)
	assert.Error(err, testError.Error())
}

func TestInt64(t *testing.T) {
	assert := assert.New(t)
	redisPool := getRedisPool()
	str := New(redisPool)

	field := "somekey"
	var value int64 = 100

	conn := redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("SET", field, value)
	assert.NoError(err)

	val, err := str.Int64(conn.Do("GET", field))
	assert.NoError(err)
	assert.Equal(value, val)

	testError := errors.New("test error")
	val, err = str.Int64(value, testError)
	assert.Error(err, testError.Error())
}

func TestUInt64(t *testing.T) {
	assert := assert.New(t)
	redisPool := getRedisPool()
	str := New(redisPool)

	field := "somekey"
	var value uint64 = 100

	conn := redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("SET", field, value)
	assert.NoError(err)

	val, err := str.UInt64(conn.Do("GET", field))
	assert.NoError(err)
	assert.Equal(value, val)

	testError := errors.New("test error")
	val, err = str.UInt64(value, testError)
	assert.Error(err, testError.Error())
}

func TestFloat64(t *testing.T) {
	assert := assert.New(t)
	redisPool := getRedisPool()
	str := New(redisPool)

	field := "somekey"
	var value float64 = 100

	conn := redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("SET", field, value)
	assert.NoError(err)

	val, err := str.Float64(conn.Do("GET", field))
	assert.NoError(err)
	assert.Equal(value, val)

	testError := errors.New("test error")
	val, err = str.Float64(value, testError)
	assert.Error(err, testError.Error())
}

func TestString(t *testing.T) {
	assert := assert.New(t)
	redisPool := getRedisPool()
	str := New(redisPool)

	field := "somekey"
	value := "abc123"

	conn := redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("SET", field, value)
	assert.NoError(err)

	val, err := str.String(conn.Do("GET", field))
	assert.NoError(err)
	assert.Equal(value, val)

	testError := errors.New("test error")
	val, err = str.String(value, testError)
	assert.Error(err, testError.Error())
}

func TestBytes(t *testing.T) {
	assert := assert.New(t)
	redisPool := getRedisPool()
	str := New(redisPool)

	field := "somekey"
	var value []byte = []byte("abc123")

	conn := redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("SET", field, value)
	assert.NoError(err)

	val, err := str.Bytes(conn.Do("GET", field))
	assert.NoError(err)
	assert.Equal(value, val)

	testError := errors.New("test error")
	val, err = str.Bytes(value, testError)
	assert.Error(err, testError.Error())
}

func TestBool(t *testing.T) {
	assert := assert.New(t)
	redisPool := getRedisPool()
	str := New(redisPool)

	field := "somekey"
	value := true

	conn := redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("SET", field, value)
	assert.NoError(err)

	val, err := str.Bool(conn.Do("GET", field))
	assert.NoError(err)
	assert.Equal(value, val)

	testError := errors.New("test error")
	val, err = str.Bool(value, testError)
	assert.Error(err, testError.Error())
}
