package myblog

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	rnd "math/rand/v2"
	"net/http"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	"github.com/wenlng/go-captcha-assets/resources/imagesv2"
	"github.com/wenlng/go-captcha-assets/resources/tiles"
	"github.com/wenlng/go-captcha/v2/slide"
	"golang.org/x/crypto/bcrypt"
)

var PublicKey *rsa.PublicKey
var PrivateKey *rsa.PrivateKey

var Auth smtp.Auth

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

func ForgetPassword(w http.ResponseWriter, r *http.Request) { // dosen't require refresh token
	payload := r.Header
	switch r.Method {
	case "GET":
		BucketHandlement("forget", "forgetPassword", w, r)
	case "POST":
		dip := payload.Get("realip")
		cookie, ero := r.Cookie("forget")
		if ero != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		token := cookie.Value

		if token == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		removed, err := Redis_client.LRem(r.Context(), "forget"+dip, 1, token).Result()

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
		} else if removed == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
		} else {
			CaptchaGeneration(dip, "forgetPasswordValidation", "forgetPassword/validation", w, r)
		}
	}
}

func ForgetPasswordValidation(w http.ResponseWriter, r *http.Request) {
	payload := r.Header
	dip := r.Header.Get("realip")
	switch r.Method {
	case "GET":
		BucketHandlement("forgetValidation", "forgetPassword/validation", w, r)
	case "POST":
		email := payload.Get("email")
		cookie, ero := r.Cookie("forgetValidation")
		if ero != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		token := cookie.Value
		cookie, erroj := r.Cookie("forgetPasswordValidation")
		decodedCookie, ear := base64.StdEncoding.DecodeString(cookie.Value)

		if erroj != nil || ear != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if email == "" || token == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		var decodedCookieMap map[string]string
		errjo := json.Unmarshal(decodedCookie, &decodedCookieMap)
		if errjo != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		sig := decodedCookieMap["signature"]
		answer := decodedCookieMap["answer"]
		if sig == "" || answer == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		removed, erro := Redis_client.LRem(r.Context(), "forgetValidation"+dip, 1, token).Result()

		if erro != nil { // dont care if its server error or client bad request
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		} else if removed == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if _, err := mail.ParseAddress(email); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		summedJwt := sha256.Sum256([]byte(answer))
		decodedSig, erri := base64.StdEncoding.DecodeString(sig)
		if erri != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		verErr := rsa.VerifyPKCS1v15(PublicKey, crypto.SHA256, summedJwt[:], decodedSig)

		if verErr != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		var marshaled map[string]string
		errq := json.Unmarshal([]byte(answer), &marshaled)
		if errq != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}

		converted, _ := strconv.Atoi(marshaled["time"])

		if (time.Now().Unix() - int64(converted)) > 120 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		captchaD := payload.Get("captchaAnswer")
		var captchaData map[string]string
		err := json.Unmarshal([]byte(captchaD), &captchaData)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		CaptchaToken(captchaData, "forgetPasswordValidationJWT", "forgetPassword/validation/jwt", email, marshaled["token"], w, dip, r)
	}
}

