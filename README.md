# Simple Sessions
Simple and framework agnostic Go sessions library.

# Why Simplesessions?
Most of the sessions library are highly opinionated and works with specific framework or network library only, simplesessions provides
callback for reading and writing cookie so it can be used with any framework or network library like net/http, fasthttp etc.
Also sessions library handles the encoding and decoding implicitly which isn't idle for all use cases, here upto user to
use encoding or just store only primitive data types like `int`, `string` etc.

# Features
1. Framework/network library agnostic.
2. Simple api and supports only primitive data types. Complex types can be stored using own encoding/decoding.
3. Supports redis and in-memory store (More are getting added).

# Installation
Install `simplesessions` and all [available stores](/stores).
```
go get github.com/zerodhatech/simplesessions/...
```

# Available backends

[in-memory](/stores/memory)
[redis](/stores/redis)
secure cookie - in progress

# Using with net/http

Register callbacks for getting and setting cookies like below. Check [example](/examples/nethttp-redis/main.go) for full example.

```go
func getCookie(name string, r interface{}) (*http.Cookie, error) {
	rd := r.(*http.Request)
	cookie, err := rd.Cookie(name)
	if err != nil {
		return nil, err
	}

	return cookie, nil
}

func setCookie(cookie *http.Cookie, w interface{}) error {
	wr := w.(http.ResponseWriter)
	http.SetCookie(wr, cookie)
	return nil
}

func main() {
	...
	sessionManager.RegisterGetCookie(getCookie)
	sessionManager.RegisterSetCookie(setCookie)
}
```

# Using with fasthttp

Register callbacks for getting and setting cookies like below. Check [example](/examples/fasthttp-redis/main.go) for full example.

```go
func getCookie(name string, r interface{}) (*http.Cookie, error) {
	ctx := r.(*fasthttp.RequestCtx)
	cBytes := ctx.Request.Header.Cookie(name)
	// If cookie if empty then send no cookie error
	if len(cBytes) == 0 {
		return nil, http.ErrNoCookie
	}

	// Create fast http cookie and parse it from cookie bytes
	var cookie fasthttp.Cookie
	if err := cookie.ParseBytes(cBytes); err != nil {
		return nil, err
	}

	// Convert fasthttp cookie to net http cookie since
	// simple sessions support cookies in net http format
	return &http.Cookie{
		Name:     name,
		Value:    string(cookie.Value()),
		Path:     string(cookie.Path()),
		Domain:   string(cookie.Domain()),
		Expires:  cookie.Expire(),
		Secure:   cookie.Secure(),
		HttpOnly: cookie.HTTPOnly(),
	}, nil
}

func setCookie(cookie *http.Cookie, w interface{}) error {
	ctx := w.(*fasthttp.RequestCtx)

	// Acquire cookie
	fck := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(fck)
	fck.SetKey(cookie.Name)
	fck.SetValue(cookie.Value)
	fck.SetPath(cookie.Path)
	fck.SetDomain(cookie.Domain)
	fck.SetExpire(cookie.Expires)
	fck.SetSecure(cookie.Secure)
	fck.SetHTTPOnly(cookie.HttpOnly)

	ctx.Response.Header.SetCookie(fck)
	return nil
}

func main() {
	...
	sessionManager.RegisterGetCookie(getCookie)
	sessionManager.RegisterSetCookie(setCookie)
}
```
