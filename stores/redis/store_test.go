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
	client := getRedisClient()
	ctx := context.Background()
	str := New(ctx, client)
	assert.Equal(t, str.prefix, defaultPrefix)
	assert.Equal(t, str.client, client)
	assert.Equal(t, str.clientCtx, ctx)
}

func TestSetPrefix(t *testing.T) {
	str := New(context.TODO(), getRedisClient())
	str.SetPrefix("test")
	assert.Equal(t, str.prefix, "test")
}

func TestSetTTL(t *testing.T) {
	testDur := time.Second * 10
	str := New(context.TODO(), getRedisClient())
	str.SetTTL(testDur)
	assert.Equal(t, str.ttl, testDur)
}

func TestCreate(t *testing.T) {
	str := New(context.TODO(), getRedisClient())
	id, err := str.Create()
	assert.Nil(t, err)
	assert.Equal(t, len(id), sessionIDLen)
}

func TestGet(t *testing.T) {
	key := "4dIHy6S2uBuKaNnTUszB218L898ikGY1"
	field := "somekey"
	value := 100
	client := getRedisClient()

	// Set a key
	err := client.HSet(context.TODO(), defaultPrefix+key, field, value).Err()
	assert.NoError(t, err)

	str := New(context.TODO(), client)

	val, err := str.Int(str.Get(key, field))
	assert.NoError(t, err)
	assert.Equal(t, val, value)

	// Check for invalid key.
	_, err = str.Int(str.Get(key, "invalidfield"))
	assert.ErrorIs(t, ErrFieldNotFound, err)
}

func TestGetInvalidSession(t *testing.T) {
	str := New(context.TODO(), getRedisClient())
	val, err := str.Get("invalidkey", "invalidkey")
	assert.Nil(t, val)
	assert.ErrorIs(t, err, ErrInvalidSession)

	id := "10IHy6S2uBuKaNnTUszB218L898ikGY1"
	val, err = str.Get(id, "invalidkey")
	assert.Nil(t, val)
	assert.ErrorIs(t, ErrInvalidSession, err)
}

func TestGetMultiInvalidSession(t *testing.T) {
	str := New(context.TODO(), getRedisClient())
	val, err := str.GetMulti("invalidkey", "invalidkey")
	assert.Nil(t, val)
	assert.ErrorIs(t, ErrInvalidSession, err)

	key := "11IHy6S2uBuKaNnTUszB218L898ikGY1"
	field := "somefield"
	_, err = str.GetMulti(key, field)
	assert.ErrorIs(t, err, ErrInvalidSession)
}

func TestGetMulti(t *testing.T) {
	var (
		key          = "5dIHy6S2uBuKaNnTUszB218L898ikGY1"
		field1       = "somekey"
		value1       = 100
		field2       = "someotherkey"
		value2       = "abc123"
		field3       = "thishouldntbethere"
		value3       = 100.10
		invalidField = "foo"
		client       = getRedisClient()
	)

	// Set a key
	err := client.HMSet(context.TODO(), defaultPrefix+key, field1, value1, field2, value2, field3, value3).Err()
	assert.NoError(t, err)

	str := New(context.TODO(), client)
	vals, err := str.GetMulti(key, field1, field2, invalidField)
	assert.NoError(t, err)
	assert.Contains(t, vals, field1)
	assert.Contains(t, vals, field2)
	assert.NotContains(t, vals, field3)

	val1, err := str.Int(vals[field1], nil)
	assert.NoError(t, err)
	assert.Equal(t, val1, value1)

	val2, err := str.String(vals[field2], nil)
	assert.NoError(t, err)
	assert.Equal(t, val2, value2)

	// Check for invalid key.
	_, err = str.String(vals[invalidField], nil)
	assert.ErrorIs(t, ErrFieldNotFound, err)
}

func TestGetAllInvalidSession(t *testing.T) {
	str := New(context.TODO(), getRedisClient())
	val, err := str.GetAll("invalidkey")
	assert.Nil(t, val)
	assert.ErrorIs(t, ErrInvalidSession, err)

	key := "11IHy6S2uBuKaNnTUszB218L898ikGY1"
	val, err = str.GetAll(key)
	assert.Nil(t, val)
	assert.ErrorIs(t, ErrInvalidSession, err)
}

