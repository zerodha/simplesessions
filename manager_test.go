package simplesessions

import (
	"context"
	"fmt"
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
	m.SetCookieHooks(mockGetCookieCb, mockSetCookieCb)
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
	// Default cookie path is set to root
	assert.Equal(t, "/", m.opts.Cookie.Path)
	// Default cookie name is set
	assert.Equal(t, defaultCookieName, m.opts.Cookie.Name)
}

func TestManagerNewManagerWithOptions(t *testing.T) {
	opts := Options{
		EnableAutoCreate: true,
		SessionIDLength:  16,
		Cookie: CookieOptions{
			Name:       "testcookiename",
			Domain:     "somedomain",
			Path:       "/abc/123",
			IsSecure:   true,
			IsHTTPOnly: true,
			SameSite:   http.SameSiteLaxMode,
			MaxAge:     time.Hour * 1,
			Expires:    time.Now(),
		},
	}

	m := New(opts)
	assert.Equal(t, opts.EnableAutoCreate, m.opts.EnableAutoCreate)
	assert.Equal(t, opts.SessionIDLength, m.opts.SessionIDLength)
	assert.Equal(t, opts.Cookie.Name, m.opts.Cookie.Name)
	assert.Equal(t, opts.Cookie.Domain, m.opts.Cookie.Domain)
	assert.Equal(t, opts.Cookie.Path, m.opts.Cookie.Path)
	assert.Equal(t, opts.Cookie.IsSecure, m.opts.Cookie.IsSecure)
	assert.Equal(t, opts.Cookie.SameSite, m.opts.Cookie.SameSite)
	assert.Equal(t, opts.Cookie.IsHTTPOnly, m.opts.Cookie.IsHTTPOnly)
	assert.Equal(t, opts.Cookie.MaxAge, m.opts.Cookie.MaxAge)
	assert.Equal(t, opts.Cookie.Expires, m.opts.Cookie.Expires)

	// Default opts.
	m = New(Options{})
	assert.NotNil(t, m.generateID)
	assert.NotNil(t, m.validateID)

	assert.Equal(t, false, m.opts.EnableAutoCreate)
	assert.Equal(t, defaultSessIDLength, m.opts.SessionIDLength)
	assert.Equal(t, defaultCookieName, m.opts.Cookie.Name)
	assert.Equal(t, defaultCookiePath, m.opts.Cookie.Path)
}

func TestManagerUseStore(t *testing.T) {
	s := newMockStore()
	m := newMockManager(s)
	assert.Equal(t, s, m.store)
}

func TestManagerSetCookieHooks(t *testing.T) {
	ck := &http.Cookie{
		Name: "testcookie",
	}

	get := func(string, interface{}) (*http.Cookie, error) {
		return ck, http.ErrNoCookie
	}
	set := func(*http.Cookie, interface{}) error {
		return http.ErrNoCookie
	}

	m := New(Options{})
	m.SetCookieHooks(get, set)

	expRes, expErr := get("", nil)
	gotRes, gotErr := m.getCookieHook("", nil)
	assert.Equal(t, expRes, gotRes)
	assert.Equal(t, expErr, gotErr)

	expErr = set(ck, nil)
	gotErr = m.setCookieHook(ck, nil)
	assert.Equal(t, expErr, gotErr)
}

func TestManagerAcquireFails(t *testing.T) {
	m := New(Options{})

	// Fail if store is not assigned.
	_, err := m.Acquire(context.Background(), nil, nil)
	assert.Equal(t, "session store not set", err.Error())

	// Fail if getCookie callback is not assigned.
	m.UseStore(&MockStore{})
	_, err = m.Acquire(context.Background(), nil, nil)
	assert.Equal(t, "`GetCookie` hook not set", err.Error())

	// Assign getCookie, returns nil cookie to make sure it
	// fails in create session with invalid session.
	m.SetCookieHooks(func(string, interface{}) (*http.Cookie, error) { return nil, nil }, nil)

	// Fail if setCookie callback is not assigned.
	_, err = m.Acquire(context.Background(), nil, nil)
	assert.Equal(t, "`SetCookie` hook not set", err.Error())

	// Register setCookie callback.
	m.SetCookieHooks(func(string, interface{}) (*http.Cookie, error) { return nil, nil },
		func(*http.Cookie, interface{}) error { return nil })

	// By default EnableAutoCreate is disabled
	// Check if it returns invalid session.
	_, err = m.Acquire(context.Background(), nil, nil)
	assert.ErrorIs(t, err, ErrInvalidSession)
}

func TestManagerAcquireAutocreate(t *testing.T) {
	m := newMockManager(newMockStore())
	// Enable autocreate.
	m.opts.EnableAutoCreate = true
	m.SetCookieHooks(func(string, interface{}) (*http.Cookie, error) { return nil, ErrInvalidSession },
		func(*http.Cookie, interface{}) error { return nil })

	// If cookie doesn't exist then should return a new one without error.
	sess, err := m.Acquire(context.Background(), nil, nil)
	assert.NoError(t, err)
	assert.True(t, m.validateID(sess.id))
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

func TestDefaultGenerateID(t *testing.T) {
	m := New(Options{})
	id, err := m.generateID()
	assert.NoError(t, err)
	assert.Equal(t, defaultSessIDLength, len(id))

	m = New(Options{
		SessionIDLength: 16,
	})
	id, err = m.generateID()
	assert.NoError(t, err)
	assert.Equal(t, 16, len(id))
}

func TestDefaultValidateID(t *testing.T) {
	m := New(Options{})
	id, err := m.generateID()
	assert.NoError(t, err)
	assert.True(t, m.validateID(id))
	assert.False(t, m.validateID("xxxx"))
	assert.False(t, m.validateID("11IHy6S2uBuKaNnTUszB218L898ikGY*"))
}

func TestSetSessionIDHooks(t *testing.T) {
	var (
		m            = New(Options{})
		genErr error = nil
		genID        = "xxx"
		valOut       = true
	)
	gen := func() (string, error) {
		return genID, genErr
	}
	validate := func(string) bool {
		return valOut
	}
	m.SetSessionIDHooks(gen, validate)

	id, err := m.generateID()
	eID, eErr := gen()
	assert.Equal(t, eID, id)
	assert.Equal(t, eErr, err)

	genErr = fmt.Errorf("custom error")
	_, err = m.generateID()
	assert.ErrorIs(t, genErr, err)

	assert.True(t, m.validateID(genID))
	valOut = false
	assert.False(t, m.validateID(genID))
}
