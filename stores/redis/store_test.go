package redis

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
	errTest   = errors.New("test error")
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
	str.SetTTL(testDur, true)
	assert.Equal(t, str.ttl, testDur)
	assert.True(t, str.extendTTL)
}

func TestCreate(t *testing.T) {
	var (
		id     = "testid_create"
		client = getRedisClient()
		str    = New(context.TODO(), client)
	)
	str.SetTTL(time.Second*100, false)
	err := str.Create(id)
	assert.Nil(t, err)

	vals, err := client.HGetAll(context.TODO(), str.prefix+id).Result()
	assert.NoError(t, err)
	assert.Contains(t, vals, defaultSessKey)

	ttl, _ := client.TTL(context.TODO(), str.prefix+id).Result()
	assert.Equal(t, ttl, time.Second*100)
}

func TestGet(t *testing.T) {
	var (
		id     = "testid_get"
		field  = "somekey"
		value  = 100
		client = getRedisClient()
		str    = New(context.TODO(), client)
	)
	// Invalid session.
	val, err := str.Get("invalidkey", "invalidkey")
	assert.Nil(t, val)
	assert.ErrorIs(t, err, ErrInvalidSession)

	// Check valid session.
	err = client.HMSet(context.TODO(), str.prefix+id, field, value, defaultSessKey, "1").Err()
	assert.NoError(t, err)

	val, err = str.Int(str.Get(id, field))
	assert.NoError(t, err)
	assert.Equal(t, val, value)

	// Check for invalid key.
	_, err = str.Int(str.Get(id, "invalidfield"))
	assert.ErrorIs(t, ErrNil, err)
}

func TestGetMulti(t *testing.T) {
	var (
		id           = "testid_getmulti"
		field1       = "somekey"
		value1       = 100
		field2       = "someotherkey"
		value2       = "abc123"
		invalidField = "foo"
		client       = getRedisClient()
		str          = New(context.TODO(), client)
	)
	// Invalid session.
	val, err := str.GetMulti("invalidkey", "invalidkey")
	assert.Nil(t, val)
	assert.ErrorIs(t, err, ErrInvalidSession)

	// Set a key
	err = client.HMSet(context.TODO(), str.prefix+id, defaultSessKey, "1", field1, value1, field2, value2).Err()
	assert.NoError(t, err)

	vals, err := str.GetMulti(id, field1, field2, invalidField)
	assert.NoError(t, err)
	assert.Contains(t, vals, field1)
	assert.Contains(t, vals, field2)
	assert.Contains(t, vals, invalidField)

	val1, err := str.Int(vals[field1], nil)
	assert.NoError(t, err)
	assert.Equal(t, val1, value1)

	val2, err := str.String(vals[field2], nil)
	assert.NoError(t, err)
	assert.Equal(t, val2, value2)

	// Check for invalid key.
	_, err = str.String(vals[invalidField], nil)
	assert.ErrorIs(t, ErrNil, err)
}

func TestGetAll(t *testing.T) {
	var (
		key    = "testid_getall"
		field1 = "somekey"
		value1 = 100
		field2 = "someotherkey"
		value2 = "abc123"
		client = getRedisClient()
		str    = New(context.TODO(), client)
	)

	// Set a key
	err := client.HMSet(context.TODO(), str.prefix+key, defaultSessKey, "1", field1, value1, field2, value2).Err()
	assert.NoError(t, err)

	vals, err := str.GetAll(key)
	assert.NoError(t, err)
	assert.Contains(t, vals, field1)
	assert.Contains(t, vals, field2)

	val1, err := str.Int(vals[field1], nil)
	assert.NoError(t, err)
	assert.Equal(t, val1, value1)

	val2, err := str.String(vals[field2], nil)
	assert.NoError(t, err)
	assert.Equal(t, val2, value2)
}

