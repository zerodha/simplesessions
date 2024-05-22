package simplesessions

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type Err struct {
	code int
	msg  string
}

func (e *Err) Error() string {
	return e.msg
}

func (e *Err) Code() int {
	return e.code
}

func TestErrorTypes(t *testing.T) {
	var (
		// Error codes for store errors. This should match the codes
		// defined in the /simplesessions package exactly.
		errInvalidSession = &Err{code: 1, msg: "invalid session"}
		errFieldNotFound  = &Err{code: 2, msg: "field not found"}
		errAssertType     = &Err{code: 3, msg: "assertion failed"}
		errNil            = &Err{code: 4, msg: "nil returned"}
		errCustom         = &Err{msg: "custom error"}
	)

	assert.Equal(t, errAs(errInvalidSession), ErrInvalidSession)
	assert.Equal(t, errAs(errFieldNotFound), ErrFieldNotFound)
	assert.Equal(t, errAs(errAssertType), ErrAssertType)
	assert.Equal(t, errAs(errNil), ErrNil)
	assert.Equal(t, errAs(errCustom), errCustom)
}

func TestSessionHelpers(t *testing.T) {
	sess := Session{
		manager: newMockManager(newMockStore()),
	}

	// Int
	var inp1 = 100
	v1, err := sess.Int(inp1, errors.New("test error"))
	assert.Equal(t, inp1, v1)
	assert.Equal(t, "test error", err.Error())

	// Int64
	var inp2 int64 = 100
	v2, err := sess.Int64(inp2, errors.New("test error"))
	assert.Equal(t, inp2, v2)
	assert.Equal(t, "test error", err.Error())

	var inp3 uint64 = 100
	v3, err := sess.UInt64(inp3, errors.New("test error"))
	assert.Equal(t, inp3, v3)
	assert.Equal(t, "test error", err.Error())

	var inp4 float64 = 100
	v4, err := sess.Float64(inp4, errors.New("test error"))
	assert.Equal(t, inp4, v4)
	assert.Equal(t, "test error", err.Error())

	var inp5 = "abc123"
	v5, err := sess.String(inp5, errors.New("test error"))
	assert.Equal(t, inp5, v5)
	assert.Equal(t, "test error", err.Error())

	var inp6 = true
	v6, err := sess.Bool(inp6, errors.New("test error"))
	assert.Equal(t, inp6, v6)
	assert.Equal(t, "test error", err.Error())

	var inp7 = []byte{}
	v7, err := sess.Bytes(inp7, errors.New("test error"))
	assert.Equal(t, inp7, v7)
	assert.Equal(t, "test error", err.Error())
}

func TestSessionNewSession(t *testing.T) {
	reader := "some reader"
	writer := "some writer"
	mgr := newMockManager(newMockStore())

	sess, err := mgr.NewSession(reader, writer)
	assert.NoError(t, err)
	assert.Equal(t, mgr, sess.manager)
	assert.Equal(t, reader, sess.reader)
	assert.Equal(t, writer, sess.writer)
	assert.NotNil(t, sess.values)
	assert.Equal(t, mockSessionID, sess.id)
	assert.Equal(t, sess.id, sess.ID())
}

func TestSessionNewSessionErrors(t *testing.T) {
	assert := assert.New(t)

	mgr := New(Options{})
	sess, err := mgr.NewSession(nil, nil)
	assert.Equal("session store is not set", err.Error())
	assert.Nil(sess)

	mgr = New(Options{})
	mgr.UseStore(&MockStore{})
	sess, err = mgr.NewSession(nil, nil)
	assert.Equal("callback `SetCookie` not set", err.Error())
	assert.Nil(sess)

	// Store error.
	tErr := errors.New("store error")
	str := newMockStore()
	str.err = tErr
	mgr = newMockManager(str)
	sess, err = mgr.NewSession(nil, nil)
	assert.ErrorIs(tErr, err)
	assert.Nil(sess)

	// Cookie write error.
	str.err = nil
	wErr := errors.New("write cookie error")
	mgr.RegisterSetCookie(func(cookie *http.Cookie, w interface{}) error {
		return wErr
	})
	sess, err = mgr.NewSession(nil, nil)
	assert.ErrorIs(wErr, err)
	assert.Nil(sess)
}

func TestSessionNewSessionCreateNewCookie(t *testing.T) {
	mgr := newMockManager(newMockStore())
	sess, err := mgr.NewSession(nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, sess.id, mockSessionID)
}

