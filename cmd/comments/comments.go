package main

/*
*
ok so we need some handlers
these are the apis we need
comments
posts
authentication/authorization
api itself

after building all this states
we separate them each making them individual apps
and tunnel them with a reverse proxy (ngnix)

*/

import (
	"crypto/rand"
	"crypto/rsa"
	_ "embed"
	"encoding/json"
	"myblog/myblog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func SecurityHandlers(router http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		/*
			Telling browser contents information, they should come from these
			   w.Header().Set("Content-Security-Policy",
			       "default-src 'self'; " +
			       "script-src 'self' https://trusted-cdn.com; " +
			       "style-src 'self' 'unsafe-inline'; " +
			       "img-src 'self' data: https:; " +
			       "font-src 'self' https://fonts.gstatic.com; " +
			       "frame-ancestors 'none';")
		*/
		w.Header().Set("x-frame-options", "DENY") // click hijacking preventation
		w.Header().Set("Strict-Transport-Security",
			"max-age=31536000; includeSubDomains; preload") // https only
		w.Header().Set("X-Content-Type-Options", "nosniff") // like i have an image but browser without this will think images are json
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy",
			"camera=(), microphone=(), geolocation=(), payment=()") // allowing nothing
		router.ServeHTTP(w, r)
	})
}

func main() {
	json.Unmarshal(myblog.ConfigBuffer, &myblog.Config)
	myblog.InitRDB()
	myblog.InitPDB()
	// userid, email, username, password, refreshToken, login, timestamp

	router := mux.NewRouter()

	router.HandleFunc("/comment", myblog.Comment)
	router.HandleFunc("/getComments", myblog.GetComments)

	rrouter := handlers.LoggingHandler(os.Stdout, router)
	rrouter = SecurityHandlers(rrouter)
	rrrouter := handlers.CORS(handlers.AllowedOrigins([]string{"http://127.0.0.1:5132"}), handlers.AllowCredentials(), handlers.AllowedHeaders([]string{
		"signature",
		"answer",
		"token",
		"realip",
		"captchaAnswer",
		"content-type",
		"verification",
		"username",
		"password",
		"email",
		"csrf-Token",
	}), handlers.AllowedMethods([]string{"GET", "POST", "OPTIONS"}))(rrouter)

	key, _ := rsa.GenerateKey(rand.Reader, 2048)

	myblog.PrivateKey = key
	myblog.PublicKey = &key.PublicKey

	server := &http.Server{
		Handler:      rrrouter,
		Addr:         "127.0.0.1:1236",
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  30 * time.Second,
	}

	server.ListenAndServe()

}