func TestSet(t *testing.T) {
	// Test should only set in internal map and not in redis
	var (
		client = getRedisClient()
		str    = New(context.TODO(), client)
		ttl    = time.Second * 10
		// this key is unique across all tests
		key   = "testid_set"
		field = "somekey"
		value = 100
	)
	str.SetTTL(ttl, true)

	err := str.Set(key, field, value)
	assert.NoError(t, err)

	// Check ifs not commited to redis
	v1, err := client.Exists(context.TODO(), str.prefix+key).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), v1)

	v2, err := str.Int(client.HGet(context.TODO(), str.prefix+key, field).Result())
	assert.NoError(t, err)
	assert.Equal(t, value, v2)

	dur, err := client.TTL(context.TODO(), str.prefix+key).Result()
	assert.NoError(t, err)
	assert.Equal(t, dur, ttl)
}

func TestSetMulti(t *testing.T) {
	// Test should only set in internal map and not in redis
	var (
		client = getRedisClient()
		str    = New(context.TODO(), client)
		ttl    = time.Second * 10
		key    = "testid_setmulti"
		field1 = "somekey1"
		value1 = 100
		field2 = "somekey2"
		value2 = "somevalue"
	)
	str.SetTTL(ttl, true)

	err := str.SetMulti(key, map[string]interface{}{
		field1: value1,
		field2: value2,
	})
	assert.NoError(t, err)

	// Check ifs not commited to redis
	v1, err := client.Exists(context.TODO(), str.prefix+key).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), v1)

	v2, err := str.Int(client.HGet(context.TODO(), str.prefix+key, field1).Result())
	assert.NoError(t, err)
	assert.Equal(t, value1, v2)

	dur, err := client.TTL(context.TODO(), str.prefix+key).Result()
	assert.NoError(t, err)
	assert.Equal(t, dur, ttl)
}

func TestDelete(t *testing.T) {
	// Test should only set in internal map and not in redis
	var (
		client = getRedisClient()
		str    = New(context.TODO(), client)

		// this key is unique across all tests
		key    = "testid_delete"
		field1 = "somekey"
		value1 = 100
		field2 = "someotherkey"
		value2 = "abc123"
	)

	err := client.HMSet(context.TODO(), str.prefix+key, defaultSessKey, "1", field1, value1, field2, value2).Err()
	assert.NoError(t, err)

	err = str.Delete(key, field1)
	assert.NoError(t, err)

	val, err := client.HExists(context.TODO(), str.prefix+key, field1).Result()
	assert.False(t, val)
	assert.NoError(t, err)

	val, err = client.HExists(context.TODO(), str.prefix+key, field2).Result()
	assert.True(t, val)
	assert.NoError(t, err)
}

func TestClear(t *testing.T) {
	// Test should only set in internal map and not in redis
	var (
		client = getRedisClient()
		str    = New(context.TODO(), client)

		// this key is unique across all tests
		key    = "testid_clear"
		field1 = "somekey"
		value1 = 100
		field2 = "someotherkey"
		value2 = "abc123"
	)

	err := client.HMSet(context.TODO(), str.prefix+key, defaultSessKey, "1", field1, value1, field2, value2).Err()
	assert.NoError(t, err)

	err = str.Clear(key)
	assert.NoError(t, err)

	val, err := client.HExists(context.TODO(), str.prefix+key, defaultSessKey).Result()
	assert.NoError(t, err)
	assert.True(t, val)

	val, err = client.HExists(context.TODO(), str.prefix+key, field1).Result()
	assert.NoError(t, err)
	assert.False(t, val)
}

func TestDestroy(t *testing.T) {
	// Test should only set in internal map and not in redis
	var (
		client = getRedisClient()
		str    = New(context.TODO(), client)

		// this key is unique across all tests
		key    = "testid_clear"
		field1 = "somekey"
		value1 = 100
	)

	err := client.HMSet(context.TODO(), str.prefix+key, defaultSessKey, "1", field1, value1).Err()
	assert.NoError(t, err)

	err = str.Destroy(key)
	assert.NoError(t, err)

	val, err := client.Exists(context.TODO(), str.prefix+key).Result()
	assert.NoError(t, err)
	assert.Equal(t, val, int64(0))
}