func ForgetPasswordValidationJWT(w http.ResponseWriter, r *http.Request) {
	payload := r.Header
	switch r.Method {
	case "GET":
		BucketHandlement("forgetValidationJWT", "forgetPassword/validation/jwt", w, r)
	case "POST":
		dip := payload.Get("realip")
		cookie, ero := r.Cookie("forgetValidationJWT")
		if ero != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		token := cookie.Value
		cookie, erroj := r.Cookie("forgetPasswordValidationJWT")
		decodedCookie, ear := base64.StdEncoding.DecodeString(cookie.Value)

		if erroj != nil || ear != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if token == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		removed, err := Redis_client.LRem(r.Context(), "forgetValidationJWT"+dip, 1, token).Result()

		if err != nil { // dont care if its server error or client bad request
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		} else if removed == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		var decodedCookieMap map[string]string
		errwo := json.Unmarshal([]byte(decodedCookie), &decodedCookieMap)
		if errwo != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		answer := decodedCookieMap["answer"]
		sig := decodedCookieMap["signature"]
		if answer == "" || sig == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		summedJwt := sha256.Sum256([]byte(answer))
		decodedSig, erri := base64.StdEncoding.DecodeString(sig)
		if erri != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		verErr := rsa.VerifyPKCS1v15(PublicKey, crypto.SHA256, summedJwt[:], decodedSig)

		if verErr != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		var marshaled map[string]string
		erro := json.Unmarshal([]byte(answer), &marshaled)
		if erro != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}

		converted, _ := strconv.Atoi(marshaled["time"])

		if (time.Now().Unix() - int64(converted)) > 120 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		rows, erra := Postgres_client.Query(r.Context(), "SELECT * FROM users WHERE email=$1", marshaled["email"])
		if erra != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error")) // Invalid user credintals
			return
		}

		if rows.Next() { // this means if the user with this details actually exist
			rows.Close()
			newHashedLink := make([]byte, 15)
			rand.Read(newHashedLink)

			pipe := Redis_client.Pipeline()
			key := base64.StdEncoding.EncodeToString(newHashedLink)
			pipe.LPush(r.Context(), key, marshaled["email"])
			pipe.Expire(r.Context(), key, time.Minute*10)
			_, eri := pipe.Exec(r.Context())
			if eri != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Server error"))
				return
			}

			go func() {
				msg := []byte("To: " + marshaled["email"] + "\r\n" +
					"Subject: Shayegan's blog\r\n" +
					"\r\n" +
					"Click this link to change your password <a href=\"https://myhostnameididntgetyet.ir/auth/forget/\">https://myhostnameididntgetyet.ir/auth/forget/" + key + "</a>.\r\n")
				err1 := smtp.SendMail("smtp.gmail.com:587", Auth, Config["user"], []string{marshaled["email"]}, []byte(msg))
				if err1 != nil {
					log.Println("Problem with smtp server", err1)
				}
			}()
			w.WriteHeader(http.StatusAccepted)
		} else {
			rows.Close()
			w.WriteHeader(http.StatusAccepted)
		}
	}
}

func rateLimiter(dip string, name string, w http.ResponseWriter, r *http.Request) bool {
	counter, err := Redis_client.Incr(r.Context(), name+dip).Result()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server error"))
		return false
	}
	l("Whats the counter:", counter)
	if counter == 1 {
		Redis_client.Expire(r.Context(), name+dip, 30*time.Second)
	}

	if counter > 10 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad request"))
		return false
	}
	return true
}

func ForgetPasswordChangeLink(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	payload := r.Header
	dip := payload.Get("realip")
	switch r.Method {
	case "GET":
		if !rateLimiter(dip, "forgetLink", w, r) {
			return
		}
		email, err := Redis_client.LPop(r.Context(), vars["token"]).Result()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		} else if email == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		} else {
			tokBuffer := make([]byte, 16)
			rand.Read(tokBuffer)
			jsonAnswer := `"{
				"tok":"` + hex.EncodeToString(tokBuffer) + `",
				"email": "` + email + `",
				"time": "` + fmt.Sprintf("%d", time.Now().Unix()) + `"
			}"`
			summed := sha256.Sum256([]byte(jsonAnswer))
			signature, _ := rsa.SignPKCS1v15(rand.Reader, PrivateKey, crypto.SHA256, summed[:])
			http.SetCookie(w, &http.Cookie{
				Name: "forgetPasswordChangeLink",
				Value: base64.StdEncoding.EncodeToString([]byte(`{
				"answer": ` + jsonAnswer + `,
				"signature": "` + base64.StdEncoding.EncodeToString(signature) + `"
			}`)),
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteStrictMode,
				Path:     "/auth/forgetPassword",
				MaxAge:   120,
			})
			w.WriteHeader(http.StatusAccepted)
		}
	case "POST":
		if !rateLimiter(dip, "forgetLink", w, r) {
			return
		}
		password := payload.Get("password")
		answerCookie, erria := r.Cookie("forgetPasswordChangeLink")
		if erria != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		decoded, eroz := base64.StdEncoding.DecodeString(answerCookie.Value)
		if eroz != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		var decodedMap map[string]string
		erriq := json.Unmarshal(decoded, &decodedMap)
		if erriq != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		sig := decodedMap["signature"]
		answer := decodedMap["answer"]
		if sig == "" || answer == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if password == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if strings.ContainsAny(password, "@.\"'") {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if len(password) < 8 || !strings.ContainsAny(password, "ABCDEFGHIKJLMNOPQRSTUVWXYZ") || !strings.ContainsAny(password, "abcdefghikjlmnopqrstuvwxyz") || !strings.ContainsAny(password, "0123456789") || !strings.ContainsAny(password, "!#$%^&*()-_+") {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		summedJwt := sha256.Sum256([]byte(answer))
		decodedSig, erri := base64.StdEncoding.DecodeString(sig)
		if erri != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		verErr := rsa.VerifyPKCS1v15(PublicKey, crypto.SHA256, summedJwt[:], decodedSig)
		if verErr != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		var decodedAnswer map[string]string
		eoo := json.Unmarshal([]byte(answer), &decodedAnswer)
		if eoo != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		converted, _ := strconv.Atoi(decodedAnswer["time"])

		if (time.Now().Unix() - int64(converted)) > 120 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}

		oo, er := Postgres_client.Exec(r.Context(), "UPDATE users SET password=$1 WHERE email=$2", hashedPassword, decodedAnswer["email"])
		if er != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
		} else if oo.RowsAffected() == 0 { // this can be impossible but anyway
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Bad request"))
		} else {
			w.WriteHeader(http.StatusAccepted)
		}
	}
}

