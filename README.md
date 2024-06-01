<a href="https://zerodha.tech"><img src="https://zerodha.tech/static/images/github-badge.svg" align="right" /></a>

# simplesessions
simplesessions is a "pure" Go session library that is completely agnostic of HTTP libraries and frameworks, backend stores, and even cookie jars.

## Why?
Most session libraries are highly opinionated and hard-wired to work with `net/http` handlers (or other 3rd party libraries like `fasthttp`) and take liberties on how session data should be encoded and stored. simplesessions takes a pragmatic approach, where everything from HTTP request and cookie handling to data encoding and session storage are plugged in as simple callback functions. Moreover, most session libraries treat data as `strings` losing type information. simplessions provides a way to maintain primitive types such as `int`, `string` etc.

## Features
1. Framework/network library agnostic.
2. Simple API and with support for primitive data types. Complex types can be stored using own encoding/decoding.
3. Pre-built redis/postgres/in-memory/securecookie stores that can be separately installed.
4. Multiple session instances with custom handlers and different backend stores.

## Installation
Install `simplesessions` and all [available stores](/stores).

```shell
go get -u github.com/zerodha/simplesessions/v3

# Install the requrired store: memory|redis|postgres|securecookie
go get -u github.com/zerodha/simplesessions/v3/stores/redis
go get -u github.com/zerodha/simplesessions/v3/stores/postgres
```

# Stores
Sessions can be stored to any backend by implementing the [store](/store.go) interface. The following stores are bundled.

* [redis](/stores/redis)
* [postgres](/stores/postgres)
* [in-memory](/stores/memory)
* [secure cookie](/stores/securecookie)

# Usage
Check the [examples](/examples) directory for complete examples.

## Connecting a store
Stores can be registered to a session instance by using `Use` method. Check individual [Stores](#stores) docs for more details.

```go
sess := simplesessions.New(simplesessions.Options{})
sess.UseStore(store.New())
```

## Connecting an HTTP handler
Any HTTP library can be connected to simplesessions by registering the get and set cookie hooks using `SetCookieHooks()`. The below example shows a simple `net/http` usecase. Another example showing `fasthttp` can be found [here](/examples).

```go
var sessMan *simplesessions.Manager

func getCookie(name string, r interface{}) (*http.Cookie, error) {
	// Get read interface registered using `Acquire` method in handlers.
	rd := r.(*http.Request)

	// Send cookie for received cookie name from request.
	// Note that other networking libs and frameworks should
	// also send back cookie in net/http cookie format.
	// If cookie is not found for given cookie name then
	// `http.ErrNoCookie` should be returned.
	// Cookie name is what you set while creating session manager
	// with custom options (`Options.CookieName`). Defaults to `session`.
	cookie, err := rd.Cookie(name)
	if err != nil {
		return nil, err
	}

	return cookie, nil
}

func setCookie(cookie *http.Cookie, w interface{}) error {
	// Get write interface registered using `Acquire` method in handlers.
	wr := w.(http.ResponseWriter)

	// net/http cookie is returned which can be
	// used to set cookie for current request.
	// Note that other network libraries or
	// framework will also receive cookie as
	// net/http cookie and it has to set cookie accordingly.
	http.SetCookie(wr, cookie)
	return nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	// Use method `Acquire` to acquire a session before you access the session.
	// Acquire takes read, write interface and context respectively.
	// Read interface sent to callback registered with get cookie hook
	// and write interface is sent to callback registered with write cookie hook
	// set using `SetCookieHooks()` method.
	//
	// Optionally `context` can be sent which is usually request context where acquire
	// session will get previously loaded session. This is useful if you have multiple
	// middlewares accessing sessions. New sessions will be created in first middleware which
	// does `Acquire` and will be reused in other places.
	//
	// If `Options.EnableAutoCreate` is set to True then if session doesn't exist it will
	// be immediately created and returned. Bydefault its set to False so if session doesn't
	// exist then `ErrInvalidSession` error is returned.
	sess, err := sessMan.Acquire(nil, r, w)

	// If session doesn't exist then create new session.
	// In a traditional login flow you can create a new session once user completes the login flow.
	if err == simplesessions.ErrInvalidSession {
		sess, err = sessMan.NewSession(r, w)
	}

	// Use 'Set` or `SetMulti` to set a field for session.
	err = sess.Set("somekey", "somevalue")
	err = sess.SetMulti(map[string]interface{}{
		"k1": "v1",
		"k2": "v2",
	})

	// Use `Get` method to get a field from current session. The result will be an interface
	// so you can use helper methods like
	// `String', `Int`, `Int64`, `UInt64`, `Float64`, `Bytes`, `Bool`.
	val, err := sess.String(sess.Get("somekey"))
	fmt.Println("val=", val)

	// Use `GetAll` to get map of all fields from session.
	// The result is map of string and interface you can use helper methods to type cast it.
	all, err := sess.GetAll()
	fmt.Println("all=", all)

	// Use `GetMulti` to get values for given fields from session.
	// The result is map of string and interface you can use helper methods to type cast it.
	// If key is not there then store should ideally send `nil` value for given key.
	vals, err := sess.GetMulti("somekey", "someotherkey")
	fmt.Println("vals=", vals)

	// Use `Delete` to delete a field from session.
	err = sess.Delete("somekey")

	// Use `Clear` to empty the session but to keep the session alive.
	err = sess.Clear()

	// Use `Destroy` to clear session from store and cookie.
	err = sess.Destroy()

	fmt.Fprintf(w, "success")
}

