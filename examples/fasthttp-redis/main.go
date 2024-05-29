package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/valyala/fasthttp"
	redisstore "github.com/vividvilla/simplesessions/stores/redis/v2"
	"github.com/vividvilla/simplesessions/v3"
)

var (
	sessionManager *simplesessions.Manager

	testKey   = "abc123"
	testValue = 123456
)

func setHandler(ctx *fasthttp.RequestCtx) {
	sess, err := sessionManager.Acquire(ctx, ctx, nil)
	if err != nil {
		ctx.Error(err.Error(), 500)
		return
	}

	err = sess.Set(testKey, testValue)
	if err != nil {
		ctx.Error(err.Error(), 500)
		return
	}

	if err = sess.Commit(); err != nil {
		ctx.Error(err.Error(), 500)
		return
	}

	fmt.Fprintf(ctx, "success")
}

func getHandler(ctx *fasthttp.RequestCtx) {
	sess, err := sessionManager.Acquire(ctx, ctx, nil)
	if err != nil {
		ctx.Error(err.Error(), 500)
		return
	}

	val, err := sess.Int(sess.Get(testKey))
	if err != nil {
		ctx.Error(err.Error(), 500)
		return
	}

	fmt.Fprintf(ctx, "success: %v", val == testValue)
}

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

func getRedisPool(address string, password string, maxActive int, maxIdle int, timeout time.Duration) *redis.Pool {
	return &redis.Pool{
		Wait:      true,
		MaxActive: maxActive,
		MaxIdle:   maxIdle,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial(
				"tcp",
				address,
				redis.DialPassword(password),
				redis.DialConnectTimeout(timeout),
				redis.DialReadTimeout(timeout),
				redis.DialWriteTimeout(timeout),
			)

			return c, err
		},
	}
}

func main() {
	rPool := getRedisPool("localhost:6379", "", 10, 10, 1000*time.Millisecond)

	sessionManager = simplesessions.New(simplesessions.Options{})
	store := redisstore.New(rPool)
	sessionManager.UseStore(store)
	sessionManager.RegisterGetCookie(getCookie)
	sessionManager.RegisterSetCookie(setCookie)

	m := func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/get":
			getHandler(ctx)
		case "/set":
			setHandler(ctx)
		default:
			ctx.Error("not found", fasthttp.StatusNotFound)
		}
	}

	fasthttp.ListenAndServe(":1111", m)
}