func LoginValidationSubmit(w http.ResponseWriter, r *http.Request) {
	payload := r.Header
	dip := payload.Get("realip")
	switch r.Method {
	case "GET":
		BucketHandlement("loginVS", "login/validation/submit", w, r)
	case "POST":
		password := payload.Get("password")
		verification := payload.Get("verification")
		cookie, ero := r.Cookie("loginVS")
		if ero != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		token := cookie.Value

		if password == "" || verification == "" || token == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		removed, err := Redis_client.LRem(r.Context(), "loginVS"+dip, 1, token).Result()

		if err != nil { // dont care if its server error or client bad request
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		} else if removed == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if strings.ContainsAny(password, "@.\"'") {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if len(password) < 8 || !strings.ContainsAny(password, "ABCDEFGHIKJLMNOPQRSTUVWXYZ") || !strings.ContainsAny(password, "abcdefghikjlmnopqrstuvwxyz") || !strings.ContainsAny(password, "0123456789") || !strings.ContainsAny(password, "!#$%^&*()-_+") {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		answerCookie, erria := r.Cookie("loginValidationSubmit")
		if erria != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		decoded, eroz := base64.StdEncoding.DecodeString(answerCookie.Value)
		if eroz != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		var decodedMap map[string]string
		erriq := json.Unmarshal(decoded, &decodedMap)
		if erriq != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		sig := decodedMap["signature"]
		answer := decodedMap["answer"]
		if sig == "" || answer == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		summedJwt := sha256.Sum256([]byte(answer))
		decodedSig, erri := base64.StdEncoding.DecodeString(sig)
		if erri != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		verErr := rsa.VerifyPKCS1v15(PublicKey, crypto.SHA256, summedJwt[:], decodedSig)

		if verErr != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		var marshaled map[string]string
		erro := json.Unmarshal([]byte(answer), &marshaled)
		if erro != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}

		converted, _ := strconv.Atoi(marshaled["time"])

		if (time.Now().Unix() - int64(converted)) > 120 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		count, era := Redis_client.Incr(r.Context(), "counter"+marshaled["email"]).Result()
		if era != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}
		if vercode, erro := strconv.Atoi(verification); erro == nil {
			// in this case user received the code and its on the header now
			vc, err := Redis_client.Get(r.Context(), marshaled["email"]).Result()
			verificationCode, err1 := strconv.Atoi(vc)
			if err1 != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Server error")) // i dont think this happens, anyway
				return
			}
			if err != nil {
				if err == redis.Nil {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte("Bad request")) // i dont think this happens, anyway
					return
				}
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Server error"))
				return
			}
			if verificationCode == vercode {
				_, errr := Redis_client.Del(r.Context(), marshaled["email"]).Result()
				_, era := Redis_client.Del(r.Context(), "counter"+marshaled["email"]).Result()
				if errr != nil || era != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Server error")) // i dont think this happens, anyway
					return
				}
			} else {
				if count == 10 {
					_, errr := Redis_client.Del(r.Context(), marshaled["email"]).Result()
					_, era := Redis_client.Del(r.Context(), "counter"+marshaled["email"]).Result()
					if errr != nil || era != nil {
						w.WriteHeader(http.StatusInternalServerError)
						w.Write([]byte("Server error")) // i dont think this happens, anyway
						return
					}
					w.WriteHeader(http.StatusBadGateway)
					w.Write([]byte("Blocked"))
					return
				}
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad request"))
			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		rows, errio := Postgres_client.Query(r.Context(), "SELECT refreshToken, userid, password FROM users WHERE email=$1", marshaled["email"])
		if errio != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}
		if rows.Next() {
			var refreshTokenHash []byte
			var userid string
			var hashedPassword []byte
			refreshToken := make([]byte, 32)
			rand.Read(refreshToken)
			refreshTokenHex := hex.EncodeToString(refreshToken)

			err1 := rows.Scan(&refreshTokenHash, &userid, &hashedPassword)
			rows.Close()
			if err1 != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad request"))
				return
			}

			if bcrypt.CompareHashAndPassword(hashedPassword, []byte(password)) != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad request"))
				return
			}

			_, erra := Postgres_client.Exec(r.Context(), "UPDATE users SET refreshToken = $1 WHERE email=$2 RETURNING refreshToken, userid, password", refreshToken, marshaled["email"])
			if erra != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Server error"))
				return
			}

			value := `"{
					"userid": "` + userid + `",
					"email": "` + marshaled["email"] + `",
					"refreshToken": "` + refreshTokenHex + `"
			}"`
			hashedValue := sha256.Sum256([]byte(value))

			signature, eara := rsa.SignPKCS1v15(rand.Reader, PrivateKey, crypto.SHA256, hashedValue[:])
			if eara != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Server error"))
				return
			}
			http.SetCookie(w, &http.Cookie{
				Name: "userData",
				Value: base64.StdEncoding.EncodeToString([]byte(`{
					"signature": "` + base64.StdEncoding.EncodeToString(signature) + `",
					"answer": ` + value + `
				}`)),
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteStrictMode,
				Path:     "/",
				MaxAge:   0,
			})

			w.WriteHeader(http.StatusAccepted)
			return
		} else {
			rows.Close()
			w.WriteHeader(http.StatusAccepted)
			return
		}
	}
}

