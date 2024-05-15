<a href="https://zerodha.tech"><img src="https://zerodha.tech/static/images/github-badge.svg" align="right" /></a>

# simplesessions
simplesessions is a "pure" Go session library that is completely agnostic of HTTP libraries and frameworks, backend stores, and even cookie jars.

## Why?
Most session libraries are highly opinionated and hard-wired to work with `net/http` handlers (or other 3rd party libraries like `fasthttp`) and take liberties on how session data should be encoded and stored. simplesessions takes a pragmatic approach, where everything from HTTP request and cookie handling to data encoding and session storage are plugged in as simple callback functions. Moreover, most session libraries treat data as `strings` losing type information. simplessions provides a way to maintain primitive types such as `int`, `string` etc.

## Features
1. Framework/network library agnostic.
2. Simple API and with support for primitive data types. Complex types can be stored using own encoding/decoding.
3. Pre-built redis/postgres/in-memory stores that can be separately installed.
4. Multiple session instances with custom handlers and different backend stores.

## Installation
Install `simplesessions` and all [available stores](/stores).

```shell
go get -u github.com/vividvilla/simplesessions

# Install the requrired store: memory|goredis|redis|postgres
go get -u github.com/vividvilla/simplesessions/stores/goredis
```

# Stores
Sessions can be stored to any backend by implementing the [store](/store.go) interface. The following stores are bundled.

* [in-memory](/stores/memory)
* [redis](/stores/redis)
* [secure cookie](/stores/securecookie)

# Usage
Check the [examples](/examples) directory for complete examples.

## Connecting a store
Stores can be registered to a session instance by using `Use` method.

```go
sess := simplesessions.New(simplesessions.Options{})
sess.UseStore(memory.New())
```

## Connecting an HTTP handler
Any HTTP library can be connected to simplesessions by registering the `RegisterGetCookie()` and `RegisterSetCookie()` callbacks. The below example shows a simple `net/http` usecase. Another example showing `fasthttp` can be found [here](/examples).

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
	// Read interface sent to callback registered with `RegisterGetCookie`
	// and write interface is sent to callback registered with `RegisterWriteCookie`
	// Optionally `context` can be sent which is usually request context where acquire
	// session will get previously loaded session. This is useful if you have multiple
	// middlewares accessing sessions. New sessions will be created in first middleware which
	// does `Acquire` and will be reused in other places.
	sess, err := sessMan.Acquire(r, w, nil)

	// Use 'Set` and `Commit` to set a field for session.
	// 'Set` ideally doesn't persist the value to store unless method `Commit` is called.
	// But note that its up to the store you are using to decide to
	// persist data only on `commit` or persist on `Set` itself.
	// Stores like redis, db etc should persist on `Commit` while in-memory does on `Set`.
	// No matter what store you use its better to explicitly
	// call `Commit` method when you set all the values.
	err = sess.Set("somekey", "somevalue")
	err = sess.Set("someotherkey", 10)
	err = sess.Commit()

	// Use `Get` method to get a field from current session. The result will be an interface
	// so you can use helper methods like
	// `String', `Int`, `Int64`, `UInt64`, `Float64`, `Bytes`, `Bool`.
	val, err := sess.String(sess.Get("somekey"))

	// Use `GetAll` to get map of all fields from session.
	// The result is map of string and interface you can use helper methods to type cast it.
	val, err := sess.GetAll()

	// Use `GetMulti` to get values for given fields from session.
	// The result is map of string and interface you can use helper methods to type cast it.
	// If key is not there then store should ideally send `nil` value for given key.
	val, err := sess.GetMulti("somekey", "someotherkey")

	// Use `Delete` to delete a field from session.
	err := sess.Delete("somekey")

	// Use `Clear` to clear session from store.
	err := sess.Clear()

	fmt.Fprintf(w, "success")
}

func main() {
	// Create a session manager with custom options like cookie name,
	// cookie domain, is secure cookie etc. Check `Options` struct for more options.
	sessMan := simplesessions.New(simplesessions.Options{})
	// Create a new store instance and attach to session manager
	sessMan.UseStore(memory.New())
	// Register callbacks for read and write cookie
	// Get cookie callback should get cookie based on cookie name and
	// sent back in net/http cookie format.
	sessMan.RegisterGetCookie(getCookie)
	// Set cookie callback should set cookie it received for received cookie name.
	sessMan.RegisterSetCookie(setCookie)

	http.HandleFunc("/set", handler)
}
```