func TestInt(t *testing.T) {
	str := New(context.TODO(), nil)

	v, err := str.Int(1, nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, v)

	v, err = str.Int("1", nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, v)

	v, err = str.Int([]byte("1"), nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, v)

	var tVal int64 = 1
	v, err = str.Int(tVal, nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, v)

	var tVal1 interface{} = 1
	v, err = str.Int(tVal1, nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, v)

	// Test if ErrNil is returned if value is nil.
	v, err = str.Int(nil, nil)
	assert.ErrorIs(t, err, ErrNil)
	assert.Equal(t, 0, v)

	// Test if custom error sent is returned.
	v, err = str.Int(nil, errTest)
	assert.ErrorIs(t, err, errTest)
	assert.Equal(t, 0, v)

	// Test invalid assert error.
	v, err = str.Int(10.1112, nil)
	assert.ErrorIs(t, err, ErrAssertType)
	assert.Equal(t, 0, v)
}

func TestInt64(t *testing.T) {
	str := New(context.TODO(), nil)

	v, err := str.Int64(int64(1), nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), v)

	v, err = str.Int64("1", nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), v)

	v, err = str.Int64([]byte("1"), nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), v)

	var tVal interface{} = 1
	v, err = str.Int64(tVal, nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), v)

	// Test if ErrNil is returned if value is nil.
	v, err = str.Int64(nil, nil)
	assert.ErrorIs(t, err, ErrNil)
	assert.Equal(t, int64(0), v)

	// Test if custom error sent is returned.
	v, err = str.Int64(nil, errTest)
	assert.ErrorIs(t, err, errTest)
	assert.Equal(t, int64(0), v)

	// Test invalid assert error.
	v, err = str.Int64(10.1112, nil)
	assert.ErrorIs(t, err, ErrAssertType)
	assert.Equal(t, int64(0), v)
}

func TestUInt64(t *testing.T) {
	str := New(context.TODO(), nil)

	v, err := str.UInt64(uint64(1), nil)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), v)

	v, err = str.UInt64("1", nil)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), v)

	v, err = str.UInt64([]byte("1"), nil)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), v)

	var tVal interface{} = 1
	v, err = str.UInt64(tVal, nil)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), v)

	// Test if ErrNil is returned if value is nil.
	v, err = str.UInt64(nil, nil)
	assert.ErrorIs(t, err, ErrNil)
	assert.Equal(t, uint64(0), v)

	// Test if custom error sent is returned.
	v, err = str.UInt64(nil, errTest)
	assert.ErrorIs(t, err, errTest)
	assert.Equal(t, uint64(0), v)

	// Test invalid assert error.
	v, err = str.UInt64(10.1112, nil)
	assert.ErrorIs(t, err, ErrAssertType)
	assert.Equal(t, uint64(0), v)
}

func TestFloat64(t *testing.T) {
	str := New(context.TODO(), nil)

	v, err := str.Float64(float64(1.11), nil)
	assert.NoError(t, err)
	assert.Equal(t, float64(1.11), v)

	v, err = str.Float64("1.11", nil)
	assert.NoError(t, err)
	assert.Equal(t, float64(1.11), v)

	v, err = str.Float64([]byte("1.11"), nil)
	assert.NoError(t, err)
	assert.Equal(t, float64(1.11), v)

	var tVal float64 = 1.11
	v, err = str.Float64(tVal, nil)
	assert.NoError(t, err)
	assert.Equal(t, float64(1.11), v)

	// Test if ErrNil is returned if value is nil.
	v, err = str.Float64(nil, nil)
	assert.ErrorIs(t, err, ErrNil)
	assert.Equal(t, float64(0), v)

	// Test if custom error sent is returned.
	v, err = str.Float64(nil, errTest)
	assert.ErrorIs(t, err, errTest)
	assert.Equal(t, float64(0), v)

	// Test invalid assert error.
	v, err = str.Float64("abc", nil)
	assert.ErrorIs(t, err, ErrAssertType)
	assert.Equal(t, float64(0), v)
}