func TestSessionNewSessionSetCookieCb(t *testing.T) {
	var (
		mgr    = newMockManager(newMockStore())
		receCk *http.Cookie
		receWr interface{}
		isCb   bool
	)

	mgr.RegisterSetCookie(func(cookie *http.Cookie, w interface{}) error {
		receCk = cookie
		receWr = w
		isCb = true
		return nil
	})

	var writer = "this is writer interface"
	_, err := mgr.NewSession(nil, writer)
	assert.NoError(t, err)

	assert.True(t, isCb)
	assert.Equal(t, mockSessionID, receCk.Value)
	assert.Equal(t, writer, receWr)
}

func TestSessionWriteCookie(t *testing.T) {
	mgr := newMockManager(newMockStore())
	mgr.opts = &Options{
		CookieName:       "somename",
		CookieDomain:     "abc.xyz",
		CookiePath:       "/abc/xyz",
		CookieLifetime:   time.Second * 1000,
		IsHTTPOnlyCookie: true,
		IsSecureCookie:   true,
		EnableAutoCreate: false,
		SameSite:         http.SameSiteDefaultMode,
	}

	sess, err := mgr.NewSession(nil, nil)
	assert.NoError(t, err)
	assert.NoError(t, sess.WriteCookie("testvalue"))
}

func TestSessionClearCookie(t *testing.T) {
	var (
		mgr    = newMockManager(newMockStore())
		receCk *http.Cookie
		isCb   bool
	)
	mgr.RegisterSetCookie(func(cookie *http.Cookie, w interface{}) error {
		receCk = cookie
		isCb = true
		return nil
	})

	sess, err := mgr.NewSession(nil, nil)
	assert.NoError(t, err)

	err = sess.clearCookie()
	assert.NoError(t, err)

	assert.True(t, isCb)
	assert.Equal(t, "", receCk.Value)
	assert.True(t, receCk.Expires.UnixNano() < time.Now().UnixNano())
}

func TestSessionLoadValues(t *testing.T) {
	str := newMockStore()
	str.data = map[string]interface{}{
		"key1": 1,
		"key2": 2,
	}
	mgr := newMockManager(str)

	sess, err := mgr.NewSession(nil, nil)
	assert.NoError(t, err)

	err = sess.LoadValues()
	assert.NoError(t, err)
	assert.Equal(t, str.data, sess.values)
}

func TestSessionResetValues(t *testing.T) {
	str := newMockStore()
	str.data = map[string]interface{}{
		"key1": 1,
		"key2": 2,
	}
	mgr := newMockManager(str)
	sess, _ := mgr.NewSession(nil, nil)
	sess.LoadValues()
	assert.NotEqual(t, 0, len(sess.values))

	sess.ResetValues()
	assert.Equal(t, 0, len(sess.values))
}

func TestSessionGetStore(t *testing.T) {
	str := newMockStore()
	str.data = map[string]interface{}{
		"key1": 1,
		"key2": 2,
		"key3": 3,
	}
	mgr := newMockManager(str)
	sess, err := mgr.NewSession(nil, nil)
	assert.NoError(t, err)

	// GetAll.
	v1, err := sess.GetAll()
	assert.NoError(t, err)
	assert.Equal(t, str.data, v1)

	// Get Multi.
	v2, err := sess.GetMulti("key1", "key2")
	assert.NoError(t, err)
	assert.Contains(t, v2, "key1")
	assert.Equal(t, str.data["key1"], v2["key1"])
	assert.Contains(t, v2, "key2")
	assert.Equal(t, str.data["key2"], v2["key2"])
	assert.NotContains(t, v2, "key3")

	// Get.
	v3, err := sess.Get("key1")
	assert.NoError(t, err)
	assert.Contains(t, str.data, "key1")
	assert.Equal(t, str.data["key1"], v3)
}

func TestSessionGetLoaded(t *testing.T) {
	str := newMockStore()
	mgr := newMockManager(str)
	sess, err := mgr.NewSession(nil, nil)
	assert.NoError(t, err)

	sess.values = map[string]interface{}{
		"key1": 1,
		"key2": 2,
		"key3": 3,
	}

	// GetAll.
	v1, err := sess.GetAll()
	assert.NoError(t, err)
	assert.Equal(t, sess.values, v1)

	// GetMulti.
	v2, err := sess.GetMulti("key1", "key2")
	assert.NoError(t, err)
	assert.Contains(t, v2, "key1")
	assert.Equal(t, sess.values["key1"], v2["key1"])
	assert.Contains(t, v2, "key2")
	assert.Equal(t, sess.values["key2"], v2["key2"])
	assert.NotContains(t, v2, "key3")

	// Get.
	v3, err := sess.Get("key1")
	assert.NoError(t, err)
	assert.Contains(t, sess.values, "key1")
	assert.Equal(t, sess.values["key1"], v3)
}

