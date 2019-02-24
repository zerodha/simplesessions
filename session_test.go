package simplesessions

import (
	"errors"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	testCookieName  = "sometestcookie"
	testCookieValue = "sometestcookievalue"
)

func newMockStore() *MockStore {
	return &MockStore{}
}

func newMockManager(store *MockStore) *Manager {
	mockManager := New(Options{})
	mockManager.UseStore(store)
	mockManager.RegisterGetCookie(getCookieCb)
	mockManager.RegisterSetCookie(setCookieCb)

	return mockManager
}

func getCookieCb(name string, r interface{}) (*http.Cookie, error) {
	return &http.Cookie{
		Name:  name,
		Value: testCookieValue,
	}, nil
}

func setCookieCb(*http.Cookie, interface{}) error {
	return nil
}

func TestSessionHelpers(t *testing.T) {
	assert := assert.New(t)
	sess := Session{
		manager: newMockManager(newMockStore()),
	}

	// Int
	var inp1 = 100
	v1, err := sess.Int(inp1, errors.New("test error"))
	assert.Equal(v1, inp1)
	assert.Error(err, "test error")

	// Int64
	var inp2 int64 = 100
	v2, err := sess.Int64(inp2, errors.New("test error"))
	assert.Equal(v2, inp2)
	assert.Error(err, "test error")

	var inp3 uint64 = 100
	v3, err := sess.UInt64(inp3, errors.New("test error"))
	assert.Equal(v3, inp3)
	assert.Error(err, "test error")

	var inp4 float64 = 100
	v4, err := sess.Float64(inp4, errors.New("test error"))
	assert.Equal(v4, inp4)
	assert.Error(err, "test error")

	var inp5 = "abc123"
	v5, err := sess.String(inp5, errors.New("test error"))
	assert.Equal(v5, inp5)
	assert.Error(err, "test error")

	var inp6 = true
	v6, err := sess.Bool(inp6, errors.New("test error"))
	assert.Equal(v6, inp6)
	assert.Error(err, "test error")

	var inp7 = []byte{}
	v7, err := sess.Bytes(inp7, errors.New("test error"))
	assert.Equal(v7, inp7)
	assert.Error(err, "test error")
}

func TestSessionGenerateRandomString(t *testing.T) {
	assert := assert.New(t)
	sess := Session{}

	// Create random length string
	rand.Seed(time.Now().Unix())
	strLen := rand.Intn(1000-1) + 1
	randStr, err := sess.GenerateRandomString(strLen)
	assert.NoError(err)
	assert.Equal(strLen, len(randStr))

	// Check if it doesn't generate same id
	randStr1, err := sess.GenerateRandomString(100)
	assert.NoError(err)
	randStr2, err := sess.GenerateRandomString(100)
	assert.NoError(err)
	assert.NotEqual(randStr1, randStr2)
}

func TestSessionisAlphaNum(t *testing.T) {
	assert := assert.New(t)
	sess := Session{}

	assert.Equal(sess.isAlphaNum("thisisvalidstring"), true)
	assert.Equal(sess.isAlphaNum("thisisNot$ a .validstring"), false)
}

func TestSessionIsValidRandomString(t *testing.T) {
	assert := assert.New(t)
	sess := Session{}

	randStr, err := sess.GenerateRandomString(100)
	assert.NoError(err)
	assert.Equal(sess.IsValidRandomString(randStr), true)
	assert.Equal(sess.IsValidRandomString(dictionary), true)
}

func TestSessionNewSession(t *testing.T) {
	reader := "some reader"
	writer := "some writer"
	mockStore := newMockStore()
	mockStore.isValid = true
	mockManager := newMockManager(mockStore)

	assert := assert.New(t)
	sess, err := NewSession(mockManager, reader, writer)
	assert.NoError(err)
	assert.Equal(sess.manager, mockManager)
	assert.Equal(sess.reader, reader)
	assert.Equal(sess.writer, writer)
	assert.NotNil(sess.values)
	assert.NotNil(sess.cookie)
	assert.Equal(sess.cookie.Name, defaultCookieName)
	assert.Equal(sess.cookie.Value, testCookieValue)
	assert.True(sess.isSet)
}