func TestGetAll(t *testing.T) {
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
	assert.NoError(t, err)

	str := New(context.TODO(), client)

	vals, err := str.GetAll(key)
	assert.NoError(t, err)
	assert.Contains(t, vals, field1)
	assert.Contains(t, vals, field2)
	assert.Contains(t, vals, field3)

	val1, err := str.Int(vals[field1], nil)
	assert.NoError(t, err)
	assert.Equal(t, val1, value1)

	val2, err := str.String(vals[field2], nil)
	assert.NoError(t, err)
	assert.Equal(t, val2, value2)

	val3, err := str.Float64(vals[field3], nil)
	assert.NoError(t, err)
	assert.Equal(t, val3, value3)
}

func TestSetInvalidSessionError(t *testing.T) {
	str := New(context.TODO(), getRedisClient())
	err := str.Set("invalidid", "key", "value")
	assert.ErrorIs(t, ErrInvalidSession, err)
}

func TestSet(t *testing.T) {
	// Test should only set in internal map and not in redis
	client := getRedisClient()
	str := New(context.TODO(), client)
	ttl := time.Second * 10
	str.SetTTL(ttl)

	// this key is unique across all tests
	key := "7dIHy6S2uBuKaNnTUszB218L898ikGY9"
	field := "somekey"
	value := 100

	err := str.Set(key, field, value)
	assert.NoError(t, err)

	// Check ifs not commited to redis
	v1, err := client.Exists(context.TODO(), defaultPrefix+key).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), v1)

	v2, err := str.Int(client.HGet(context.TODO(), defaultPrefix+key, field).Result())
	assert.NoError(t, err)
	assert.Equal(t, value, v2)

	dur, err := client.TTL(context.TODO(), defaultPrefix+key).Result()
	assert.NoError(t, err)
	assert.Equal(t, dur, ttl)
}

func TestSetMulti(t *testing.T) {
	// Test should only set in internal map and not in redis
	client := getRedisClient()
	str := New(context.TODO(), client)
	ttl := time.Second * 10
	str.SetTTL(ttl)

	// this key is unique across all tests
	key := "7dIHy6S2uBuKaNnTUszB218L898ikGY9"
	field1 := "somekey1"
	value1 := 100
	field2 := "somekey2"
	value2 := "somevalue"

	err := str.SetMulti(key, map[string]interface{}{
		field1: value1,
		field2: value2,
	})
	assert.NoError(t, err)

	// Check ifs not commited to redis
	v1, err := client.Exists(context.TODO(), defaultPrefix+key).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), v1)

	v2, err := str.Int(client.HGet(context.TODO(), defaultPrefix+key, field1).Result())
	assert.NoError(t, err)
	assert.Equal(t, value1, v2)

	dur, err := client.TTL(context.TODO(), defaultPrefix+key).Result()
	assert.NoError(t, err)
	assert.Equal(t, dur, ttl)
}

func TestDeleteInvalidSessionError(t *testing.T) {
	str := New(context.TODO(), getRedisClient())
	err := str.Delete("invalidkey", "somefield")
	assert.ErrorIs(t, ErrInvalidSession, err)

	str = New(context.TODO(), getRedisClient())
	err = str.Delete("8dIHy6S2uBuKaNnTUszB2180898ikGY1", "somefield")
	assert.ErrorIs(t, ErrInvalidSession, err)
}

func TestDelete(t *testing.T) {
	// Test should only set in internal map and not in redis
	client := getRedisClient()
	str := New(context.TODO(), client)

	// this key is unique across all tests
	key := "8dIHy6S2uBuKaNnTUszB2180898ikGY1"
	field1 := "somekey"
	value1 := 100
	field2 := "someotherkey"
	value2 := "abc123"

	err := client.HMSet(context.TODO(), defaultPrefix+key, field1, value1, field2, value2).Err()
	assert.NoError(t, err)

	err = str.Delete(key, field1)
	assert.NoError(t, err)

	val, err := client.HExists(context.TODO(), defaultPrefix+key, field1).Result()
	assert.False(t, val)
	assert.NoError(t, err)

	val, err = client.HExists(context.TODO(), defaultPrefix+key, field2).Result()
	assert.True(t, val)
	assert.NoError(t, err)

	err = str.Delete(key, "xxxxx")
	assert.ErrorIs(t, err, ErrFieldNotFound)
}

func TestClearInvalidSessionError(t *testing.T) {
	str := New(context.TODO(), getRedisClient())
	err := str.Clear("invalidkey")
	assert.ErrorIs(t, ErrInvalidSession, err)
}

