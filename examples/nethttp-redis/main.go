package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/vividvilla/simplesessions"
	redisstore "github.com/vividvilla/simplesessions/stores/redis"
)

var (
	sessionManager *simplesessions.Manager

	testKey   = "abc123"
	testValue = 123456
)

func setHandler(w http.ResponseWriter, r *http.Request) {
	sess, err := sessionManager.Acquire(r, w, nil)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	err = sess.Set(testKey, testValue)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if err = sess.Commit(); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	fmt.Fprintf(w, "success")
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	sess, err := sessionManager.Acquire(r, w, nil)
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

	http.HandleFunc("/set", setHandler)
	http.HandleFunc("/get", getHandler)
	log.Fatal(http.ListenAndServe(":1111", nil))
}
