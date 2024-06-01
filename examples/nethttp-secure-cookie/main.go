package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/zerodha/simplesessions/stores/securecookie/v3"
	"github.com/zerodha/simplesessions/v3"
)

var (
	sessMgr *simplesessions.Manager

	store = securecookie.New(
		[]byte("0dIHy6S2uBuKaNnTUszB218L898ikGYA"),
		[]byte("0dIHy6S2uBuKaNnTUszB218L898ikGYA"),
	)

	testKey   = "abc123"
	testValue = 123456
)

func setHandler(w http.ResponseWriter, r *http.Request) {
	sess, err := sessMgr.Acquire(nil, r, w)
	// Create new session if it doesn't exist.
	if err == simplesessions.ErrInvalidSession {
		sess, err = sessMgr.NewSession(r, w)

		// IMPORTANT: any Set/SetMulti/Delete/Clear/Destroy and NewSession()
		// should flush the values using `store.Flush()` otherwise cookie won't be updated.
		if err == nil {
			ck, err := store.Flush(sess.ID())
			if err == nil {
				err = sess.WriteCookie(ck)
			}
		}
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

	// For securecookies, ID() of the session is the encoded cookie
	// data itself.
	v, err := store.Flush(sess.ID())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Write the cookie.
	if err := sess.WriteCookie(v); err != nil {
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

func main() {
	sessMgr = simplesessions.New(simplesessions.Options{})
	sessMgr.UseStore(store)
	sessMgr.SetCookieHooks(getCookie, setCookie)

	http.HandleFunc("/set", setHandler)
	http.HandleFunc("/get", getHandler)
	log.Fatal(http.ListenAndServe(":1111", nil))
}
