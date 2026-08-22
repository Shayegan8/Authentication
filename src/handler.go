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
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	rnd "math/rand/v2"
	"myblog"
	"myblog/src/module"
	"net/http"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	"github.com/wenlng/go-captcha-assets/resources/imagesv2"
	"github.com/wenlng/go-captcha-assets/resources/tiles"
	"github.com/wenlng/go-captcha/v2/slide"
	"golang.org/x/crypto/bcrypt"
)

type CaptchData struct {
	token        string
	timestamp    int64
	x            int
	y            int
	verification int
}

type BucketData struct {
	tokens    []string
	timestamp int64
}

var redis_client *redis.Client
var postgres_client *sql.DB
var captchaTokens map[string]CaptchData
var captchaTokensLock *sync.Mutex
var users map[string]slide.CaptData
var auth smtp.Auth

func Init() slide.Captcha {
	builder := slide.NewBuilder(
		slide.WithEnableGraphVerticalRandom(true),
	)

	imgs, _ := imagesv2.GetImages()

	graphs, _ := tiles.GetTiles()

	var newGraphs = make([]*slide.GraphImage, 0, len(graphs))
	for i := 0; i < len(graphs); i++ {
		graph := graphs[i]
		newGraphs = append(newGraphs, &slide.GraphImage{
			OverlayImage: graph.OverlayImage,
			MaskImage:    graph.MaskImage,
			ShadowImage:  graph.ShadowImage,
		})
	}

	builder.SetResources(
		slide.WithGraphImages(newGraphs),
		slide.WithBackgrounds(imgs),
	)

	return builder.Make()
}

func generateBucket(n int) []string {
	var arr []string = make([]string, n)
	for i := range len(arr) {
		var buffer []byte = make([]byte, 10)
		rand.Read(buffer)
		arr[i] = hex.EncodeToString(buffer)
	}
	return arr
}

func main() {
	json.Unmarshal(myblog.ConfigBuffer, &myblog.Config)
	redis_client = module.InitRDB()
	postgres_client = module.InitPDB()
	router := mux.NewRouter()

	auth = smtp.PlainAuth("", myblog.Config["user"], myblog.Config["password"], "smtp.gmail.com")
	router.HandleFunc("/login", login)
	router.HandleFunc("/register", register)
	rrouter := handlers.LoggingHandler(os.Stdout, router)

	server := &http.Server{
		Handler:      rrouter,
		Addr:         "127.0.0.1:1234",
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  30 * time.Second,
	}

	server.ListenAndServe()

}

func bucketHandlement(w http.ResponseWriter, r *http.Request) {
	dip := r.Header.Get("realip")
	token, err := redis_client.RPop(context.Background(), dip).Result()
	if err == redis.Nil {

		counter, err := redis_client.Incr(context.Background(), "counter"+dip).Result()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}

		if counter == 1 {
			redis_client.Expire(context.Background(), "counter"+dip, 30*time.Second)
		}

		if counter > 10 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Bad request"))
			return
		}

		redis_pipe := redis_client.Pipeline()
		buckdat := generateBucket(rnd.IntN(50) + 20)
		buckdatInterfaces := make([]any, len(buckdat))
		for k, v := range buckdat {
			buckdatInterfaces[k] = v
		}
		redis_pipe.LPush(context.Background(), dip, buckdatInterfaces...)
		redis_pipe.Expire(context.Background(), dip, 5*time.Minute)
		redis_pipe.Exec(context.Background())
		token, err := redis_client.RPop(context.Background(), dip).Result()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(token))
		return
	} else {
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}
		result, err := redis_client.LIndex(context.Background(), dip, -1).Result()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}
		if result == "in-queue" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return

		}
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(token))
		length, err := redis_client.LLen(context.Background(), dip).Result()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}
		if length == 0 {
			redis_client.LPush(context.Background(), dip, "in-queue")
		}
	}
}

