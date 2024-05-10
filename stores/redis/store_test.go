package redis

import (
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
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

func TestCreate(t *testing.T) {
	assert := assert.New(t)
	str := New(getRedisPool())

	id, err := str.Create()
	assert.Nil(err)
	assert.Equal(len(id), sessionIDLen)
}

func TestGetInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New(getRedisPool())

	val, err := str.Get("invalidkey", "invalidkey")
	assert.Nil(val)
	assert.Error(err, ErrInvalidSession.Error())
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

	val, err := redis.Int(str.Get(key, field))
	assert.NoError(err)
	assert.Equal(val, value)
}

func TestGetFieldNotFoundError(t *testing.T) {
	assert := assert.New(t)
	str := New(getRedisPool())

	key := "10IHy6S2uBuKaNnTUszB218L898ikGY1"
	val, err := str.Get(key, "invalidkey")
	assert.Nil(val)
	assert.Error(err, ErrFieldNotFound.Error())
}

func TestGetMultiInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New(getRedisPool())

	val, err := str.GetMulti("invalidkey", "invalidkey")
	assert.Nil(val)
	assert.Error(err, ErrInvalidSession.Error())
}

func TestGetMultiFieldEmptySession(t *testing.T) {
	assert := assert.New(t)
	str := New(getRedisPool())

	key := "11IHy6S2uBuKaNnTUszB218L898ikGY1"
	field := "somefield"
	_, err := str.GetMulti(key, field)
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

	vals, err := str.GetMulti(key, field1, field2)
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

	val, err := str.GetAll("invalidkey")
	assert.Nil(val)
	assert.Error(err, ErrInvalidSession.Error())
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

	vals, err := str.GetAll(key)
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

	err := str.Set("invalidid", "key", "value")
	assert.Error(err, ErrInvalidSession.Error())
}

func TestSet(t *testing.T) {
	// Test should only set in internal map and not in redis
	assert := assert.New(t)
	redisPool := getRedisPool()
	str := New(redisPool)

	// this key is unique across all tests
	key := "7dIHy6S2uBuKaNnTUszB218L898ikGY9"
	field := "somekey"
	value := 100

	assert.NotNil(str.tempSetMap)
	assert.NotContains(str.tempSetMap, key)

	err := str.Set(key, field, value)
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

	err := str.Commit("invalidkey")
	assert.Error(err, ErrInvalidSession.Error())
}

func TestEmptyCommit(t *testing.T) {
	assert := assert.New(t)
	str := New(getRedisPool())

	err := str.Commit("15IHy6S2uBuKaNnTUszB2180898ikGY1")
	assert.NoError(err)
}

func TestCommit(t *testing.T) {
	// Test should commit in redis with expiry on key
	assert := assert.New(t)
	redisPool := getRedisPool()
	str := New(redisPool)

	str.SetTTL(10 * time.Second)

	// this key is unique across all tests
	key := "8dIHy6S2uBuKaNnTUszB2180898ikGY1"
	field1 := "somekey"
	value1 := 100
	field2 := "someotherkey"
	value2 := "abc123"

	err := str.Set(key, field1, value1)
	assert.NoError(err)

	err = str.Set(key, field2, value2)
	assert.NoError(err)

	err = str.Commit(key)
	assert.NoError(err)

	conn := redisPool.Get()
	defer conn.Close()
	vals, err := redis.Values(conn.Do("HGETALL", defaultPrefix+key))
	assert.Equal(2*2, len(vals))

	ttl, err := redis.Int(conn.Do("TTL", defaultPrefix+key))
	assert.NoError(err)

	assert.Equal(true, ttl > 0 && ttl <= 10)
}

func TestDeleteInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New(getRedisPool())

	err := str.Delete("invalidkey", "somefield")
	assert.Error(err, ErrInvalidSession.Error())
}

func TestDelete(t *testing.T) {
	// Test should only set in internal map and not in redis
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

	err = str.Delete(key, field1)
	assert.NoError(err)

	val, err := redis.Bool(conn.Do("HEXISTS", defaultPrefix+key, field1))
	assert.False(val)

	val, err = redis.Bool(conn.Do("HEXISTS", defaultPrefix+key, field2))
	assert.True(val)
}

func TestClearInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New(getRedisPool())

	err := str.Clear("invalidkey")
	assert.Error(err, ErrInvalidSession.Error())
}

func TestClear(t *testing.T) {
	// Test should only set in internal map and not in redis
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

	// Check if its set
	val, err := conn.Do("TTL", defaultPrefix+key)
	assert.NoError(err)
	// -2 represents key doesn't exist
	assert.NotEqual(val, int64(-2))

	err = str.Clear(key)
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