func LoginValidationJWT(w http.ResponseWriter, r *http.Request) {
	payload := r.Header
	dip := payload.Get("realip")
	switch r.Method {
	case "GET":
		BucketHandlement("loginVJ", "login/validation/jwt", w, r)
	case "POST":
		password := payload.Get("password")
		cookie, ero := r.Cookie("loginVJ")
		if ero != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		token := cookie.Value

		if password == "" || token == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		removed, err := Redis_client.LRem(r.Context(), "loginVJ"+dip, 1, token).Result()

		if err != nil { // dont care if its server error or client bad request
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		} else if removed == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if strings.ContainsAny(password, "@.\"'") {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if len(password) < 8 || !strings.ContainsAny(password, "ABCDEFGHIKJLMNOPQRSTUVWXYZ") || !strings.ContainsAny(password, "abcdefghikjlmnopqrstuvwxyz") || !strings.ContainsAny(password, "0123456789") || !strings.ContainsAny(password, "!#$%^&*()-_+") {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		answerCookie, erria := r.Cookie("loginValidationJWT")
		if erria != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		decoded, eroz := base64.StdEncoding.DecodeString(answerCookie.Value)
		if eroz != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		var decodedMap map[string]string
		erriq := json.Unmarshal(decoded, &decodedMap)
		if erriq != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		sig := decodedMap["signature"]
		answer := decodedMap["answer"]
		if sig == "" || answer == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		summedJwt := sha256.Sum256([]byte(answer))
		decodedSig, erri := base64.StdEncoding.DecodeString(sig)
		if erri != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		verErr := rsa.VerifyPKCS1v15(PublicKey, crypto.SHA256, summedJwt[:], decodedSig)

		if verErr != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		var marshaled map[string]string
		erro := json.Unmarshal([]byte(answer), &marshaled)
		if erro != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}

		converted, _ := strconv.Atoi(marshaled["time"])

		if (time.Now().Unix() - int64(converted)) > 120 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		Verify(marshaled["email"], "loginValidationSubmit", "login/validation/submit", w, r)
	}
}

func LoginValidation(w http.ResponseWriter, r *http.Request) {
	payload := r.Header
	switch r.Method {
	case "GET":
		BucketHandlement("loginV", "login/validation", w, r)
	case "POST":
		email := payload.Get("email")
		captchaD := payload.Get("captchaAnswer")
		dip := payload.Get("realip")
		cookie, ero := r.Cookie("loginV")
		if ero != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		token := cookie.Value

		if email == "" || captchaD == "" || token == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		removed, erro := Redis_client.LRem(r.Context(), "loginV"+dip, 1, token).Result()

		if erro != nil { // dont care if its server error or client bad request
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		} else if removed == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if _, er := mail.ParseAddress(email); er != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		answerCookie, erria := r.Cookie("loginValidation")
		if erria != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		decoded, eroz := base64.StdEncoding.DecodeString(answerCookie.Value)
		if eroz != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		var decodedMap map[string]string
		erriq := json.Unmarshal(decoded, &decodedMap)
		if erriq != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		sig := decodedMap["signature"]
		answer := decodedMap["answer"]
		if sig == "" || answer == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		summedJwt := sha256.Sum256([]byte(answer))
		decodedSig, erri := base64.StdEncoding.DecodeString(sig)
		if erri != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		verErr := rsa.VerifyPKCS1v15(PublicKey, crypto.SHA256, summedJwt[:], decodedSig)

		if verErr != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		var marshaled map[string]string
		erra := json.Unmarshal([]byte(answer), &marshaled)
		if erra != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		converted, _ := strconv.Atoi(marshaled["time"])

		if (time.Now().Unix() - int64(converted)) > 120 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		var captchaData map[string]string
		err := json.Unmarshal([]byte(captchaD), &captchaData)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		CaptchaToken(captchaData, "loginValidationJWT", "login/validation/jwt", email, marshaled["token"], w, dip, r)
	}
}

func Login(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		BucketHandlement("login", "login", w, r)
	case "POST":
		dip := r.Header.Get("realip")
		cookie, ero := r.Cookie("login")
		if ero != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		token := cookie.Value
		if token == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		removed, err := Redis_client.LRem(r.Context(), "login"+dip, 1, token).Result()

		if err != nil { // dont care if its server error or client bad request
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		} else if removed == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		CaptchaGeneration(dip, "loginValidation", "login/validation", w, r)
	}
}

