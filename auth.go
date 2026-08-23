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
	"context"
	"crypto/rand"
	"crypto/rsa"
	_ "embed"
	"encoding/json"
	"myblog/myblog"
	"net/http"
	"net/smtp"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func main() {
	json.Unmarshal(myblog.ConfigBuffer, &myblog.Config)
	myblog.InitRDB()
	myblog.InitPDB()
	// userid, email, username, password, refreshToken, login, timestamp
	myblog.Postgres_client.Query(context.Background(),
		`CREATE TABLE IF NOT EXIST users(uuid UUID PRIMARY KEY, email VARCHAR(255),
		 username VARCHAR(50), password BYTEA, refreshToken VARCHAR(64), login BOOLEAN, timestamp BIGINT)`,
	)
	router := mux.NewRouter()

	myblog.Auth = smtp.PlainAuth("", myblog.Config["user"], myblog.Config["password"], "smtp.gmail.com")
	router.HandleFunc("/login", myblog.Login)
	router.HandleFunc("/register", myblog.Register)
	rrouter := handlers.LoggingHandler(os.Stdout, router)

	key, _ := rsa.GenerateKey(rand.Reader, 2048)

	myblog.PrivateKey = key
	myblog.PublicKey = &key.PublicKey

	server := &http.Server{
		Handler:      rrouter,
		Addr:         "127.0.0.1:1234",
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  30 * time.Second,
	}

	server.ListenAndServe()

}