func TestString(t *testing.T) {
	str := New(context.TODO(), nil)

	v, err := str.String("abc", nil)
	assert.NoError(t, err)
	assert.Equal(t, "abc", v)

	v, err = str.String([]byte("abc"), nil)
	assert.NoError(t, err)
	assert.Equal(t, "abc", v)

	var tVal interface{} = "abc"
	v, err = str.String(tVal, nil)
	assert.NoError(t, err)
	assert.Equal(t, "abc", v)

	// Test if ErrNil is returned if value is nil.
	v, err = str.String(nil, nil)
	assert.ErrorIs(t, err, ErrNil)
	assert.Equal(t, "", v)

	// Test if custom error sent is returned.
	v, err = str.String(nil, errTest)
	assert.ErrorIs(t, err, errTest)
	assert.Equal(t, "", v)

	// Test invalid assert error.
	v, err = str.String(10.1112, nil)
	assert.ErrorIs(t, err, ErrAssertType)
	assert.Equal(t, "", v)
}

func TestBytes(t *testing.T) {
	str := New(context.TODO(), nil)

	v, err := str.Bytes("abc", nil)
	assert.NoError(t, err)
	assert.Equal(t, []byte("abc"), v)

	v, err = str.Bytes([]byte("abc"), nil)
	assert.NoError(t, err)
	assert.Equal(t, []byte("abc"), v)

	var tVal interface{} = "abc"
	v, err = str.Bytes(tVal, nil)
	assert.NoError(t, err)
	assert.Equal(t, []byte("abc"), v)

	// Test if ErrNil is returned if value is nil.
	v, err = str.Bytes(nil, nil)
	assert.ErrorIs(t, err, ErrNil)
	assert.Equal(t, []byte(nil), v)

	// Test if custom error sent is returned.
	v, err = str.Bytes(nil, errTest)
	assert.ErrorIs(t, err, errTest)
	assert.Equal(t, []byte(nil), v)

	// Test invalid assert error.
	v, err = str.Bytes(10.1112, nil)
	assert.ErrorIs(t, err, ErrAssertType)
	assert.Equal(t, []byte(nil), v)
}

func TestBool(t *testing.T) {
	str := New(context.TODO(), nil)

	v, err := str.Bool(true, nil)
	assert.NoError(t, err)
	assert.Equal(t, true, v)

	v, err = str.Bool(false, nil)
	assert.NoError(t, err)
	assert.Equal(t, false, v)

	v, err = str.Bool(0, nil)
	assert.NoError(t, err)
	assert.Equal(t, false, v)

	v, err = str.Bool(1, nil)
	assert.NoError(t, err)
	assert.Equal(t, true, v)

	v, err = str.Bool(int64(0), nil)
	assert.NoError(t, err)
	assert.Equal(t, false, v)

	v, err = str.Bool(int64(1), nil)
	assert.NoError(t, err)
	assert.Equal(t, true, v)

	v, err = str.Bool([]byte("true"), nil)
	assert.NoError(t, err)
	assert.Equal(t, true, v)

	v, err = str.Bool([]byte("false"), nil)
	assert.NoError(t, err)
	assert.Equal(t, false, v)

	v, err = str.Bool("true", nil)
	assert.NoError(t, err)
	assert.Equal(t, true, v)

	v, err = str.Bool("false", nil)
	assert.NoError(t, err)
	assert.Equal(t, false, v)

	// Test if ErrNil is returned if value is nil.
	v, err = str.Bool(nil, nil)
	assert.ErrorIs(t, err, ErrNil)
	assert.Equal(t, false, v)

	// Test if custom error sent is returned.
	v, err = str.Bool(nil, errTest)
	assert.ErrorIs(t, err, errTest)
	assert.Equal(t, false, v)

	// Test invalid assert error.
	v, err = str.Bool(10.1112, nil)
	assert.ErrorIs(t, err, ErrAssertType)
	assert.Equal(t, false, v)
}

func TestError(t *testing.T) {
	err := Err{
		code: 1,
		msg:  "test",
	}
	assert.Equal(t, 1, err.Code())
	assert.Equal(t, "test", err.Error())
}