func TestSessionSet(t *testing.T) {
	str := newMockStore()
	str.data = map[string]interface{}{}
	mgr := newMockManager(str)
	sess, err := mgr.NewSession(nil, nil)
	assert.NoError(t, err)

	err = sess.Set("key1", 1)
	assert.NoError(t, err)

	// Check if its set on temp.
	assert.Contains(t, str.temp, "key1")
	assert.NotContains(t, str.data, "key1")
	assert.Equal(t, str.temp["key1"], 1)

	// Commit.
	err = sess.Commit()
	assert.NoError(t, err)

	// Check if its set on data after commit.
	assert.Contains(t, str.data, "key1")
	assert.NotContains(t, str.temp, "key1")
	assert.Equal(t, 1, str.data["key1"])
}

func TestSessionSetMulti(t *testing.T) {
	str := newMockStore()
	str.data = map[string]interface{}{}
	mgr := newMockManager(str)
	sess, err := mgr.NewSession(nil, nil)
	assert.NoError(t, err)

	data := map[string]interface{}{
		"key1": 1,
		"key2": 2,
		"key3": 3,
	}
	err = sess.SetMulti(data)
	assert.NoError(t, err)

	// Check if its set on temp.
	assert.Contains(t, str.temp, "key1")
	assert.Contains(t, str.temp, "key2")
	assert.Contains(t, str.temp, "key3")
	assert.NotContains(t, str.data, "key1")
	assert.NotContains(t, str.data, "key2")
	assert.NotContains(t, str.data, "key3")
	assert.Equal(t, data["key1"], str.temp["key1"])
	assert.Equal(t, data["key2"], str.temp["key2"])
	assert.Equal(t, data["key3"], str.temp["key3"])

	// Commit.
	err = sess.Commit()
	assert.NoError(t, err)

	// Check if its set on data after commit.
	assert.Contains(t, str.data, "key1")
	assert.Contains(t, str.data, "key2")
	assert.Contains(t, str.data, "key3")
	assert.NotContains(t, str.temp, "key1")
	assert.NotContains(t, str.temp, "key2")
	assert.NotContains(t, str.temp, "key3")
	assert.Equal(t, data["key1"], str.data["key1"])
	assert.Equal(t, data["key2"], str.data["key2"])
	assert.Equal(t, data["key3"], str.data["key3"])

	// Test error.
	str.err = errors.New("store error")
	err = sess.SetMulti(data)
	assert.ErrorIs(t, str.err, err)

	// Test error.
	str.err = nil
	err = sess.SetMulti(data)
	assert.NoError(t, err)
	str.err = errors.New("store error")
	err = sess.Commit()
	assert.ErrorIs(t, str.err, err)
}

func TestSessionDelete(t *testing.T) {
	str := newMockStore()
	str.data = map[string]interface{}{
		"key1": 1,
		"key2": 2,
		"key3": 3,
	}
	mgr := newMockManager(str)
	sess, err := mgr.NewSession(nil, nil)
	assert.NoError(t, err)

	assert.Contains(t, str.data, "key1")
	err = sess.Delete("key1")
	assert.NoError(t, err)
	assert.NotContains(t, str.data, "key1")

	// Test error.
	str.err = errors.New("store error")
	err = sess.Delete("key2")
	assert.ErrorIs(t, str.err, err)
}

func TestSessionClear(t *testing.T) {
	// Test errors.
	str := newMockStore()
	mgr := newMockManager(str)
	sess, err := mgr.NewSession(nil, nil)
	assert.NoError(t, err)
	str.err = errors.New("store error")
	err = sess.Clear()
	assert.ErrorIs(t, str.err, err)

	// Test cookie write error.
	str.err = nil
	ckErr := errors.New("cookie error")
	mgr.RegisterSetCookie(func(cookie *http.Cookie, w interface{}) error {
		return ckErr
	})
	err = sess.Clear()
	assert.ErrorIs(t, ckErr, err)

	// Test clear.
	str = newMockStore()
	str.data = map[string]interface{}{
		"key1": 1,
		"key2": 2,
	}
	mgr = newMockManager(str)
	sess, err = mgr.NewSession(nil, nil)
	assert.NoError(t, err)
	err = sess.Clear()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(str.data))

	// Test deleteCookie callback.
	var (
		receCk *http.Cookie
		isCb   bool
	)
	mgr.RegisterSetCookie(func(cookie *http.Cookie, w interface{}) error {
		receCk = cookie
		isCb = true
		return nil
	})
	err = sess.Clear()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(str.data))
	assert.True(t, isCb)
	assert.Greater(t, time.Now(), receCk.Expires)
}
