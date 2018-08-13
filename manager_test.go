package simplesessions

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewManagerWithDefaultOptions(t *testing.T) {
	m := New(Options{})

	assert := assert.New(t)
	// Default cookie path is set to root
	assert.Equal(m.opts.CookiePath, "/")
	// Default cookie name is set
	assert.Equal(m.opts.CookieName, defaultCookieName)
}

func TestManagerNewManagerWithOptions(t *testing.T) {
	opts := Options{
		DisableAutoSet:   true,
		CookieName:       "testcookiename",
		CookieDomain:     "somedomain",
		CookiePath:       "/abc/123",
		IsSecureCookie:   true,
		IsHTTPOnlyCookie: true,
		CookieLifetime:   2000 * time.Millisecond,
	}

	m := New(opts)

	assert := assert.New(t)

	// Default cookie path is set to root
	assert.Equal(m.opts.DisableAutoSet, opts.DisableAutoSet)
	assert.Equal(m.opts.CookieName, opts.CookieName)
	assert.Equal(m.opts.CookieDomain, opts.CookieDomain)
	assert.Equal(m.opts.CookiePath, opts.CookiePath)
	assert.Equal(m.opts.IsSecureCookie, opts.IsSecureCookie)
	assert.Equal(m.opts.IsHTTPOnlyCookie, opts.IsHTTPOnlyCookie)
	assert.Equal(m.opts.CookieLifetime, opts.CookieLifetime)
}

func TestManagerUseStore(t *testing.T) {
	assert := assert.New(t)
	mockStr := &MockStore{}
	assert.Implements((*Store)(nil), mockStr)

	m := New(Options{})
	m.UseStore(mockStr)
	assert.Equal(m.store, mockStr)
}

func TestManagerRegisterGetCookie(t *testing.T) {
	assert := assert.New(t)
	m := New(Options{})

	testCookie := &http.Cookie{
		Name: "testcookie",
	}

	cb := func(string, interface{}) (*http.Cookie, error) {
		return testCookie, http.ErrNoCookie
	}

	m.RegisterGetCookie(cb)

	expectCbRes, expectCbErr := cb("", nil)
	actualCbRes, actualCbErr := m.getCookieCb("", nil)

	assert.Equal(expectCbRes, actualCbRes)
	assert.Equal(expectCbErr, actualCbErr)
}

func TestManagerRegisterSetCookie(t *testing.T) {
	assert := assert.New(t)
	m := New(Options{})

	testCookie := &http.Cookie{
		Name: "testcookie",
	}

	cb := func(*http.Cookie, interface{}) error {
		return http.ErrNoCookie
	}

	m.RegisterSetCookie(cb)

	expectCbErr := cb(testCookie, nil)
	actualCbErr := m.setCookieCb(testCookie, nil)

	assert.Equal(expectCbErr, actualCbErr)
}

func TestManagerAcquireFails(t *testing.T) {
	assert := assert.New(t)
	m := New(Options{})

	_, err := m.Acquire(nil, nil, nil)
	assert.Error(err, "session store is not set")

	m.UseStore(&MockStore{})
	_, err = m.Acquire(nil, nil, nil)
	assert.Error(err, "callback `GetCookie` not set")

	getCb := func(string, interface{}) (*http.Cookie, error) {
		return nil, nil
	}
	m.RegisterGetCookie(getCb)
	_, err = m.Acquire(nil, nil, nil)
	assert.Error(err, "callback `SetCookie` not set")
}

func TestManagerAcquireSucceeds(t *testing.T) {
	m := New(Options{})
	m.UseStore(&MockStore{
		isValid: true,
	})

	getCb := func(string, interface{}) (*http.Cookie, error) {
		return &http.Cookie{
			Name:  "testcookie",
			Value: "",
		}, nil
	}
	m.RegisterGetCookie(getCb)

	setCb := func(*http.Cookie, interface{}) error {
		return http.ErrNoCookie
	}
	m.RegisterSetCookie(setCb)

	_, err := m.Acquire(nil, nil, nil)
	assert := assert.New(t)
	assert.NoError(err)
}

func TestManagerAcquireFromContext(t *testing.T) {
	assert := assert.New(t)
	m := New(Options{})
	m.UseStore(&MockStore{
		isValid: true,
	})

	getCb := func(string, interface{}) (*http.Cookie, error) {
		return &http.Cookie{
			Name:  "testcookie",
			Value: "",
		}, nil
	}
	m.RegisterGetCookie(getCb)

	setCb := func(*http.Cookie, interface{}) error {
		return http.ErrNoCookie
	}
	m.RegisterSetCookie(setCb)

	sess, err := m.Acquire(nil, nil, nil)
	assert.NoError(err)
	sess.cookie.Value = "updated"

	sessNew, err := m.Acquire(nil, nil, nil)
	assert.NoError(err)
	assert.NotEqual(sessNew.cookie.Value, sess.cookie.Value)

	ctx := context.Background()
	ctx = context.WithValue(ctx, ContextName, sess)
	sessNext, err := m.Acquire(nil, nil, ctx)
	assert.Equal(sessNext.cookie.Value, sess.cookie.Value)
}