func RegisterValidationSubmit(w http.ResponseWriter, r *http.Request) {
	payload := r.Header
	switch r.Method {
	case "GET":
		BucketHandlement("registerVS", "register/validation/submit", w, r)
	case "POST":
		cookie, ero := r.Cookie("registerVS")
		if ero != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		token := cookie.Value
		verification := payload.Get("verification")
		dip := payload.Get("realip")
		password := payload.Get("password")
		username := payload.Get("username")
		if verification == "" || username == "" || password == "" || token == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		removed, err := Redis_client.LRem(r.Context(), "registerVS"+dip, 1, token).Result()

		if err != nil { // dont care if its server error or client bad request
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		} else if removed == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if strings.ContainsAny(username, "@.\"'") || strings.ContainsAny(password, "@.\"'") {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if len(password) < 8 || !strings.ContainsAny(password, "ABCDEFGHIKJLMNOPQRSTUVWXYZ") || !strings.ContainsAny(password, "abcdefghikjlmnopqrstuvwxyz") || !strings.ContainsAny(password, "0123456789") || !strings.ContainsAny(password, "!#$%^&*()-_+") {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		answerCookie, erria := r.Cookie("registerValidationSubmit")
		if erria != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		decoded, eroz := base64.StdEncoding.DecodeString(answerCookie.Value)
		if eroz != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		var decodedMap map[string]string
		erriq := json.Unmarshal(decoded, &decodedMap)
		if erriq != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		sig := decodedMap["signature"]
		answer := decodedMap["answer"]
		if sig == "" || answer == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		summedJwt := sha256.Sum256([]byte(answer))
		decodedSig, erri := base64.StdEncoding.DecodeString(sig)
		if erri != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		verErr := rsa.VerifyPKCS1v15(PublicKey, crypto.SHA256, summedJwt[:], decodedSig)

		if verErr != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		var marshaled map[string]string
		erro := json.Unmarshal([]byte(answer), &marshaled)
		if erro != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}

		converted, _ := strconv.Atoi(marshaled["time"])

		if (time.Now().Unix() - int64(converted)) > 120 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		count, era := Redis_client.Incr(r.Context(), "counter"+marshaled["email"]).Result()
		if era != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}
		if vercode, erro := strconv.Atoi(verification); erro == nil {
			// in this case user received the code and its on the header now
			vc, err := Redis_client.Get(r.Context(), marshaled["email"]).Result()
			verificationCode, err1 := strconv.Atoi(vc)
			if err1 != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Server error")) // i dont think this happens, anyway
				return
			}
			if err != nil {
				if err == redis.Nil {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte("Bad request")) // i dont think this happens, anyway
					return
				}
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Server error"))
				return
			}
			if verificationCode == vercode {
				_, errr := Redis_client.Del(r.Context(), marshaled["email"]).Result()
				_, era := Redis_client.Del(r.Context(), "counter"+marshaled["email"]).Result()
				if errr != nil || era != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Server error")) // i dont think this happens, anyway
					return
				}
			} else {
				if count == 10 {
					_, errr := Redis_client.Del(r.Context(), marshaled["email"]).Result()
					_, era := Redis_client.Del(r.Context(), "counter"+marshaled["email"]).Result()
					if errr != nil || era != nil {
						w.WriteHeader(http.StatusInternalServerError)
						w.Write([]byte("Server error")) // i dont think this happens, anyway
						return
					}
					w.WriteHeader(http.StatusBadGateway)
					w.Write([]byte("Blocked"))
					return
				}
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad request"))
			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		hashedPassword, erri := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if erri != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}
		refreshToken := make([]byte, 32)
		rand.Read(refreshToken)
		refreshTokenHex := hex.EncodeToString(refreshToken)
		query, err := Postgres_client.Query(r.Context(),
			"INSERT INTO users(email, username, password, refreshToken)"+
				" VALUES ($1, $2, $3, $4) RETURNING userid", marshaled["email"], username, hashedPassword, refreshToken)
		if err != nil {
			// this isnt possible but anyway
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		if query.Next() {
			var userid string
			query.Scan(&userid)
			query.Close()
			// give token and write success
			value := `"{
					"userid": "` + userid + `",
					"email": "` + marshaled["email"] + `",
					"refreshToken": "` + refreshTokenHex + `"
			}"`
			hashedValue := sha256.Sum256([]byte(value))

			signature, eara := rsa.SignPKCS1v15(rand.Reader, PrivateKey, crypto.SHA256, hashedValue[:])
			if eara != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Server error"))
				return
			}
			http.SetCookie(w, &http.Cookie{
				Name: "userData",
				Value: base64.StdEncoding.EncodeToString([]byte(`{
					"signature": "` + base64.StdEncoding.EncodeToString(signature) + `",
					"answer": ` + value + `
				}`)),
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteStrictMode,
				Path:     "/",
				MaxAge:   0,
			})

			w.WriteHeader(http.StatusAccepted)
		}
	}
}