func TestClear(t *testing.T) {
	// Test should only set in internal map and not in redis
	client := getRedisClient()
	str := New(context.TODO(), client)

	// this key is unique across all tests
	key := "8dIHy6S2uBuKaNnTUszB2180898ikGY1"
	field1 := "somekey"
	value1 := 100
	field2 := "someotherkey"
	value2 := "abc123"

	err := client.HMSet(context.TODO(), defaultPrefix+key, field1, value1, field2, value2).Err()
	assert.NoError(t, err)

	// Check if its set
	val, err := client.Exists(context.TODO(), defaultPrefix+key).Result()
	assert.NoError(t, err)
	assert.NotEqual(t, val, int64(0))

	err = str.Clear(key)
	assert.NoError(t, err)

	val, err = client.Exists(context.TODO(), defaultPrefix+key).Result()
	assert.NoError(t, err)
	assert.Equal(t, val, int64(0))
}

func TestInt(t *testing.T) {
	client := getRedisClient()
	str := New(context.TODO(), client)

	field := "somekey"
	value := 100

	err := client.Set(context.TODO(), field, value, 0).Err()
	assert.NoError(t, err)

	val, err := str.Int(client.Get(context.TODO(), field).Result())
	assert.NoError(t, err)
	assert.Equal(t, value, val)

	testError := errors.New("test error")
	_, err = str.Int(value, testError)
	assert.ErrorIs(t, testError, err)
}

func TestInt64(t *testing.T) {
	client := getRedisClient()
	str := New(context.TODO(), client)

	field := "somekey"
	var value int64 = 100

	err := client.Set(context.TODO(), field, value, 0).Err()
	assert.NoError(t, err)

	val, err := str.Int64(client.Get(context.TODO(), field).Result())
	assert.NoError(t, err)
	assert.Equal(t, value, val)

	testError := errors.New("test error")
	_, err = str.Int64(value, testError)
	assert.ErrorIs(t, testError, err)
}

func TestUInt64(t *testing.T) {
	client := getRedisClient()
	str := New(context.TODO(), client)

	field := "somekey"
	var value uint64 = 100

	err := client.Set(context.TODO(), field, value, 0).Err()
	assert.NoError(t, err)

	val, err := str.UInt64(client.Get(context.TODO(), field).Result())
	assert.NoError(t, err)
	assert.Equal(t, value, val)

	testError := errors.New("test error")
	_, err = str.UInt64(value, testError)
	assert.ErrorIs(t, testError, err)
}

func TestFloat64(t *testing.T) {
	client := getRedisClient()
	str := New(context.TODO(), client)

	field := "somekey"
	var value float64 = 100

	err := client.Set(context.TODO(), field, value, 0).Err()
	assert.NoError(t, err)

	val, err := str.Float64(client.Get(context.TODO(), field).Result())
	assert.NoError(t, err)
	assert.Equal(t, value, val)

	testError := errors.New("test error")
	_, err = str.Float64(value, testError)
	assert.ErrorIs(t, testError, err)
}

func TestString(t *testing.T) {
	client := getRedisClient()
	str := New(context.TODO(), client)

	field := "somekey"
	value := "abc123"

	err := client.Set(context.TODO(), field, value, 0).Err()
	assert.NoError(t, err)

	val, err := str.String(client.Get(context.TODO(), field).Result())
	assert.NoError(t, err)
	assert.Equal(t, value, val)

	testError := errors.New("test error")
	_, err = str.String(value, testError)
	assert.ErrorIs(t, testError, err)
}

func TestBytes(t *testing.T) {
	client := getRedisClient()
	str := New(context.TODO(), client)

	field := "somekey"
	var value []byte = []byte("abc123")

	err := client.Set(context.TODO(), field, value, 0).Err()
	assert.NoError(t, err)

	val, err := str.Bytes(client.Get(context.TODO(), field).Result())
	assert.NoError(t, err)
	assert.Equal(t, value, val)

	testError := errors.New("test error")
	_, err = str.Bytes(value, testError)
	assert.ErrorIs(t, testError, err)
}

func TestBool(t *testing.T) {
	client := getRedisClient()
	str := New(context.TODO(), client)

	field := "somekey"
	value := true

	err := client.Set(context.TODO(), field, value, 0).Err()
	assert.NoError(t, err)

	val, err := str.Bool(client.Get(context.TODO(), field).Result())
	assert.NoError(t, err)
	assert.Equal(t, value, val)

	testError := errors.New("test error")
	_, err = str.Bool(value, testError)
	assert.ErrorIs(t, testError, err)
}

func TestValidateID(t *testing.T) {
	ok := validateID("xxxx")
	assert.False(t, ok)

	ok = validateID("8dIHy6S2uBuKaNnTUszB2180898ikGY&")
	assert.False(t, ok)

	id, err := generateID(sessionIDLen)
	assert.NoError(t, err)
	ok = validateID(id)
	assert.True(t, ok)
}