func login(w http.ResponseWriter, r *http.Request) {
	payload := r.Header
	switch r.Method {
	case "GET":
		bucketHandlement(w, r)
	case "POST":
		token := payload.Get("token") // generated token from bucket
		if token == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		dip := r.Header.Get("realip")

		removed, err := redis_client.LRem(context.Background(), dip, 1, token).Result()

		if err != nil { // dont care if its server error or client bad request
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		} else if removed == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		captchaD := payload.Get("captchaToken")
		var captchaData map[string]string
		if captchaD != "" {
			err := json.Unmarshal([]byte(captchaD), &captchaData)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("It should be all string"))
				return
			}
		}

		verification := payload.Get("verification")
		password := payload.Get("password")
		email := payload.Get("email") // for login email field can be either username or email
		CaptchaGeneration(verification, email, password, email, dip, w)
		CaptchaToken(captchaD, captchaData, w, dip)

		if email == "" || password == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		var rows *sql.Rows
		var erra error
		if strings.Contains(email, "@") {
			rows, erra = postgres_client.Query("SELECT password FROM users WHERE email=$1", email)
		} else {
			rows, erra = postgres_client.Query("SELECT password FROM users WHERE username=$1", email)
		}
		if erra != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error")) // Invalid user credintals
			return
		}
		if !rows.Next() {
			rows.Close()
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request")) // Invalid user credintals
			return
		} else {
			var hashedPass []byte
			erri := rows.Scan(&hashedPass)
			rows.Close()
			if erri != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Server error")) // Invalid user credintals
				return
			}
			err := bcrypt.CompareHashAndPassword(hashedPass, []byte(password))
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad request"))
				return
			}
		}
		Verify(verification, password, email, dip, w)
		Authing(email, email, password, verification, dip, w, false)
	}
}

func register(w http.ResponseWriter, r *http.Request) {
	payload := r.Header
	switch r.Method {
	case "GET":
		bucketHandlement(w, r)
	case "POST":
		token := payload.Get("token") // generated token from bucket
		if token == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("You have to use token"))
			return
		}

		dip := r.Header.Get("realip")

		removed, err := redis_client.LRem(context.Background(), dip, 1, token).Result()

		if err != nil { // dont care if its server error or client bad request
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		} else if removed == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		captchaD := payload.Get("captchaToken")
		var captchaData map[string]string
		if captchaD != "" {
			err := json.Unmarshal([]byte(captchaD), &captchaData)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("It should be all string"))
				return
			}
		}

		verification := payload.Get("verification")
		username := payload.Get("username")
		password := payload.Get("password")
		email := payload.Get("email")

		CaptchaGeneration(verification, username, password, email, dip, w)
		CaptchaToken(captchaD, captchaData, w, dip)

		if username == "" || email == "" || password == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if strings.ContainsAny(username, "@") {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		rows, erra := postgres_client.Query("SELECT * FROM users WHERE email=$1 OR username=$2", email, username)
		if erra != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error")) // Invalid user credintals
			return
		}

		if rows.Next() { // this means if the user with this details actually exist we reject them
			rows.Close()
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request")) // Invalid user credintals
			return
		}

		Verify(verification, password, email, dip, w)
		Authing(username, email, password, verification, dip, w, true)
	}
}

func CaptchaGeneration(verification string, username string, password string, email string, dip string, w http.ResponseWriter) {
	if verification == "" && username == "" && password == "" && email == "" { // this means user wants captcha
		captcha := Init()
		captData, err := captcha.Generate()
		if err != nil {
			log.Fatalln(err)
		}

		dotData := captData.GetData()
		if dotData == nil {
			log.Fatalln("ERROR FOR CAPTCHA")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Problem with server"))
			return
		}

		masterImage, _ := captData.GetMasterImage().ToBase64()
		tileImage, _ := captData.GetTileImage().ToBase64()
		jsoned := map[string]string{
			"masterImage": masterImage,
			"tileImage":   tileImage,
		}
		jsonedBuffer, _ := json.Marshal(jsoned)
		// we should have store the answer in some storage,
		captchaTokensLock.Lock()
		captchaTokens[dip] = CaptchData{x: captData.GetData().X, y: captData.GetData().Y, timestamp: time.Now().Unix()}
		captchaTokensLock.Unlock()
		w.WriteHeader(http.StatusAccepted)
		w.Write(jsonedBuffer)
		return
	}

}