func main() {
	// Create a session manager with custom options like cookie name,
	// cookie domain, is secure cookie etc. Check `Options` struct for more options.
	sessMan := simplesessions.New(simplesessions.Options{
		// If set to true then `Acquire()` method will create new session instead of throwing
		// `ErrInvalidSession` when the session doesn't exist. By default its set to false.
		EnableAutoCreate: false,
		Cookie: simplesessions.CookieOptions{
			// Name sets http cookie name. This is also sent as cookie name in `GetCookie` callback.
			Name: "session",
			// Domain sets hostname for the cookie. Domain specifies allowed hosts to receive the cookie.
			Domain: "example.com",
			// Path sets path for the cookie. Path indicates a URL path that must exist in the requested URL in order to send the cookie header.
			Path: "/",
			// IsSecure marks the cookie as secure cookie (only sent in HTTPS).
			IsSecure: true,
			// IsHTTPOnly marks the cookie as http only cookie. JS won't be able to access the cookie so prevents XSS attacks.
			IsHTTPOnly: true,
			// SameSite sets allows you to declare if your cookie should be restricted to a first-party or same-site context.
			SameSite: http.SameSiteDefaultMode,
			// Expires sets absolute expiration date and time for the cookie.
			// If both Expires and MaxAge are sent then MaxAge takes precedence over Expires.
			// Cookies without a Max-age or Expires attribute – are deleted when the current session ends
			// and some browsers use session restoring when restarting. This can cause session cookies to last indefinitely.
			Expires: time.Now().Add(time.Hour * 24),
			// Sets the cookie's expiration in seconds from the current time, internally its rounder off to nearest seconds.
			// If both Expires and MaxAge are sent then MaxAge takes precedence over Expires.
			// Cookies without a Max-age or Expires attribute – are deleted when the current session ends
			// and some browsers use session restoring when restarting. This can cause session cookies to last indefinitely.
			MaxAge: time.Hour * 24,
		},
	})

	// Create a new store instance and attach to session manager
	sessMan.UseStore(memory.New())
	// Register callbacks for read and write cookie.
	// Get cookie callback should get cookie based on cookie name and
	// sent back in net/http cookie format.
	// Set cookie callback should set cookie it received for received cookie name.
	sessMan.SetCookieHooks(getCookie, setCookie)

	// Initialize the handler.
	http.HandleFunc("/", handler)
}
```
