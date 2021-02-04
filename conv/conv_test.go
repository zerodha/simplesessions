package conv

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vividvilla/simplesessions"
)

var (
	errTest = errors.New("test error")
)

func TestInt(t *testing.T) {
	assert := assert.New(t)

	v, err := Int(1, nil)
	assert.NoError(err)
	assert.Equal(1, v)

	v, err = Int("1", nil)
	assert.NoError(err)
	assert.Equal(1, v)

	v, err = Int([]byte("1"), nil)
	assert.NoError(err)
	assert.Equal(1, v)

	var tVal int64 = 1
	v, err = Int(tVal, nil)
	assert.NoError(err)
	assert.Equal(1, v)

	var tVal1 interface{} = 1
	v, err = Int(tVal1, nil)
	assert.NoError(err)
	assert.Equal(1, v)

	// Test if ErrNil is returned if value is nil.
	v, err = Int(nil, nil)
	assert.Error(err, simplesessions.ErrNil)
	assert.Equal(0, v)

	// Test if custom error sent is returned.
	v, err = Int(nil, errTest)
	assert.Error(err, errTest)
	assert.Equal(0, v)

	// Test invalid assert error.
	v, err = Int(10.1112, nil)
	assert.Error(err, simplesessions.ErrAssertType)
	assert.Equal(0, v)
}

func TestInt64(t *testing.T) {
	assert := assert.New(t)

	v, err := Int64(int64(1), nil)
	assert.NoError(err)
	assert.Equal(int64(1), v)

	v, err = Int64("1", nil)
	assert.NoError(err)
	assert.Equal(int64(1), v)

	v, err = Int64([]byte("1"), nil)
	assert.NoError(err)
	assert.Equal(int64(1), v)

	var tVal interface{} = 1
	v, err = Int64(tVal, nil)
	assert.NoError(err)
	assert.Equal(int64(1), v)

	// Test if ErrNil is returned if value is nil.
	v, err = Int64(nil, nil)
	assert.Error(err, simplesessions.ErrNil)
	assert.Equal(int64(0), v)

	// Test if custom error sent is returned.
	v, err = Int64(nil, errTest)
	assert.Error(err, errTest)
	assert.Equal(int64(0), v)

	// Test invalid assert error.
	v, err = Int64(10.1112, nil)
	assert.Error(err, simplesessions.ErrAssertType)
	assert.Equal(int64(0), v)
}

func TestUInt64(t *testing.T) {
	assert := assert.New(t)

	v, err := UInt64(uint64(1), nil)
	assert.NoError(err)
	assert.Equal(uint64(1), v)

	v, err = UInt64("1", nil)
	assert.NoError(err)
	assert.Equal(uint64(1), v)

	v, err = UInt64([]byte("1"), nil)
	assert.NoError(err)
	assert.Equal(uint64(1), v)

	var tVal interface{} = 1
	v, err = UInt64(tVal, nil)
	assert.NoError(err)
	assert.Equal(uint64(1), v)

	// Test if ErrNil is returned if value is nil.
	v, err = UInt64(nil, nil)
	assert.Error(err, simplesessions.ErrNil)
	assert.Equal(uint64(0), v)

	// Test if custom error sent is returned.
	v, err = UInt64(nil, errTest)
	assert.Error(err, errTest)
	assert.Equal(uint64(0), v)

	// Test invalid assert error.
	v, err = UInt64(10.1112, nil)
	assert.Error(err, simplesessions.ErrAssertType)
	assert.Equal(uint64(0), v)
}