func TestSessionNewSessionInvalidSession(t *testing.T) {
	assert := assert.New(t)
	mockStore := newMockStore()
	mockManager := newMockManager(mockStore)

	sess, err := NewSession(mockManager, nil, nil)
	assert.Error(err, ErrInvalidSession.Error())
	assert.Nil(sess)
}

func TestSessionNewSessionErrorStoreCreate(t *testing.T) {
	assert := assert.New(t)
	mockStore := newMockStore()
	mockStore.isValid = true

	testError := errors.New("this is test error")
	newCookieVal := "somerandomid"
	mockStore.val = newCookieVal
	mockStore.err = testError
	mockManager := newMockManager(mockStore)
	mockManager.RegisterGetCookie(func(name string, r interface{}) (*http.Cookie, error) {
		return nil, http.ErrNoCookie
	})

	sess, err := NewSession(mockManager, nil, nil)
	assert.Error(err, testError.Error())
	assert.Nil(sess)
}

func TestSessionNewSessionErrorWriteCookie(t *testing.T) {
	assert := assert.New(t)
	mockStore := newMockStore()
	mockStore.isValid = true

	testError := errors.New("this is test error")
	newCookieVal := "somerandomid"
	mockStore.val = newCookieVal
	mockManager := newMockManager(mockStore)
	mockManager.RegisterGetCookie(func(name string, r interface{}) (*http.Cookie, error) {
		return nil, http.ErrNoCookie
	})
	mockManager.RegisterSetCookie(func(cookie *http.Cookie, w interface{}) error {
		return testError
	})

	sess, err := NewSession(mockManager, nil, nil)
	assert.Error(err, testError.Error())
	assert.Nil(sess)
}

func TestSessionNewSessionErrorStoreIsValid(t *testing.T) {
	assert := assert.New(t)
	mockStore := newMockStore()
	mockManager := newMockManager(mockStore)

	testError := errors.New("this is test error")
	mockStore.err = testError

	sess, err := NewSession(mockManager, nil, nil)
	assert.Error(err, testError.Error())
	assert.Nil(sess)
}

func TestSessionNewSessionInvalidGetCookie(t *testing.T) {
	assert := assert.New(t)
	mockStore := newMockStore()
	mockManager := newMockManager(mockStore)
	testError := errors.New("custom error")
	mockManager.RegisterGetCookie(func(name string, r interface{}) (*http.Cookie, error) {
		return nil, testError
	})

	sess, err := NewSession(mockManager, nil, nil)
	assert.Error(err, testError.Error())
	assert.Nil(sess)
}

func TestSessionNewSessionCreateNewCookie(t *testing.T) {
	assert := assert.New(t)
	mockStore := newMockStore()

	newCookieVal := "somerandomid"
	mockStore.val = newCookieVal
	mockStore.isValid = true
	mockManager := newMockManager(mockStore)
	mockManager.RegisterGetCookie(func(name string, r interface{}) (*http.Cookie, error) {
		return nil, http.ErrNoCookie
	})

	sess, err := NewSession(mockManager, nil, nil)
	assert.NoError(err)
	assert.True(sess.isSet)
	assert.Equal(sess.cookie.Value, newCookieVal)
}

func TestSessionNewSessionWithDisableAutoSet(t *testing.T) {
	assert := assert.New(t)
	mockStore := newMockStore()

	mockManager := newMockManager(mockStore)
	mockManager.opts.DisableAutoSet = true
	mockManager.RegisterGetCookie(func(name string, r interface{}) (*http.Cookie, error) {
		return nil, http.ErrNoCookie
	})

	sess, err := NewSession(mockManager, nil, nil)
	assert.NoError(err)
	assert.False(sess.isSet)
}