func RegisterValidationJWT(w http.ResponseWriter, r *http.Request) {
	payload := r.Header
	switch r.Method {
	case "GET":
		BucketHandlement("registerVJ", "register/validation/jwt", w, r)
	case "POST":
		cookie, ero := r.Cookie("registerVJ")
		if ero != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		token := cookie.Value
		dip := payload.Get("realip")
		username := payload.Get("username")
		password := payload.Get("password")
		if username == "" || password == "" || token == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		removed, err := Redis_client.LRem(r.Context(), "registerVJ"+dip, 1, token).Result()

		if err != nil { // dont care if its server error or client bad request
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		} else if removed == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if strings.ContainsAny(username, "@.\"'") || strings.ContainsAny(password, "@.\"'") {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if len(password) < 8 || !strings.ContainsAny(password, "ABCDEFGHIKJLMNOPQRSTUVWXYZ") || !strings.ContainsAny(password, "abcdefghikjlmnopqrstuvwxyz") || !strings.ContainsAny(password, "0123456789") || !strings.ContainsAny(password, "!#$%^&*()-_+") {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		answerCookie, erria := r.Cookie("registerValidationJWT")
		if erria != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		decoded, eroz := base64.StdEncoding.DecodeString(answerCookie.Value)
		if eroz != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		var decodedMap map[string]string
		erriq := json.Unmarshal(decoded, &decodedMap)
		if erriq != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		sig := decodedMap["signature"]
		answer := decodedMap["answer"]
		if sig == "" || answer == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		summedJwt := sha256.Sum256([]byte(answer))
		decodedSig, erri := base64.StdEncoding.DecodeString(sig)
		if erri != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		verErr := rsa.VerifyPKCS1v15(PublicKey, crypto.SHA256, summedJwt[:], decodedSig)

		if verErr != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		var marshaled map[string]string
		erro := json.Unmarshal([]byte(answer), &marshaled)
		if erro != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}

		converted, _ := strconv.Atoi(marshaled["time"])

		if (time.Now().Unix() - int64(converted)) > 120 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		Verify(marshaled["email"], "registerValidationSubmit", "register/validation/submit", w, r)
	}
}

func RegisterValidation(w http.ResponseWriter, r *http.Request) {
	payload := r.Header
	switch r.Method {
	case "GET":
		BucketHandlement("registerV", "register/validation", w, r)
	case "POST":
		captchaD := payload.Get("captchaAnswer")
		email := payload.Get("email")
		dip := payload.Get("realip")
		cookie, ero := r.Cookie("registerV")
		if ero != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		token := cookie.Value
		if captchaD == "" || email == "" || token == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		removed, erro := Redis_client.LRem(r.Context(), "registerV"+dip, 1, token).Result()

		if erro != nil { // dont care if its server error or client bad request
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		} else if removed == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if _, er := mail.ParseAddress(email); er != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		answerCookie, erria := r.Cookie("registerValidation")
		if erria != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		decoded, eroz := base64.StdEncoding.DecodeString(answerCookie.Value)
		if eroz != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		var decodedMap map[string]string
		erriq := json.Unmarshal(decoded, &decodedMap)
		if erriq != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		sig := decodedMap["signature"]
		answer := decodedMap["answer"]
		if sig == "" || answer == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		summedJwt := sha256.Sum256([]byte(answer))
		decodedSig, erri := base64.StdEncoding.DecodeString(sig)
		if erri != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		verErr := rsa.VerifyPKCS1v15(PublicKey, crypto.SHA256, summedJwt[:], decodedSig)

		if verErr != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		var marshaled map[string]string
		erra := json.Unmarshal([]byte(answer), &marshaled)
		if erra != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		converted, _ := strconv.Atoi(marshaled["time"])

		if (time.Now().Unix() - int64(converted)) > 120 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		var captchaData map[string]string
		err := json.Unmarshal([]byte(captchaD), &captchaData)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		CaptchaToken(captchaData, "registerValidationJWT", "register/validation/jwt", email, marshaled["token"], w, dip, r)
	}
}

func Register(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		BucketHandlement("register", "register", w, r)
	case "POST":
		cookie, ero := r.Cookie("register")
		if ero != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		token := cookie.Value
		log.Println("Token seems obtained")
		if token == "" {
			log.Println("What the fuck?")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		dip := r.Header.Get("realip")
		log.Println("Ip of that guy", dip)

		removed, err := Redis_client.LRem(r.Context(), "register"+dip, 1, token).Result()

		if err != nil { // dont care if its server error or client bad request
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
		} else if removed == 0 {
			log.Println("Removed problem")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
		} else {
			CaptchaGeneration(dip, "registerValidation", "register/validation", w, r)
		}
	}
}

func CaptchaGeneration(dip string, name string, endpoint string, w http.ResponseWriter, r *http.Request) {
	captcha := Init()
	captData, err := captcha.Generate()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server error"))
		return
	}

	dotData := captData.GetData()
	if dotData == nil {
		log.Println("ERROR FOR CAPTCHA")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server error"))
		return
	}

	masterImage, _ := captData.GetMasterImage().ToBase64()
	tileImage, _ := captData.GetTileImage().ToBase64()
	buffTok := make([]byte, 32)
	rand.Read(buffTok)
	buffTokHex := hex.EncodeToString(buffTok)
	answer := `"{
				"token": "` + buffTokHex + `",
				"time": "` + fmt.Sprintf("%d", time.Now().Unix()) + `"
			}"`
	summedAnswer := sha256.Sum256([]byte(answer))
	signature, erri := rsa.SignPKCS1v15(rand.Reader, PrivateKey, crypto.SHA256, summedAnswer[:])
	if erri != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server error"))
		return
	}

	// we should have store the answer in some storage,
	log.Println("This is the answer, X:", captData.GetData().X, ", Y:", captData.GetData().Y)
	Redis_client.Set(r.Context(), "captcha"+dip+buffTokHex, fmt.Sprintf("%d,%d", captData.GetData().X, captData.GetData().Y), 1*time.Minute)
	http.SetCookie(w, &http.Cookie{
		Name: name,
		Value: base64.StdEncoding.EncodeToString([]byte(`{
			"signature": "` + base64.StdEncoding.EncodeToString(signature) + `",
			"answer": ` + answer + `
		}`)),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/auth/" + endpoint, // Only sent to auth endpoints
		MaxAge:   120,
	})
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{
		"masterImage": "` + masterImage + `",
		"titleImage": "` + tileImage + `"
	}`))
}

func CaptchaToken(captchaData map[string]string, name string, endpoint string, email string, token string, w http.ResponseWriter, dip string, r *http.Request) {
	x, err := strconv.Atoi(captchaData["x"])
	if err != nil { // this blocks are for testing and might be removed or above code might be changed
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad request"))
		return
	}
	y, err1 := strconv.Atoi(captchaData["y"])
	if err1 != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad request"))
		return
	}

	str, err := Redis_client.GetDel(r.Context(), "captcha"+dip+token).Result()
	if err != nil {
		if err == redis.Nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server error"))
		return
	}
	ts := strings.Split(str, ",")
	xx, _ := strconv.Atoi(ts[0])
	yy, _ := strconv.Atoi(ts[1])
	if captchaData["x"] == "" || captchaData["y"] == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad request"))
		return
	}
	if x == xx && y == yy {
		buff := make([]byte, 10)
		rand.Read(buff)
		tok := hex.EncodeToString(buff)
		//jwt
		jsonAnswer := `"{
				"tok":"` + tok + `",
				"email":"` + email + `",
				"time":"` + fmt.Sprintf("%d", time.Now().Unix()) + `"
			}"`
		summed := sha256.Sum256([]byte(jsonAnswer))
		signature, _ := rsa.SignPKCS1v15(rand.Reader, PrivateKey, crypto.SHA256, summed[:])
		http.SetCookie(w, &http.Cookie{
			Name: name,
			Value: base64.StdEncoding.EncodeToString([]byte(`{
				"answer": ` + jsonAnswer + `,
				"signature": "` + base64.StdEncoding.EncodeToString(signature) + `"
			}`)),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			Path:     "/auth/" + endpoint, // Only sent to auth endpoints
			MaxAge:   120,
		})
		w.WriteHeader(http.StatusAccepted)
		return
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad request"))
		return
	}
}

