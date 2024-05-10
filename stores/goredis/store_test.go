package goredis

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
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

func getRedisClient() redis.UniversalClient {
	return redis.NewClient(&redis.Options{
		Addr: mockRedis.Addr(),
	})
}

func TestNew(t *testing.T) {
	assert := assert.New(t)
	client := getRedisClient()
	ctx := context.Background()
	str := New(ctx, client)
	assert.Equal(str.prefix, defaultPrefix)
	assert.Equal(str.client, client)
	assert.Equal(str.clientCtx, ctx)
	assert.NotNil(str.tempSetMap)
}

func TestSetPrefix(t *testing.T) {
	assert := assert.New(t)
	str := New(context.TODO(), getRedisClient())
	str.SetPrefix("test")
	assert.Equal(str.prefix, "test")
}

func TestSetTTL(t *testing.T) {
	assert := assert.New(t)
	testDur := time.Second * 10
	str := New(context.TODO(), getRedisClient())
	str.SetTTL(testDur)
	assert.Equal(str.ttl, testDur)
}

func TestCreate(t *testing.T) {
	assert := assert.New(t)
	str := New(context.TODO(), getRedisClient())

	id, err := str.Create()
	assert.Nil(err)
	assert.Equal(len(id), sessionIDLen)
}

func TestGetInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New(context.TODO(), getRedisClient())

	val, err := str.Get("invalidkey", "invalidkey")
	assert.Nil(val)
	assert.Error(err, ErrInvalidSession)
}

func TestGet(t *testing.T) {
	assert := assert.New(t)
	key := "4dIHy6S2uBuKaNnTUszB218L898ikGY1"
	field := "somekey"
	value := 100
	client := getRedisClient()

	// Set a key
	err := client.HSet(context.TODO(), defaultPrefix+key, field, value).Err()
	assert.NoError(err)

	str := New(context.TODO(), client)

	val, err := str.Int(str.Get(key, field))
	assert.NoError(err)
	assert.Equal(val, value)
}

func TestGetFieldNotFoundError(t *testing.T) {
	assert := assert.New(t)
	str := New(context.TODO(), getRedisClient())

	key := "10IHy6S2uBuKaNnTUszB218L898ikGY1"
	val, err := str.Get(key, "invalidkey")
	assert.Nil(val)
	assert.Error(err, ErrFieldNotFound.Error())
}

func TestGetMultiInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New(context.TODO(), getRedisClient())

	val, err := str.GetMulti("invalidkey", "invalidkey")
	assert.Nil(val)
	assert.Error(err, ErrInvalidSession.Error())
}

func TestGetMultiFieldEmptySession(t *testing.T) {
	assert := assert.New(t)
	str := New(context.TODO(), getRedisClient())

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
	client := getRedisClient()

	// Set a key
	err := client.HMSet(context.TODO(), defaultPrefix+key, field1, value1, field2, value2, field3, value3).Err()
	assert.NoError(err)

	str := New(context.TODO(), client)

	vals, err := str.GetMulti(key, field1, field2)
	assert.NoError(err)
	assert.Contains(vals, field1)
	assert.Contains(vals, field2)
	assert.NotContains(vals, field3)

	val1, err := str.Int(vals[field1], nil)
	assert.NoError(err)
	assert.Equal(val1, value1)

	val2, err := str.String(vals[field2], nil)
	assert.NoError(err)
	assert.Equal(val2, value2)
}

func TestGetAllInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New(context.TODO(), getRedisClient())

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
	client := getRedisClient()

	// Set a key
	err := client.HMSet(context.TODO(), defaultPrefix+key, field1, value1, field2, value2, field3, value3).Err()
	assert.NoError(err)

	str := New(context.TODO(), client)

	vals, err := str.GetAll(key)
	assert.NoError(err)
	assert.Contains(vals, field1)
	assert.Contains(vals, field2)
	assert.Contains(vals, field3)

	val1, err := str.Int(vals[field1], nil)
	assert.NoError(err)
	assert.Equal(val1, value1)

	val2, err := str.String(vals[field2], nil)
	assert.NoError(err)
	assert.Equal(val2, value2)

	val3, err := str.Float64(vals[field3], nil)
	assert.NoError(err)
	assert.Equal(val3, value3)
}

func TestSetInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New(context.TODO(), getRedisClient())

	err := str.Set("invalidid", "key", "value")
	assert.Error(err, ErrInvalidSession.Error())
}

func TestSet(t *testing.T) {
	// Test should only set in internal map and not in redis
	assert := assert.New(t)
	client := getRedisClient()
	str := New(context.TODO(), client)

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
	val, err := client.Exists(context.TODO(), defaultPrefix+key).Result()
	assert.NoError(err)
	assert.Equal(val, int64(0))
}

func TestCommitInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New(context.TODO(), getRedisClient())

	err := str.Commit("invalidkey")
	assert.Error(err, ErrInvalidSession.Error())
}

func TestEmptyCommit(t *testing.T) {
	assert := assert.New(t)
	str := New(context.TODO(), getRedisClient())

	err := str.Commit("15IHy6S2uBuKaNnTUszB2180898ikGY1")
	assert.NoError(err)
}

func TestCommit(t *testing.T) {
	// Test should commit in redis with expiry on key
	assert := assert.New(t)
	client := getRedisClient()
	str := New(context.TODO(), client)

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

	vals, err := client.HGetAll(context.TODO(), defaultPrefix+key).Result()
	assert.Equal(2, len(vals))

	ttl, err := client.TTL(context.TODO(), defaultPrefix+key).Result()
	assert.NoError(err)
	assert.Equal(true, ttl.Seconds() > 0 && ttl.Seconds() <= 10)
}

func TestDeleteInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New(context.TODO(), getRedisClient())

	err := str.Delete("invalidkey", "somefield")
	assert.Error(err, ErrInvalidSession.Error())
}

func TestDelete(t *testing.T) {
	// Test should only set in internal map and not in redis
	assert := assert.New(t)
	client := getRedisClient()
	str := New(context.TODO(), client)

	// this key is unique across all tests
	key := "8dIHy6S2uBuKaNnTUszB2180898ikGY1"
	field1 := "somekey"
	value1 := 100
	field2 := "someotherkey"
	value2 := "abc123"

	err := client.HMSet(context.TODO(), defaultPrefix+key, field1, value1, field2, value2).Err()
	assert.NoError(err)

	err = str.Delete(key, field1)
	assert.NoError(err)

	val, err := client.HExists(context.TODO(), defaultPrefix+key, field1).Result()
	assert.False(val)

	val, err = client.HExists(context.TODO(), defaultPrefix+key, field2).Result()
	assert.True(val)
}

func TestClearInvalidSessionError(t *testing.T) {
	assert := assert.New(t)
	str := New(context.TODO(), getRedisClient())

	err := str.Clear("invalidkey")
	assert.Error(err, ErrInvalidSession.Error())
}

func TestClear(t *testing.T) {
	// Test should only set in internal map and not in redis
	assert := assert.New(t)
	client := getRedisClient()
	str := New(context.TODO(), client)

	// this key is unique across all tests
	key := "8dIHy6S2uBuKaNnTUszB2180898ikGY1"
	field1 := "somekey"
	value1 := 100
	field2 := "someotherkey"
	value2 := "abc123"

	err := client.HMSet(context.TODO(), defaultPrefix+key, field1, value1, field2, value2).Err()
	assert.NoError(err)

	// Check if its set
	val, err := client.Exists(context.TODO(), defaultPrefix+key).Result()
	assert.NoError(err)
	assert.NotEqual(val, int64(0))

	err = str.Clear(key)
	assert.NoError(err)

	val, err = client.Exists(context.TODO(), defaultPrefix+key).Result()
	assert.NoError(err)
	assert.Equal(val, int64(0))
}

func TestInt(t *testing.T) {
	assert := assert.New(t)
	client := getRedisClient()
	str := New(context.TODO(), client)

	field := "somekey"
	value := 100

	err := client.Set(context.TODO(), field, value, 0).Err()
	assert.NoError(err)

	val, err := str.Int(client.Get(context.TODO(), field).Result())
	assert.NoError(err)
	assert.Equal(value, val)

	testError := errors.New("test error")
	val, err = str.Int(value, testError)
	assert.Error(err, testError.Error())
}

func TestInt64(t *testing.T) {
	assert := assert.New(t)
	client := getRedisClient()
	str := New(context.TODO(), client)

	field := "somekey"
	var value int64 = 100

	err := client.Set(context.TODO(), field, value, 0).Err()
	assert.NoError(err)

	val, err := str.Int64(client.Get(context.TODO(), field).Result())
	assert.NoError(err)
	assert.Equal(value, val)

	testError := errors.New("test error")
	val, err = str.Int64(value, testError)
	assert.Error(err, testError.Error())
}

func TestUInt64(t *testing.T) {
	assert := assert.New(t)
	client := getRedisClient()
	str := New(context.TODO(), client)

	field := "somekey"
	var value uint64 = 100

	err := client.Set(context.TODO(), field, value, 0).Err()
	assert.NoError(err)

	val, err := str.UInt64(client.Get(context.TODO(), field).Result())
	assert.NoError(err)
	assert.Equal(value, val)

	testError := errors.New("test error")
	val, err = str.UInt64(value, testError)
	assert.Error(err, testError.Error())
}

func TestFloat64(t *testing.T) {
	assert := assert.New(t)
	client := getRedisClient()
	str := New(context.TODO(), client)

	field := "somekey"
	var value float64 = 100

	err := client.Set(context.TODO(), field, value, 0).Err()
	assert.NoError(err)

	val, err := str.Float64(client.Get(context.TODO(), field).Result())
	assert.NoError(err)
	assert.Equal(value, val)

	testError := errors.New("test error")
	val, err = str.Float64(value, testError)
	assert.Error(err, testError.Error())
}

func TestString(t *testing.T) {
	assert := assert.New(t)
	client := getRedisClient()
	str := New(context.TODO(), client)

	field := "somekey"
	value := "abc123"

	err := client.Set(context.TODO(), field, value, 0).Err()
	assert.NoError(err)

	val, err := str.String(client.Get(context.TODO(), field).Result())
	assert.NoError(err)
	assert.Equal(value, val)

	testError := errors.New("test error")
	val, err = str.String(value, testError)
	assert.Error(err, testError.Error())
}

func TestBytes(t *testing.T) {
	assert := assert.New(t)
	client := getRedisClient()
	str := New(context.TODO(), client)

	field := "somekey"
	var value []byte = []byte("abc123")

	err := client.Set(context.TODO(), field, value, 0).Err()
	assert.NoError(err)

	val, err := str.Bytes(client.Get(context.TODO(), field).Result())
	assert.NoError(err)
	assert.Equal(value, val)

	testError := errors.New("test error")
	val, err = str.Bytes(value, testError)
	assert.Error(err, testError.Error())
}

func TestBool(t *testing.T) {
	assert := assert.New(t)
	client := getRedisClient()
	str := New(context.TODO(), client)

	field := "somekey"
	value := true

	err := client.Set(context.TODO(), field, value, 0).Err()
	assert.NoError(err)

	val, err := str.Bool(client.Get(context.TODO(), field).Result())
	assert.NoError(err)
	assert.Equal(value, val)

	testError := errors.New("test error")
	val, err = str.Bool(value, testError)
	assert.Error(err, testError.Error())
}