func CaptchaToken(captchaD string, captchaData map[string]string, w http.ResponseWriter, dip string) {
	if captchaD != "" {
		x, err := strconv.Atoi(captchaData["x"])
		if err != nil { // this blocks are for testing and might be removed or above code might be changed
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("It should be string"))
			return
		}
		y, err1 := strconv.Atoi(captchaData["y"])
		if err1 != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("It should be string"))
			return
		}

		captchaTokensLock.Lock()
		if (time.Now().Unix() - captchaTokens[dip].timestamp) < 600 {
			if (captchaData["x"] == "" || captchaData["y"] == "") && captchaData["token"] != captchaTokens[dip].token {
				captchaTokensLock.Unlock()
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad request"))
				return
			}
			if x == captchaTokens[dip].x && y == captchaTokens[dip].y {
				buff := make([]byte, 10)
				rand.Read(buff)
				tok := hex.EncodeToString(buff)
				captchaTokens[dip] = CaptchData{token: tok, timestamp: time.Now().Unix()}
				captchaTokensLock.Unlock()
				w.WriteHeader(http.StatusAccepted)
				return
			} else {
				captchaTokensLock.Unlock()
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad request"))
				return
			}
		} else {
			captchaTokens[dip] = CaptchData{}
			captchaTokensLock.Unlock()
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad request"))
	}
}

func Verify(verification string, password string, email string, dip string, w http.ResponseWriter) {
	if verification == "" {
		if len(password) < 8 || !(strings.ContainsAny(password, "!@#$%^&*") && strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") && strings.ContainsAny(password, "0123456789") && strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz")) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		// if the user was already in our storage means its login other
		vcode := rnd.IntN(90000) + 10000
		msg := []byte("To: " + email + "\r\n" +
			"Subject: Shayegan's blog verification code\r\n" +
			"\r\n" +
			"Heres the code " + fmt.Sprint(vcode) + ".\r\n")
		err := smtp.SendMail("smtp.gmail.com:587", auth, myblog.Config["user"], []string{email}, []byte(msg))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Problem with sending email to you happened in server"))
			return
		}
		captchaTokensLock.Lock()
		captchaTokens[dip] = CaptchData{verification: vcode, timestamp: time.Now().Unix()}
		captchaTokensLock.Unlock()
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad request"))
		return
	}
}

func Authing(username string, email string, password string, verification string, dip string, w http.ResponseWriter, registery bool) {
	if vercode, erro := strconv.Atoi(verification); erro != nil {
		// in this case user received the code and its on the header now
		captchaTokensLock.Lock()
		if captchaTokens[dip].verification != vercode {
			captchaTokensLock.Unlock()
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		// now we can perform a shitty sql
		if registery {
			hashedPassword, erri := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if erri != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Server error"))
				return
			}
			refreshToken := make([]byte, 32)
			rand.Read(refreshToken)
			refreshTokenHex := hex.EncodeToString(refreshToken)
			_, err := postgres_client.Exec(
				"INSERT INTO users(userid, email, username, password, refreshToken, timestamp)"+
					" VALUES ($1, $2, $3, $4, $5, $6)", uuid.New().String(), email, username, hashedPassword, refreshTokenHex, time.Now().Unix())
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Server error"))
			} else {
				// give token and write success
				w.WriteHeader(http.StatusAccepted)
				w.Write([]byte(refreshTokenHex))
			}
		} else {
			var rows *sql.Rows
			var erra error
			if strings.Contains(email, "@") {
				rows, erra = postgres_client.Query("SELECT password FROM users WHERE email=$1", email)
			} else {
				rows, erra = postgres_client.Query("SELECT password FROM users WHERE username=$1", username)
			}
			if erra != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Server error"))
				return
			}
			if rows.Next() {
				var refreshTokenHex string
				err1 := rows.Scan(&refreshTokenHex)
				rows.Close()
				if err1 != nil {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte("Bad request"))
					return
				}
				w.WriteHeader(http.StatusAccepted)
				w.Write([]byte(refreshTokenHex))
				return
			} else {
				rows.Close()
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Bad request"))
				return
			}
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad request"))
		return
	}
}
