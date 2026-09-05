package main

import (
	"crypto/x509"
	_ "embed"
	"encoding/json"
	"encoding/pem"
	"log"
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
		"postid",
		"body",
		"page",
		"csrf-token",
	}), handlers.AllowedMethods([]string{"GET", "POST", "OPTIONS"}))(rrouter)

	privateK, e := os.ReadFile(os.Getenv("HOME") + "/private_key")
	if e != nil {
		log.Fatal(e)
	}
	pemjerked, _ := pem.Decode(privateK)
	prk, e := x509.ParsePKCS1PrivateKey(pemjerked.Bytes)
	if e != nil {
		log.Fatal(e)
	}

	publicK, e := os.ReadFile(os.Getenv("HOME") + "/public_key")
	if e != nil {
		log.Fatal(e)
	}
	pemjerked2, _ := pem.Decode(publicK)

	puk, e := x509.ParsePKCS1PublicKey(pemjerked2.Bytes)
	if e != nil {
		log.Fatal(e)
	}

	myblog.PrivateKey = prk
	myblog.PublicKey = puk

	server := &http.Server{
		Handler:      rrrouter,
		Addr:         "127.0.0.1:1236",
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  30 * time.Second,
	}

	server.ListenAndServe()

}