func TestSessionNewSessionGetCookieCb(t *testing.T) {
	assert := assert.New(t)
	mockStore := newMockStore()

	// Calls write cookie callback if cookie is not set already
	newCookieVal := "somerandomid"
	mockStore.val = newCookieVal
	mockStore.isValid = true
	mockManager := newMockManager(mockStore)

	var receivedName string
	var receivedReader interface{}
	var isCallbackTriggered bool
	mockManager.RegisterGetCookie(func(name string, r interface{}) (*http.Cookie, error) {
		isCallbackTriggered = true
		receivedName = name
		receivedReader = r
		return nil, http.ErrNoCookie
	})

	var reader = "this is reader interface"
	sess, err := NewSession(mockManager, reader, nil)
	assert.NoError(err)
	assert.True(sess.isSet)
	assert.True(isCallbackTriggered)
	assert.Equal(receivedName, mockManager.opts.CookieName)
	assert.Equal(receivedReader, reader)
}

func TestSessionNewSessionSetCookieCb(t *testing.T) {
	assert := assert.New(t)
	mockStore := newMockStore()

	// Calls write cookie callback if cookie is not set already
	newCookieVal := "somerandomid"
	mockStore.val = newCookieVal
	mockStore.isValid = true
	mockManager := newMockManager(mockStore)
	mockManager.RegisterGetCookie(func(name string, r interface{}) (*http.Cookie, error) {
		return nil, http.ErrNoCookie
	})

	var receivedCookie *http.Cookie
	var receivedWriter interface{}
	var isCallbackTriggered bool
	mockManager.RegisterSetCookie(func(cookie *http.Cookie, w interface{}) error {
		receivedCookie = cookie
		receivedWriter = w
		isCallbackTriggered = true
		return nil
	})

	var writer = "this is writer interface"
	sess, err := NewSession(mockManager, nil, writer)
	assert.NoError(err)
	assert.True(sess.isSet)
	assert.True(isCallbackTriggered)
	assert.Equal(receivedCookie.Value, newCookieVal)
	assert.Equal(receivedWriter, writer)
}

func TestSessionWriteCookie(t *testing.T) {
	assert := assert.New(t)
	mockStore := newMockStore()
	mockManager := newMockManager(mockStore)
	mockManager.opts = &Options{
		CookieName:       "somename",
		CookieDomain:     "abc.xyz",
		CookiePath:       "/abc/xyz",
		CookieLifetime:   time.Second * 1000,
		IsHTTPOnlyCookie: true,
		IsSecureCookie:   true,
		DisableAutoSet:   true,
	}
	mockStore.isValid = true

	sess, err := NewSession(mockManager, nil, nil)
	assert.NoError(err)

	sess.WriteCookie("testvalue")
	assert.Equal(sess.cookie.Name, mockManager.opts.CookieName)
	assert.Equal(sess.cookie.Value, "testvalue")
	assert.Equal(sess.cookie.Domain, mockManager.opts.CookieDomain)
	assert.Equal(sess.cookie.Path, mockManager.opts.CookiePath)
	assert.Equal(sess.cookie.Secure, mockManager.opts.IsSecureCookie)
	assert.Equal(sess.cookie.HttpOnly, mockManager.opts.IsHTTPOnlyCookie)

	// Ignore seconds
	expiry := time.Now().Add(mockManager.opts.CookieLifetime)
	assert.Equal(sess.cookie.Expires.Format("2006-01-02 15:04:05"), expiry.Format("2006-01-02 15:04:05"))
	assert.WithinDuration(expiry, sess.cookie.Expires, time.Millisecond*1000)
}

