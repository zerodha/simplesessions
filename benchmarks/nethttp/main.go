package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/alexedwards/scs"
	scsredisstore "github.com/alexedwards/scs/stores/redisstore"
	gosessions "github.com/kataras/go-sessions"
	gsredisstore "github.com/kataras/go-sessions/sessiondb/redis"
	gsredisservice "github.com/kataras/go-sessions/sessiondb/redis/service"
	simplesessions "github.com/zerodhatech/simplesessions"

	gredis "github.com/garyburd/redigo/redis"
	"github.com/gomodule/redigo/redis"
	"github.com/zerodhatech/simplesessions/stores/redis"
)

var (
	scsManager *scs.Manager
	gsManager  *gosessions.Sessions
	ssManager  *simplesessions.Manager

	redisPool  *redis.Pool
	gRedisPool *gredis.Pool

	testKey   = "abc123"
	testValue = 123456
)

func ssSetHandler(w http.ResponseWriter, r *http.Request) {
	sess, err := ssManager.Acquire(r, w, nil)
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

func ssGetHandler(w http.ResponseWriter, r *http.Request) {
	sess, err := ssManager.Acquire(r, w, nil)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	val, err := sess.Int(sess.Get(testKey))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// error
	fmt.Fprintf(w, "success: %v", val == testValue)
}

func ssGetCookie(name string, r interface{}) (*http.Cookie, error) {
	rd := r.(*http.Request)
	cookie, err := rd.Cookie(name)
	if err != nil {
		// log.Printf("couldn't read cookie - %v", err)
		return nil, err
	}

	return cookie, nil
}

func ssSetCookie(cookie *http.Cookie, w interface{}) error {
	wr := w.(http.ResponseWriter)
	http.SetCookie(wr, cookie)
	return nil
}

func scsGetHandler(w http.ResponseWriter, r *http.Request) {
	sess := scsManager.Load(r)
	val, err := sess.GetInt(testKey)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	fmt.Fprintf(w, "success: %v", val == testValue)
}

func scsSetHandler(w http.ResponseWriter, r *http.Request) {
	sess := scsManager.Load(r)
	err := sess.PutInt(w, testKey, 123456)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	fmt.Fprintf(w, "success")
}

func gsGetHandler(w http.ResponseWriter, r *http.Request) {
	sess := gsManager.Start(w, r)
	rVal := sess.Get(testKey)
	fmt.Fprintf(w, "success: %v", rVal)
}

func gsSetHandler(w http.ResponseWriter, r *http.Request) {
	sess := gsManager.Start(w, r)
	sess.Set(testKey, 123456)
	fmt.Fprintf(w, "success")
}

func redisGetHandler(w http.ResponseWriter, r *http.Request) {
	conn := redisPool.Get()
	defer conn.Close()

	val, err := redis.Int(conn.Do("GET", testKey))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	fmt.Fprintf(w, "success: %v", val == testValue)
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

func getGRedisPool(address string, password string, maxActive int, maxIdle int, timeout time.Duration) *gredis.Pool {
	return &gredis.Pool{
		Wait:      true,
		MaxActive: maxActive,
		MaxIdle:   maxIdle,
		Dial: func() (gredis.Conn, error) {
			c, err := gredis.Dial(
				"tcp",
				address,
				gredis.DialPassword(password),
				gredis.DialConnectTimeout(timeout),
				gredis.DialReadTimeout(timeout),
				gredis.DialWriteTimeout(timeout),
			)

			return c, err
		},
	}
}

func main() {
	redisPool = getRedisPool("localhost:6379", "", 50, 50, 1000*time.Millisecond)
	gRedisPool = getGRedisPool("localhost:6379", "", 50, 50, 1000*time.Millisecond)

	// Simple sessions
	ssManager = simplesessions.New(simplesessions.Options{
		CookieName: "sscookie",
	})
	ssManager.UseStore(redisstore.New(redisPool))
	ssManager.RegisterGetCookie(ssGetCookie)
	ssManager.RegisterSetCookie(ssSetCookie)

	// scs sessions
	scsManager = scs.NewManager(scsredisstore.New(gRedisPool))
	scsManager.Name("scscookie")

	// go sessions
	gsManager = gosessions.New(gosessions.Config{
		Cookie: "gscookie",
	})
	gsStore := gsredisstore.New(gsredisservice.Config{Network: gsredisservice.DefaultRedisNetwork,
		Addr:        "localhost:6379",
		MaxIdle:     50,
		MaxActive:   50,
		IdleTimeout: 1000,
	})
	gsManager.UseDatabase(gsStore)

	// Simple sessions handler
	http.HandleFunc("/ss/set", ssSetHandler)
	http.HandleFunc("/ss/get", ssGetHandler)

	http.HandleFunc("/scs/set", scsSetHandler)
	http.HandleFunc("/scs/get", scsGetHandler)

	http.HandleFunc("/gs/set", gsSetHandler)
	http.HandleFunc("/gs/get", gsGetHandler)

	http.HandleFunc("/redis/get", redisGetHandler)

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello world")
	})

	log.Fatal(http.ListenAndServe(":1111", nil))
}
