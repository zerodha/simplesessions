package simplesessions

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const mockSessionID = "sometestcookievalue"

func newMockStore() *MockStore {
	return &MockStore{
		id:   mockSessionID,
		data: map[string]interface{}{},
		err:  nil,
	}
}

func newMockManager(store *MockStore) *Manager {
	m := New(Options{})
	m.UseStore(store)
	m.RegisterGetCookie(mockGetCookieCb)
	m.RegisterSetCookie(mockSetCookieCb)
	return m
}

func mockGetCookieCb(name string, r interface{}) (*http.Cookie, error) {
	return &http.Cookie{
		Name:  name,
		Value: mockSessionID,
	}, nil
}

func mockSetCookieCb(*http.Cookie, interface{}) error {
	return nil
}

func TestNewManagerWithDefaultOptions(t *testing.T) {
	m := New(Options{})

	assert := assert.New(t)
	// Default cookie path is set to root
	assert.Equal("/", m.opts.CookiePath)
	// Default cookie name is set
	assert.Equal(defaultCookieName, m.opts.CookieName)
}

func TestManagerNewManagerWithOptions(t *testing.T) {
	opts := Options{
		EnableAutoCreate: true,
		CookieName:       "testcookiename",
		CookieDomain:     "somedomain",
		CookiePath:       "/abc/123",
		IsSecureCookie:   true,
		IsHTTPOnlyCookie: true,
		SameSite:         http.SameSiteLaxMode,
		CookieLifetime:   2000 * time.Millisecond,
	}

	m := New(opts)

	assert := assert.New(t)

	// Default cookie path is set to root
	assert.Equal(opts.EnableAutoCreate, m.opts.EnableAutoCreate)
	assert.Equal(opts.CookieName, m.opts.CookieName)
	assert.Equal(opts.CookieDomain, m.opts.CookieDomain)
	assert.Equal(opts.CookiePath, m.opts.CookiePath)
	assert.Equal(opts.IsSecureCookie, m.opts.IsSecureCookie)
	assert.Equal(opts.SameSite, m.opts.SameSite)
	assert.Equal(opts.IsHTTPOnlyCookie, m.opts.IsHTTPOnlyCookie)
	assert.Equal(opts.CookieLifetime, m.opts.CookieLifetime)
}

func TestManagerUseStore(t *testing.T) {
	assert := assert.New(t)
	s := newMockStore()
	m := newMockManager(s)
	assert.Equal(s, m.store)
}

func TestManagerRegisterGetCookie(t *testing.T) {
	assert := assert.New(t)
	m := New(Options{})

	ck := &http.Cookie{
		Name: "testcookie",
	}

	cb := func(string, interface{}) (*http.Cookie, error) {
		return ck, http.ErrNoCookie
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

	ck := &http.Cookie{
		Name: "testcookie",
	}

	cb := func(*http.Cookie, interface{}) error {
		return http.ErrNoCookie
	}

	m.RegisterSetCookie(cb)

	expectCbErr := cb(ck, nil)
	actualCbErr := m.setCookieCb(ck, nil)

	assert.Equal(expectCbErr, actualCbErr)
}

func TestManagerAcquireFails(t *testing.T) {
	assert := assert.New(t)
	m := New(Options{})

	// Fail if store is not assigned.
	_, err := m.Acquire(context.Background(), nil, nil)
	assert.Equal("session store is not set", err.Error())

	// Fail if getCookie callback is not assigned.
	m.UseStore(&MockStore{})
	_, err = m.Acquire(context.Background(), nil, nil)
	assert.Equal("callback `GetCookie` not set", err.Error())

	// Assign getCookie, returns nil cookie to make sure it
	// fails in create session with invalid session.
	m.RegisterGetCookie(func(string, interface{}) (*http.Cookie, error) {
		return nil, nil
	})

	// Fail if setCookie callback is not assigned.
	_, err = m.Acquire(context.Background(), nil, nil)
	assert.Equal("callback `SetCookie` not set", err.Error())

	// Register setCookie callback.
	m.RegisterSetCookie(func(*http.Cookie, interface{}) error {
		return nil
	})

	// By default EnableAutoCreate is disabled
	// Check if it returns invalid session.
	_, err = m.Acquire(context.Background(), nil, nil)
	assert.ErrorIs(err, ErrInvalidSession)
}

func TestManagerAcquireAutocreate(t *testing.T) {
	m := newMockManager(newMockStore())
	// Enable autocreate.
	m.opts.EnableAutoCreate = true
	m.RegisterGetCookie(func(string, interface{}) (*http.Cookie, error) {
		return nil, ErrInvalidSession
	})

	// If cookie doesn't exist then should return a new one without error.
	sess, err := m.Acquire(context.Background(), nil, nil)
	assert := assert.New(t)
	assert.NoError(err)
	assert.Equal(mockSessionID, sess.id)
}

func TestManagerAcquireFromContext(t *testing.T) {
	assert := assert.New(t)
	m := newMockManager(newMockStore())

	sess, err := m.Acquire(context.Background(), nil, nil)
	sess.id = "updated"
	assert.NoError(err)

	ctx := context.WithValue(context.Background(), ContextName, sess)
	sessNext, err := m.Acquire(ctx, nil, nil)
	assert.Equal(sess.id, sessNext.id)
	assert.NoError(err)
}