func TestSessionClearCookie(t *testing.T) {
	assert := assert.New(t)
	mockStore := newMockStore()
	mockManager := newMockManager(mockStore)
	mockStore.isValid = true

	var receivedCookie *http.Cookie
	var isCallbackTriggered bool
	mockManager.RegisterSetCookie(func(cookie *http.Cookie, w interface{}) error {
		receivedCookie = cookie
		isCallbackTriggered = true
		return nil
	})

	sess, err := NewSession(mockManager, nil, nil)
	assert.NoError(err)

	err = sess.clearCookie()
	assert.NoError(err)

	assert.True(isCallbackTriggered)
	assert.Equal(receivedCookie.Value, "")
	assert.True(receivedCookie.Expires.UnixNano() < time.Now().UnixNano())
}

func TestSessionCreate(t *testing.T) {
	assert := assert.New(t)
	mockStore := newMockStore()
	mockStore.isValid = true
	mockStore.val = "test"
	mockManager := newMockManager(mockStore)
	mockManager.opts.DisableAutoSet = true
	mockManager.RegisterGetCookie(func(name string, r interface{}) (*http.Cookie, error) {
		return nil, http.ErrNoCookie
	})

	var isCallbackTriggered bool
	mockManager.RegisterSetCookie(func(cookie *http.Cookie, w interface{}) error {
		isCallbackTriggered = true
		return nil
	})

	sess, err := NewSession(mockManager, nil, nil)
	assert.NoError(err)
	assert.False(sess.isSet)
	assert.False(isCallbackTriggered)

	err = sess.Create()
	assert.NoError(err)
	assert.True(isCallbackTriggered)
	assert.True(sess.isSet)
}

func TestSessionLoadValues(t *testing.T) {
	mockStore := newMockStore()
	mockStore.isValid = true
	mockStore.val = 100
	mockManager := newMockManager(mockStore)

	assert := assert.New(t)
	sess, err := NewSession(mockManager, nil, nil)
	assert.NoError(err)

	err = sess.LoadValues()
	assert.NoError(err)
	assert.Contains(sess.values, "val")
	assert.Equal(sess.values["val"], 100)
}

func TestSessionResetValues(t *testing.T) {
	mockStore := newMockStore()
	mockStore.isValid = true
	mockStore.val = 100
	mockManager := newMockManager(mockStore)

	assert := assert.New(t)
	sess, err := NewSession(mockManager, nil, nil)
	assert.NoError(err)

	err = sess.LoadValues()
	assert.NoError(err)
	assert.Contains(sess.values, "val")
	assert.Equal(sess.values["val"], 100)

	sess.ResetValues()
	assert.Equal(len(sess.values), 0)
}

func TestSessionGetAllFromStore(t *testing.T) {
	mockStore := newMockStore()
	mockStore.isValid = true
	mockStore.val = 100
	mockManager := newMockManager(mockStore)

	assert := assert.New(t)
	sess, err := NewSession(mockManager, nil, nil)
	assert.NoError(err)

	vals, err := sess.GetAll()
	assert.NoError(err)
	assert.Contains(vals, "val")
	assert.Equal(vals["val"], 100)
}

func TestSessionGetAllLoadedValues(t *testing.T) {
	mockStore := newMockStore()
	mockStore.isValid = true
	mockManager := newMockManager(mockStore)

	assert := assert.New(t)
	sess, err := NewSession(mockManager, nil, nil)
	assert.NoError(err)

	setVals := make(map[string]interface{})
	setVals["sample"] = "someval"
	sess.values = setVals

	vals, err := sess.GetAll()
	assert.NoError(err)
	assert.Contains(vals, "sample")
	assert.Equal(vals["sample"], "someval")
}

func TestSessionGetAllInvalidSession(t *testing.T) {
	mockStore := newMockStore()
	mockManager := newMockManager(mockStore)
	mockManager.opts.DisableAutoSet = true
	mockManager.RegisterGetCookie(func(name string, r interface{}) (*http.Cookie, error) {
		return nil, http.ErrNoCookie
	})

	assert := assert.New(t)
	sess, err := NewSession(mockManager, nil, nil)
	assert.NoError(err)

	vals, err := sess.GetAll()
	assert.Error(err, ErrInvalidSession.Error())
	assert.Nil(vals)
}

