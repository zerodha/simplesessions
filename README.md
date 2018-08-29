# simplesessions
simplesessions is a "pure" Go session library that is completely agnostic of HTTP libraries and frameworks, backend stores, and even cookie jars.

## Why?
Most session libraries highly opinionated and are hard-wired to work with `net/http` handlers (or other 3rd party libraries like `fasthttp`) and take liberties on how session data should be encoded and stored. simplesessions takes a pragmatic approach, where everything from HTTP request and cookie handling to data encoding and session storage are plugged in as simple callback functions. Moreover, most session libraries treat data as `strings` losing type information. simplessions provides a way to maintain primitive types such as `int`, `string` etc.

## Features
1. Framework/network library agnostic.
2. Simple API and with support for primitive data types. Complex types can be stored using own encoding/decoding.
3. Bundled Redis and in-memory stores.
4. Multiple session instances with custom handlers and different backend stores.

## Installation
Install `simplesessions` and all [available stores](/stores).

```
go get github.com/zerodhatech/simplesessions/...
```

# Stores
Sessions can be stored to any backend by implementing the [store](/store.go) interface. The following stores are bundled.

* [in-memory](/stores/memory)
* [redis](/stores/redis)
* Secure cookie - in progress

# Usage
Check the [examples](/examples) directory for complete examples.

## Connecting a store
Stores can be registered to a session instance by using `Use` method.

```go
sess := simplesessions.New(simplesessions.Options{})
sess.UseStore(memorystore.New())
```

## Connecting an HTTP handler
Any HTTP library can be connected to simplesessions by registering the `RegisterGetCookie()` and `RegisterSetCookie()` callbacks. The below example shows a simple `net/http` usecase. Another example showing `fasthttp` can be found [here](/examples).

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
	sess := simplesessions.New(simplesessions.Options{})
	sess.UseStore(memorystore.New())

	sess.RegisterGetCookie(getCookie)
	sess.RegisterSetCookie(setCookie)
}
```

## License
Copyright (c) Zerodha Technology Pvt. Ltd. All rights reserved. Licensed under the MIT License.