func TestFloat64(t *testing.T) {
	assert := assert.New(t)

	v, err := Float64(float64(1.11), nil)
	assert.NoError(err)
	assert.Equal(float64(1.11), v)

	v, err = Float64("1.11", nil)
	assert.NoError(err)
	assert.Equal(float64(1.11), v)

	v, err = Float64([]byte("1.11"), nil)
	assert.NoError(err)
	assert.Equal(float64(1.11), v)

	var tVal float64 = 1.11
	v, err = Float64(tVal, nil)
	assert.NoError(err)
	assert.Equal(float64(1.11), v)

	// Test if ErrNil is returned if value is nil.
	v, err = Float64(nil, nil)
	assert.Error(err, simplesessions.ErrNil)
	assert.Equal(float64(0), v)

	// Test if custom error sent is returned.
	v, err = Float64(nil, errTest)
	assert.Error(err, errTest)
	assert.Equal(float64(0), v)

	// Test invalid assert error.
	v, err = Float64("abc", nil)
	assert.Error(err, simplesessions.ErrAssertType)
	assert.Equal(float64(0), v)
}

func TestString(t *testing.T) {
	assert := assert.New(t)

	v, err := String("abc", nil)
	assert.NoError(err)
	assert.Equal("abc", v)

	v, err = String([]byte("abc"), nil)
	assert.NoError(err)
	assert.Equal("abc", v)

	var tVal interface{} = "abc"
	v, err = String(tVal, nil)
	assert.NoError(err)
	assert.Equal("abc", v)

	// Test if ErrNil is returned if value is nil.
	v, err = String(nil, nil)
	assert.Error(err, simplesessions.ErrNil)
	assert.Equal("", v)

	// Test if custom error sent is returned.
	v, err = String(nil, errTest)
	assert.Error(err, errTest)
	assert.Equal("", v)

	// Test invalid assert error.
	v, err = String(10.1112, nil)
	assert.Error(err, simplesessions.ErrAssertType)
	assert.Equal("", v)
}

func TestBytes(t *testing.T) {
	assert := assert.New(t)

	v, err := Bytes("abc", nil)
	assert.NoError(err)
	assert.Equal([]byte("abc"), v)

	v, err = Bytes([]byte("abc"), nil)
	assert.NoError(err)
	assert.Equal([]byte("abc"), v)

	var tVal interface{} = "abc"
	v, err = Bytes(tVal, nil)
	assert.NoError(err)
	assert.Equal([]byte("abc"), v)

	// Test if ErrNil is returned if value is nil.
	v, err = Bytes(nil, nil)
	assert.Error(err, simplesessions.ErrNil)
	assert.Equal([]byte(nil), v)

	// Test if custom error sent is returned.
	v, err = Bytes(nil, errTest)
	assert.Error(err, errTest)
	assert.Equal([]byte(nil), v)

	// Test invalid assert error.
	v, err = Bytes(10.1112, nil)
	assert.Error(err, simplesessions.ErrAssertType)
	assert.Equal([]byte(nil), v)
}

func TestBool(t *testing.T) {
	assert := assert.New(t)

	v, err := Bool(true, nil)
	assert.NoError(err)
	assert.Equal(true, v)

	v, err = Bool(false, nil)
	assert.NoError(err)
	assert.Equal(false, v)

	v, err = Bool(0, nil)
	assert.NoError(err)
	assert.Equal(false, v)

	v, err = Bool(1, nil)
	assert.NoError(err)
	assert.Equal(true, v)

	v, err = Bool(int64(0), nil)
	assert.NoError(err)
	assert.Equal(false, v)

	v, err = Bool(int64(1), nil)
	assert.NoError(err)
	assert.Equal(true, v)

	v, err = Bool([]byte("true"), nil)
	assert.NoError(err)
	assert.Equal(true, v)

	v, err = Bool([]byte("false"), nil)
	assert.NoError(err)
	assert.Equal(false, v)

	v, err = Bool("true", nil)
	assert.NoError(err)
	assert.Equal(true, v)

	v, err = Bool("false", nil)
	assert.NoError(err)
	assert.Equal(false, v)

	// Test if ErrNil is returned if value is nil.
	v, err = Bool(nil, nil)
	assert.Error(err, simplesessions.ErrNil)
	assert.Equal(false, v)

	// Test if custom error sent is returned.
	v, err = Bool(nil, errTest)
	assert.Error(err, errTest)
	assert.Equal(false, v)

	// Test invalid assert error.
	v, err = Bool(10.1112, nil)
	assert.Error(err, simplesessions.ErrAssertType)
	assert.Equal(false, v)
}
