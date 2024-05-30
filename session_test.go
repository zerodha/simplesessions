package simplesessions

import (
	"errors"
	"fmt"
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
		errNil            = &Err{code: 2, msg: "nil returned"}
		errAssertType     = &Err{code: 3, msg: "assertion failed"}
		errCustom         = &Err{msg: "custom error"}
	)

	assert.Equal(t, errAs(errInvalidSession), ErrInvalidSession)
	assert.Equal(t, errAs(errAssertType), ErrAssertType)
	assert.Equal(t, errAs(errNil), ErrNil)
	assert.Equal(t, errAs(errCustom), errCustom)
}

func TestHelpers(t *testing.T) {
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

func TestNewSession(t *testing.T) {
	reader := "some reader"
	writer := "some writer"
	mgr := newMockManager(newMockStore())

	sess, err := mgr.NewSession(reader, writer)
	assert.NoError(t, err)
	assert.Equal(t, mgr, sess.manager)
	assert.Equal(t, reader, sess.reader)
	assert.Equal(t, writer, sess.writer)
	assert.Nil(t, sess.cache)
	assert.Equal(t, sess.id, sess.ID())
}

func TestNewSessionErrors(t *testing.T) {
	assert := assert.New(t)

	mgr := New(Options{})
	sess, err := mgr.NewSession(nil, nil)
	assert.Equal("session store not set", err.Error())
	assert.Nil(sess)

	mgr = New(Options{})
	mgr.UseStore(&MockStore{})
	sess, err = mgr.NewSession(nil, nil)
	assert.Equal("`SetCookie` hook not set", err.Error())
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
	mgr.SetCookieHooks(nil, func(*http.Cookie, interface{}) error { return wErr })

	sess, err = mgr.NewSession(nil, nil)
	assert.ErrorIs(wErr, err)
	assert.Nil(sess)

	genErr := fmt.Errorf("generate error")
	gen := func() (string, error) { return "xxx", genErr }
	validate := func(string) bool { return false }
	mgr.SetSessionIDHooks(gen, validate)
	sess, err = mgr.NewSession(nil, nil)
	assert.ErrorIs(genErr, err)
	assert.Nil(sess)
}

func TestNewSessionCreateNewCookie(t *testing.T) {
	mgr := newMockManager(newMockStore())
	sess, err := mgr.NewSession(nil, nil)
	assert.NoError(t, err)
	assert.True(t, mgr.validateID(sess.id))
}

func TestNewSessionSetCookieCb(t *testing.T) {
	var (
		mgr    = newMockManager(newMockStore())
		receCk *http.Cookie
		receWr interface{}
		isCb   bool
	)

	mgr.SetCookieHooks(nil, func(ck *http.Cookie, w interface{}) error {
		receCk = ck
		receWr = w
		isCb = true
		return nil
	})

	var writer = "this is writer interface"
	sess, err := mgr.NewSession(nil, writer)
	assert.NoError(t, err)

	assert.True(t, isCb)
	assert.Equal(t, sess.id, receCk.Value)
	assert.Equal(t, writer, receWr)
}

func TestWriteCookie(t *testing.T) {
	mgr := newMockManager(newMockStore())
	mgr.opts = &Options{
		EnableAutoCreate: false,
		Cookie: CookieOptions{
			Name:       "somename",
			Domain:     "abc.xyz",
			Path:       "/abc/xyz",
			IsHTTPOnly: true,
			IsSecure:   true,
			SameSite:   http.SameSiteDefaultMode,
			MaxAge:     time.Hour,
			Expires:    time.Now(),
		},
	}

	var receCk *http.Cookie
	mgr.SetCookieHooks(nil, func(ck *http.Cookie, w interface{}) error {
		receCk = ck
		return nil
	})
	sess, err := mgr.NewSession(nil, nil)
	assert.NoError(t, err)
	assert.NoError(t, sess.WriteCookie("testvalue"))

	assert.Equal(t, mgr.opts.Cookie.Name, receCk.Name)
	assert.Equal(t, mgr.opts.Cookie.Domain, receCk.Domain)
	assert.Equal(t, mgr.opts.Cookie.Path, receCk.Path)
	assert.Equal(t, mgr.opts.Cookie.IsSecure, receCk.Secure)
	assert.Equal(t, mgr.opts.Cookie.SameSite, receCk.SameSite)
	assert.Equal(t, mgr.opts.Cookie.IsHTTPOnly, receCk.HttpOnly)
	assert.Equal(t, int(mgr.opts.Cookie.MaxAge.Seconds()), receCk.MaxAge)
	assert.Equal(t, mgr.opts.Cookie.Expires, receCk.Expires)
}

func TestClearCookie(t *testing.T) {
	var (
		mgr    = newMockManager(newMockStore())
		receCk *http.Cookie
		isCb   bool
	)
	mgr.SetCookieHooks(nil, func(ck *http.Cookie, w interface{}) error {
		receCk = ck
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

func TestCacheAll(t *testing.T) {
	str := newMockStore()
	str.data = map[string]interface{}{
		"key1": 1,
		"key2": 2,
	}
	mgr := newMockManager(str)

	sess, err := mgr.NewSession(nil, nil)
	assert.NoError(t, err)

	// Test error.
	str.err = errors.New("store error")
	err = sess.CacheAll()
	assert.ErrorIs(t, str.err, err)
	assert.Nil(t, sess.cache)

	// Test without error.
	str.err = nil
	err = sess.CacheAll()
	assert.NoError(t, err)
	assert.Equal(t, str.data, sess.cache)
}

func TestResetCache(t *testing.T) {
	str := newMockStore()
	str.data = map[string]interface{}{
		"key1": 1,
		"key2": 2,
	}
	mgr := newMockManager(str)
	sess, _ := mgr.NewSession(nil, nil)
	sess.CacheAll()
	assert.Equal(t, str.data, sess.cache)

	sess.ResetCache()
	assert.Nil(t, sess.cache)
}

func TestGetStore(t *testing.T) {
	str := newMockStore()
	mgr := newMockManager(str)
	sess, err := mgr.NewSession(nil, nil)
	assert.NoError(t, err)
	str.data = map[string]interface{}{
		"key1": 1,
		"key2": 2,
		"key3": 3,
	}

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

func TestGetCached(t *testing.T) {
	str := newMockStore()
	mgr := newMockManager(str)
	sess, err := mgr.NewSession(nil, nil)
	assert.NoError(t, err)

	sess.cache = map[string]interface{}{
		"key1": 1,
		"key2": 2,
		"key3": 3,
	}

	// GetAll.
	v1, err := sess.GetAll()
	assert.NoError(t, err)
	assert.Equal(t, sess.cache, v1)

	// GetMulti.
	v2, err := sess.GetMulti("key1", "key2")
	assert.NoError(t, err)
	assert.Contains(t, v2, "key1")
	assert.Equal(t, sess.cache["key1"], v2["key1"])
	assert.Contains(t, v2, "key2")
	assert.Equal(t, sess.cache["key2"], v2["key2"])
	assert.NotContains(t, v2, "key3")

	// Get.
	v3, err := sess.Get("key1")
	assert.NoError(t, err)
	assert.Contains(t, sess.cache, "key1")
	assert.Equal(t, sess.cache["key1"], v3)

	// Get unknowm field.
	v3, err = sess.Get("key99")
	assert.NoError(t, err)
	assert.Nil(t, v3)

	// GetMulti unknown fields
	v4, err := sess.GetMulti("key1", "key2", "key99", "key100")
	assert.NoError(t, err)
	assert.Contains(t, v4, "key1")
	assert.Equal(t, sess.cache["key1"], v4["key1"])
	assert.Contains(t, v4, "key99")
	assert.Contains(t, v4, "key100")

	v5, ok := v4["key99"]
	assert.True(t, ok)
	assert.Nil(t, v5)

	v5, ok = v4["key100"]
	assert.True(t, ok)
	assert.Nil(t, v5)
}

func TestSet(t *testing.T) {
	str := newMockStore()
	str.data = map[string]interface{}{}
	mgr := newMockManager(str)
	sess, err := mgr.NewSession(nil, nil)
	assert.NoError(t, err)

	err = sess.Set("key1", 1)
	assert.NoError(t, err)

	// Check if its set on data after commit.
	assert.Contains(t, str.data, "key1")
	assert.Equal(t, 1, str.data["key1"])
	assert.Nil(t, sess.cache)

	// Cache and set.
	err = sess.CacheAll()
	assert.NoError(t, err)
	err = sess.Set("key1", 1)
	assert.NoError(t, err)
	assert.NotNil(t, sess.cache)
	assert.Equal(t, sess.cache, str.data)
}

func TestSetMulti(t *testing.T) {
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

	// Check if its set on data after commit.
	assert.Contains(t, str.data, "key1")
	assert.Contains(t, str.data, "key2")
	assert.Contains(t, str.data, "key3")
	assert.Equal(t, data["key1"], str.data["key1"])
	assert.Equal(t, data["key2"], str.data["key2"])
	assert.Equal(t, data["key3"], str.data["key3"])
	assert.Nil(t, sess.cache)

	// Cache and set.
	str.data = map[string]interface{}{}
	err = sess.CacheAll()
	assert.NoError(t, err)
	err = sess.SetMulti(data)
	assert.NoError(t, err)
	assert.NotNil(t, sess.cache)
	assert.Equal(t, sess.cache, str.data)

	// Test error.
	sess.ResetCache()
	str.err = errors.New("store error")
	err = sess.SetMulti(data)
	assert.ErrorIs(t, str.err, err)
}

func TestDelete(t *testing.T) {
	str := newMockStore()
	mgr := newMockManager(str)
	sess, err := mgr.NewSession(nil, nil)
	assert.NoError(t, err)
	str.data = map[string]interface{}{
		"key1": 1,
		"key2": 2,
		"key3": 3,
	}

	assert.Contains(t, str.data, "key1")
	err = sess.Delete("key1")
	assert.NoError(t, err)
	assert.NotContains(t, str.data, "key1")

	// Cache and set.
	err = sess.CacheAll()
	assert.NoError(t, err)
	err = sess.Delete("key2")
	assert.NoError(t, err)
	assert.NotNil(t, sess.cache)
	assert.Equal(t, sess.cache, str.data)

	// Test error.
	str.err = errors.New("store error")
	err = sess.Delete("key2")
	assert.ErrorIs(t, str.err, err)
}

func TestClear(t *testing.T) {
	// Test errors.
	str := newMockStore()
	mgr := newMockManager(str)
	sess, err := mgr.NewSession(nil, nil)
	assert.NoError(t, err)
	str.err = errors.New("store error")
	err = sess.Clear()
	assert.ErrorIs(t, str.err, err)

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
	assert.Nil(t, sess.cache)

	// Test clear.
	str = newMockStore()
	str.data = map[string]interface{}{
		"key1": 1,
		"key2": 2,
	}
	mgr = newMockManager(str)
	sess, err = mgr.NewSession(nil, nil)
	assert.NoError(t, err)
	err = sess.CacheAll()
	assert.NoError(t, err)
	assert.NotNil(t, sess.cache)
	err = sess.Clear()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(str.data))
	assert.Nil(t, sess.cache)
}

func TestDestroy(t *testing.T) {
	// Test errors.
	str := newMockStore()
	mgr := newMockManager(str)
	sess, err := mgr.NewSession(nil, nil)
	assert.NoError(t, err)
	str.err = errors.New("store error")
	err = sess.Destroy()
	assert.ErrorIs(t, str.err, err)

	// Test cookie write error.
	str.err = nil
	ckErr := errors.New("cookie error")
	mgr.SetCookieHooks(nil, func(*http.Cookie, interface{}) error { return ckErr })

	str.data = map[string]interface{}{"foo": "bar"}
	err = sess.Destroy()
	assert.ErrorIs(t, ckErr, err)

	// Test clear.
	str = newMockStore()
	mgr = newMockManager(str)
	sess, err = mgr.NewSession(nil, nil)
	str.data = map[string]interface{}{
		"key1": 1,
		"key2": 2,
	}
	assert.NoError(t, err)
	err = sess.Destroy()
	assert.NoError(t, err)
	assert.Nil(t, str.data)
	assert.Nil(t, sess.cache)

	// Test clear.
	str = newMockStore()
	mgr = newMockManager(str)
	sess, err = mgr.NewSession(nil, nil)
	str.data = map[string]interface{}{
		"key1": 1,
		"key2": 2,
	}
	assert.NoError(t, err)
	err = sess.CacheAll()
	assert.NoError(t, err)
	assert.NotNil(t, sess.cache)
	err = sess.Clear()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(str.data))
	assert.Nil(t, sess.cache)

	// Test deleteCookie callback.
	var (
		receCk *http.Cookie
		isCb   bool
	)
	mgr.SetCookieHooks(nil, func(ck *http.Cookie, w interface{}) error {
		receCk = ck
		isCb = true
		return nil
	})
	err = sess.Destroy()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(str.data))
	assert.True(t, isCb)
	assert.NotNil(t, receCk)
	assert.Greater(t, time.Now(), receCk.Expires)
}
