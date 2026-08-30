package main

import (
	"crypto/x509"
	"encoding/json"
	"log"
	"myblog/myblog"
	"net/http"
	"net/smtp"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/wenlng/go-captcha-assets/resources/imagesv2"
	"github.com/wenlng/go-captcha-assets/resources/tiles"
	"github.com/wenlng/go-captcha/v2/slide"
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

func Init() slide.Captcha {
	builder := slide.NewBuilder(
		slide.WithEnableGraphVerticalRandom(true),
	)

	imgs, _ := imagesv2.GetImages()

	graphs, _ := tiles.GetTiles()

	var newGraphs = make([]*slide.GraphImage, 0, len(graphs))
	for i := range graphs {
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

func main() {
	json.Unmarshal(myblog.ConfigBuffer, &myblog.Config)
	myblog.InitRDB()
	myblog.InitPDB()
	myblog.Captcha = Init()
	router := mux.NewRouter()

	myblog.Auth = smtp.PlainAuth("", myblog.Config["username"], myblog.Config["password"], "smtp.gmail.com")
	router.HandleFunc("/login", myblog.Login)
	router.HandleFunc("/login/validation", myblog.LoginValidation)
	router.HandleFunc("/login/validation/jwt", myblog.LoginValidationJWT)
	router.HandleFunc("/login/validation/submit", myblog.LoginValidationSubmit)
	router.HandleFunc("/register", myblog.Register)
	router.HandleFunc("/register/validation", myblog.RegisterValidation)
	router.HandleFunc("/register/validation/jwt", myblog.RegisterValidationJWT)
	router.HandleFunc("/register/validation/submit", myblog.RegisterValidationSubmit)
	router.HandleFunc("/forgetPassword", myblog.ForgetPassword)
	router.HandleFunc("/forgetPassword/validate", myblog.ForgetPasswordValidation)
	router.HandleFunc("/forgetPassword/validate/jwt", myblog.ForgetPasswordValidationJWT)
	router.HandleFunc("/forgetPassword/{key}", myblog.ForgetPasswordChangeLink)

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
		"csrf-token",
	}), handlers.AllowedMethods([]string{"GET", "POST", "OPTIONS"}))(rrouter)

	privateK, e := os.ReadFile("~/private_key")
	if e != nil {
		log.Fatal(e)
	}

	prk, e := x509.ParsePKCS1PrivateKey(privateK)
	if e != nil {
		log.Fatal(e)
	}

	publicK, e := os.ReadFile("~/public_key")
	if e != nil {
		log.Fatal(e)
	}

	puk, e := x509.ParsePKCS1PublicKey(publicK)
	if e != nil {
		log.Fatal(e)
	}

	myblog.PrivateKey = prk
	myblog.PublicKey = puk

	server := &http.Server{
		Handler:      rrrouter,
		Addr:         "127.0.0.1:1234",
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  30 * time.Second,
	}

	server.ListenAndServe()

}
