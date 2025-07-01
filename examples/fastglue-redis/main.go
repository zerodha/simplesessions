package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	redisstore "github.com/zerodha/simplesessions/stores/redis/v3"
	"github.com/zerodha/simplesessions/v3"
)

const (
	GeneralError = "GeneralException"
)

var (
	sessMgr   *simplesessions.Manager
	testKey   = "question"
	testValue = 42
)

func initRedisGo(address, password string) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: password,
		DB:       0, // use default DB
	})
	return rdb
}

// returns a fasthttp server used for serving fastglue routes
func initServer(name string, timeout int) *fasthttp.Server {
	return &fasthttp.Server{
		Name:         name,
		ReadTimeout:  time.Second * time.Duration(timeout),
		WriteTimeout: time.Second * time.Duration(timeout),
	}

}

func setHandler(r *fastglue.Request) error {
	sess, err := sessMgr.Acquire(nil, r.RequestCtx, r.RequestCtx)
	// Create new session if it doesn't exist.
	if err == simplesessions.ErrInvalidSession {
		sess, err = sessMgr.NewSession(r.RequestCtx, r.RequestCtx)
	}
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, GeneralError)
	}

	err = sess.Set(testKey, testValue)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, GeneralError)
	}

	return r.SendEnvelope("success")
}

func getHandler(r *fastglue.Request) error {
	sess, err := sessMgr.Acquire(nil, r.RequestCtx, r.RequestCtx)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, GeneralError)
	}

	val, err := sess.Int(sess.Get(testKey))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, GeneralError)
	}

	return r.SendEnvelope(val == testValue)
}

func destroyHandler(r *fastglue.Request) error {
	sess, err := sessMgr.Acquire(nil, r.RequestCtx, r.RequestCtx)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, GeneralError)
	}
	err = sess.Destroy()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, GeneralError)
	}

	return r.SendEnvelope(true)
}

// getCookie() gets the fasthttp cookie and passes its values to a http.Cookie
func getCookie(name string, r interface{}) (*http.Cookie, error) {
	ctx := r.(*fasthttp.RequestCtx)
	cookieBody := ctx.Request.Header.Cookie(name)
	// If cookie if empty then send no cookie error
	if len(cookieBody) == 0 {
		return nil, http.ErrNoCookie
	}

	// Create fast http cookie and parse it from cookie bytes
	var cookie fasthttp.Cookie
	if err := cookie.ParseBytes(cookieBody); err != nil {
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

// setCookie() sets a fasthttp.Cookie by passing http.Cookie values
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

	rc := initRedisGo("localhost:6379", "")
	ctx := context.Background()
	store := redisstore.New(ctx, rc)

	sessMgr = simplesessions.New(simplesessions.Options{})
	sessMgr.UseStore(store)
	sessMgr.SetCookieHooks(getCookie, setCookie)

	g := fastglue.New()
	g.GET("/get", getHandler)
	g.GET("/set", setHandler)
	g.GET("/destroy", destroyHandler)

	// 5s read/write timeout
	server := initServer("go-redis", 5)
	if err := g.ListenAndServe(":1111", "", server); err != nil {
		log.Fatal(err)
	}
}
