package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	redisstore "github.com/zerodha/simplesessions/stores/redis/v3"
	"github.com/zerodha/simplesessions/v3"
)

var (
	sessMgr *simplesessions.Manager

	testKey   = "abc123"
	testValue = 123456
)

func setHandler(w http.ResponseWriter, r *http.Request) {
	sess, err := sessMgr.Acquire(nil, r, w)

	// Create new session if it doesn't exist.
	if err == simplesessions.ErrInvalidSession {
		sess, err = sessMgr.NewSession(r, w)
	}

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	err = sess.Set(testKey, testValue)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	fmt.Fprintf(w, "success")
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	sess, err := sessMgr.Acquire(nil, r, w)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	val, err := sess.Int(sess.Get(testKey))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	fmt.Fprintf(w, "success: %v", val == testValue)
}

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

func getRedisPool() redis.UniversalClient {
	o := &redis.Options{
		Addr:        "localhost:6379",
		Username:    "",
		Password:    "",
		DialTimeout: time.Second * 3,
		DB:          0,
	}

	var (
		ctx = context.TODO()
		cl  = redis.NewClient(o)
	)
	if err := cl.Ping(ctx).Err(); err != nil {
		log.Fatalf("error initializing redis: %v", err)
	}

	return cl
}

func main() {
	sessMgr = simplesessions.New(simplesessions.Options{})
	store := redisstore.New(context.Background(), getRedisPool())
	sessMgr.UseStore(store)
	sessMgr.SetCookieHooks(getCookie, setCookie)

	http.HandleFunc("/set", setHandler)
	http.HandleFunc("/get", getHandler)
	log.Fatal(http.ListenAndServe(":1111", nil))
}