func Verify(email string, name string, endpoint string, w http.ResponseWriter, r *http.Request) {
	// if the user was already in our storage means its login other
	vcode := rnd.IntN(90000) + 10000
	go func() {
		msg := []byte("To: " + email + "\r\n" +
			"Subject: Shayegan's blog verification code\r\n" +
			"\r\n" +
			"Heres the code " + fmt.Sprint(vcode) + ".\r\n")
		err := smtp.SendMail("smtp.gmail.com:587", Auth, Config["user"], []string{email}, []byte(msg))
		if err != nil {
			log.Println("Problem with smtp server:", err)
		}
	}()
	Redis_client.Set(r.Context(), email, vcode, 3*time.Minute)
	buff := make([]byte, 10)
	rand.Read(buff)
	tok := hex.EncodeToString(buff)
	//jwt
	jsonAnswer := `"{
				"tok":"` + tok + `",
				"email":"` + email + `",
				"time":"` + fmt.Sprintf("%d", time.Now().Unix()) + `"
			}"`
	summed := sha256.Sum256([]byte(jsonAnswer))
	signature, _ := rsa.SignPKCS1v15(rand.Reader, PrivateKey, crypto.SHA256, summed[:])
	http.SetCookie(w, &http.Cookie{
		Name: name,
		Value: base64.StdEncoding.EncodeToString([]byte(`{
				"answer": ` + jsonAnswer + `,
				"signature": "` + base64.StdEncoding.EncodeToString(signature) + `"
			}`)),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/auth/" + endpoint, // Only sent to auth endpoints
		MaxAge:   120,
	})
	w.WriteHeader(http.StatusAccepted)
}