func TestSessionGetMultiFromStore(t *testing.T) {
	mockStore := newMockStore()
	mockStore.isValid = true
	mockStore.val = 100
	mockManager := newMockManager(mockStore)

	assert := assert.New(t)
	sess, err := NewSession(mockManager, nil, nil)
	assert.NoError(err)

	vals, err := sess.GetMulti("val")
	assert.NoError(err)
	assert.Contains(vals, "val")
	assert.Equal(vals["val"], 100)
}

func TestSessionGetMultiLoadedValues(t *testing.T) {
	mockStore := newMockStore()
	mockStore.isValid = true
	mockManager := newMockManager(mockStore)

	assert := assert.New(t)
	sess, err := NewSession(mockManager, nil, nil)
	assert.NoError(err)

	setVals := make(map[string]interface{})
	setVals["key1"] = "someval"
	setVals["key2"] = "someval"
	sess.values = setVals

	vals, err := sess.GetMulti("key1")
	assert.NoError(err)
	assert.Contains(vals, "key1")
	assert.Equal(vals["key1"], "someval")
	assert.NotContains(vals, "key2")
}

func TestSessionGetMultiInvalidSession(t *testing.T) {
	mockStore := newMockStore()
	mockManager := newMockManager(mockStore)
	mockManager.opts.DisableAutoSet = true
	mockManager.RegisterGetCookie(func(name string, r interface{}) (*http.Cookie, error) {
		return nil, http.ErrNoCookie
	})

	assert := assert.New(t)
	sess, err := NewSession(mockManager, nil, nil)
	assert.NoError(err)

	vals, err := sess.GetMulti("val")
	assert.Error(err, ErrInvalidSession.Error())
	assert.Nil(vals)
}

func TestSessionGetFromStore(t *testing.T) {
	mockStore := newMockStore()
	mockStore.isValid = true
	mockStore.val = 100
	mockManager := newMockManager(mockStore)

	assert := assert.New(t)
	sess, err := NewSession(mockManager, nil, nil)
	assert.NoError(err)

	val, err := sess.Get("val")
	assert.NoError(err)
	assert.Equal(val, 100)
}

func TestSessionGetLoadedValues(t *testing.T) {
	mockStore := newMockStore()
	mockStore.isValid = true
	mockManager := newMockManager(mockStore)

	assert := assert.New(t)
	sess, err := NewSession(mockManager, nil, nil)
	assert.NoError(err)

	setVals := make(map[string]interface{})
	setVals["key1"] = "someval1"
	setVals["key2"] = "someval2"
	sess.values = setVals

	val, err := sess.Get("key1")
	assert.NoError(err)
	assert.Equal(val, "someval1")
}

func TestSessionGetInvalidSession(t *testing.T) {
	mockStore := newMockStore()
	mockManager := newMockManager(mockStore)
	mockManager.opts.DisableAutoSet = true
	mockManager.RegisterGetCookie(func(name string, r interface{}) (*http.Cookie, error) {
		return nil, http.ErrNoCookie
	})

	assert := assert.New(t)
	sess, err := NewSession(mockManager, nil, nil)
	assert.NoError(err)

	vals, err := sess.Get("val")
	assert.Error(err, ErrInvalidSession.Error())
	assert.Nil(vals)
}

func TestSessionSet(t *testing.T) {
	mockStore := newMockStore()
	mockStore.isValid = true
	mockManager := newMockManager(mockStore)

	assert := assert.New(t)
	sess, err := NewSession(mockManager, nil, nil)
	assert.NoError(err)

	err = sess.Set("key", 100)
	assert.NoError(err)
	assert.Equal(mockStore.val, 100)
}

