package main

import (
	"fmt"
	"net/http"

	"github.com/valyala/fasthttp"
	"github.com/zerodha/simplesessions/stores/memory/v3"
	"github.com/zerodha/simplesessions/v3"
)

var (
	sessMgr *simplesessions.Manager

	testKey   = "abc123"
	testValue = 123456
)

func setHandler(ctx *fasthttp.RequestCtx) {
	sess, err := sessMgr.Acquire(nil, ctx, ctx)
	// Create new session if it doesn't exist.
	if err == simplesessions.ErrInvalidSession {
		sess, err = sessMgr.NewSession(ctx, ctx)
	}

	if err != nil {
		ctx.Error(err.Error(), 500)
		return
	}

	err = sess.Set(testKey, testValue)
	if err != nil {
		ctx.Error(err.Error(), 500)
		return
	}

	fmt.Fprintf(ctx, "success")
}

func getHandler(ctx *fasthttp.RequestCtx) {
	sess, err := sessMgr.Acquire(nil, ctx, ctx)
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

func main() {
	sessMgr = simplesessions.New(simplesessions.Options{})
	sessMgr.UseStore(memory.New())
	sessMgr.SetCookieHooks(getCookie, setCookie)

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