func TestSessionSetInvalidSession(t *testing.T) {
	mockStore := newMockStore()
	mockManager := newMockManager(mockStore)
	mockManager.opts.DisableAutoSet = true
	mockManager.RegisterGetCookie(func(name string, r interface{}) (*http.Cookie, error) {
		return nil, http.ErrNoCookie
	})

	assert := assert.New(t)
	sess, err := NewSession(mockManager, nil, nil)
	assert.NoError(err)

	err = sess.Set("key", 100)
	assert.Error(err, ErrInvalidSession.Error())
}

func TestSessionCommit(t *testing.T) {
	mockStore := newMockStore()
	mockStore.isValid = true
	mockManager := newMockManager(mockStore)

	assert := assert.New(t)
	sess, err := NewSession(mockManager, nil, nil)
	assert.NoError(err)

	err = sess.Set("key", 100)
	assert.NoError(err)
	assert.NoError(err)
	assert.False(mockStore.isCommited)
	err = sess.Commit()
	assert.NoError(err)
	assert.True(mockStore.isCommited)
}

func TestSessionCommitInvalidSession(t *testing.T) {
	mockStore := newMockStore()
	mockManager := newMockManager(mockStore)
	mockManager.opts.DisableAutoSet = true
	mockManager.RegisterGetCookie(func(name string, r interface{}) (*http.Cookie, error) {
		return nil, http.ErrNoCookie
	})

	assert := assert.New(t)
	sess, err := NewSession(mockManager, nil, nil)
	assert.NoError(err)

	err = sess.Commit()
	assert.Error(err, ErrInvalidSession.Error())
}

func TestSessionDelete(t *testing.T) {
	assert := assert.New(t)
	mockStore := newMockStore()
	mockManager := newMockManager(mockStore)
	mockStore.isValid = true
	mockStore.val = 100

	sess, err := NewSession(mockManager, nil, nil)
	assert.NoError(err)
	assert.Equal(mockStore.val, 100)

	err = sess.Delete("somekey")
	assert.NoError(err)
	assert.Nil(mockStore.val)

	testError := errors.New("this is test error")
	mockStore.err = testError
	err = sess.Delete("somekey")
	assert.Error(err, testError.Error())
}

func TestSessionClear(t *testing.T) {
	assert := assert.New(t)
	mockStore := newMockStore()
	mockManager := newMockManager(mockStore)
	mockStore.isValid = true
	mockStore.val = 100

	var isCallbackTriggered bool
	mockManager.RegisterSetCookie(func(cookie *http.Cookie, w interface{}) error {
		isCallbackTriggered = true
		return nil
	})

	sess, err := NewSession(mockManager, nil, nil)
	assert.NoError(err)
	assert.Equal(mockStore.val, 100)

	err = sess.Clear()
	assert.NoError(err)

	assert.True(isCallbackTriggered)
	assert.Equal(mockStore.val, nil)
}

func TestSessionClearError(t *testing.T) {
	assert := assert.New(t)
	mockStore := newMockStore()
	mockManager := newMockManager(mockStore)
	mockStore.isValid = true

	sess, err := NewSession(mockManager, nil, nil)
	assert.NoError(err)

	testError := errors.New("this is test error")
	mockStore.err = testError
	err = sess.Clear()
	assert.Error(err, testError.Error())
}

func TestSessionClearInvalidSession(t *testing.T) {
	mockStore := newMockStore()
	mockManager := newMockManager(mockStore)
	mockManager.opts.DisableAutoSet = true
	mockManager.RegisterGetCookie(func(name string, r interface{}) (*http.Cookie, error) {
		return nil, http.ErrNoCookie
	})

	assert := assert.New(t)
	sess, err := NewSession(mockManager, nil, nil)
	assert.NoError(err)

	err = sess.Clear()
	assert.Error(err, ErrInvalidSession.Error())
}
