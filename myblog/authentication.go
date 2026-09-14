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
	"math"
	rnd "math/rand/v2"
	"net/http"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	"github.com/wenlng/go-captcha/v2/slide"
	"golang.org/x/crypto/bcrypt"
)

var Auth smtp.Auth
var Captcha slide.Captcha

func ForgetPassword(w http.ResponseWriter, r *http.Request) { // dosen't require refresh token
	switch r.Method {
	case "POST":
		CaptchaGeneration("forgetPasswordValidation", "forgetPassword/validation", w, r)
	}
}

func ForgetPasswordValidation(w http.ResponseWriter, r *http.Request) {
	payload := r.Header
	switch r.Method {
	case "POST":
		email := payload.Get("email")
		cookie, erroj := r.Cookie("forgetPasswordValidation")
		decodedCookie, ear := base64.StdEncoding.DecodeString(cookie.Value)

		if erroj != nil || ear != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if email == "" {
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

		converted, err := strconv.Atoi(marshaled["time"])
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if (time.Now().Unix() - int64(converted)) > 120 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		captchaD := payload.Get("captchaAnswer")
		var captchaData map[string]string
		err = json.Unmarshal([]byte(captchaD), &captchaData)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		CaptchaToken(captchaData, "forgetPasswordValidationJWT", "forgetPassword/validation/jwt", email, "", "", marshaled["token"], w, r)
	}
}

func ForgetPasswordValidationJWT(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		cookie, erroj := r.Cookie("forgetPasswordValidationJWT")
		decodedCookie, ear := base64.StdEncoding.DecodeString(cookie.Value)

		if erroj != nil || ear != nil {
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

		converted, err := strconv.Atoi(marshaled["time"])
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
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
			newHashedLink := make([]byte, 60)
			rand.Read(newHashedLink)

			pipe := Redis_client.Pipeline()
			key := base64.StdEncoding.EncodeToString(newHashedLink)
			pipe.LPush(r.Context(), marshaled["email"]+key, marshaled["email"])
			pipe.Expire(r.Context(), marshaled["email"]+key, time.Minute*10)
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
				err1 := smtp.SendMail("smtp.gmail.com:465", Auth, Config["user"], []string{marshaled["email"]}, []byte(msg))
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

func Logout(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		userData, eri := r.Cookie("userData")
		if eri != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		var marshaled map[string]string
		eri = json.Unmarshal([]byte(userData.Value), &marshaled)
		if eri != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if marshaled["sessionid"] == "" || marshaled["email"] == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		rowi, erra := Postgres_client.Exec(r.Context(), "DELETE FROM sessions WHERE sessionid=$1 AND refreshToken=$2 AND email=$3", marshaled["sessionid"], marshaled["refreshToken"], marshaled["email"])
		if erra != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}

		if rowi.RowsAffected() == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "userData",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			Path:     "/",
			MaxAge:   -1,
		})

		w.WriteHeader(http.StatusAccepted)
	}
}

func ForgetPasswordChangeLink(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	payload := r.Header
	switch r.Method {
	case "POST":
		password := payload.Get("password")
		token := vars["token"]
		email := vars["email"]
		if password == "" || token == "" || email == "" || len(password) > 1000 {
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

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}
		ea := Redis_client.Del(r.Context(), email+token)
		if ea.Err() != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}
		res, erri := ea.Result()
		if erri != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		} else if res == 0 {
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte("Bad request"))
			return
		} else {
			oo, er := Postgres_client.Exec(r.Context(), "UPDATE users SET password=$1 WHERE email=$2", hashedPassword, email)
			if er != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Server error"))
				return
			} else if oo.RowsAffected() == 0 {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Bad request"))
				return
			}
			w.WriteHeader(http.StatusAccepted)
		}
	}
}

func LoginValidationSubmit(w http.ResponseWriter, r *http.Request) {
	payload := r.Header
	switch r.Method {
	case "POST":
		verification := payload.Get("verification")
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

		timee := marshaled["time"]
		password := marshaled["password"]
		email := marshaled["email"]
		tok := marshaled["tok"]
		if timee == "" || password == "" || email == "" || tok == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		converted, err := strconv.Atoi(timee)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if (time.Now().Unix() - int64(converted)) > 120 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		count, era := Redis_client.Incr(r.Context(), "counter"+tok+email).Result()
		if era != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}

		if count == 1 {
			Redis_client.Expire(r.Context(), "counter"+tok+email, 2*time.Minute)
		}

		if vercode, erro := strconv.Atoi(verification); erro == nil {
			// in this case user received the code and its on the header now
			vc, err := Redis_client.Get(r.Context(), tok+email).Result()
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
				_, errr := Redis_client.Del(r.Context(), tok+email).Result()
				_, era := Redis_client.Del(r.Context(), "counter"+tok+email).Result()
				if errr != nil || era != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Server error")) // i dont think this happens, anyway
					return
				}
			} else {
				if count == 10 {
					_, errr := Redis_client.Del(r.Context(), tok+email).Result()
					_, era := Redis_client.Del(r.Context(), "counter"+tok+email).Result()
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
		l("OK im here in login")
		rows, errio := Postgres_client.Query(r.Context(), "SELECT userid, password FROM users WHERE email=$1", email)
		if errio != nil {
			l("server cheror?", errio)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}
		if rows.Next() {
			l("ok some pussy pussy")
			var userid string
			var hashedPassword []byte
			refreshToken := make([]byte, 1000)
			rand.Read(refreshToken)
			refreshTokenHex := hex.EncodeToString(refreshToken)

			err1 := rows.Scan(&userid, &hashedPassword)
			rows.Close()
			if err1 != nil {
				l("Its because of hereee")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad request"))
				return
			}

			if bcrypt.CompareHashAndPassword(hashedPassword, []byte(password)) != nil {
				l("No its heeeree")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad request"))
				return
			}

			var sessionid string
			eri := Postgres_client.QueryRow(r.Context(), "UPDATE sessions SET refreshToken = $1 WHERE userid=$2 RETURNING sessionid", refreshToken, userid).Scan(&sessionid)
			if eri != nil {
				l(eri)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Server error"))
				return
			}
			value := `{"userid": "` + userid + `","email": "` + email + `","refreshToken": "` + refreshTokenHex + `","sessionId": "` + sessionid + `"}`
			http.SetCookie(w, &http.Cookie{
				Name:     "userData",
				Value:    base64.StdEncoding.EncodeToString([]byte(value)),
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
			l("Hereeee")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

func LoginValidationJWT(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
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

		timee := marshaled["time"]
		password := marshaled["password"]

		if timee == "" || password == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		converted, err := strconv.Atoi(timee)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		if (time.Now().Unix() - int64(converted)) > 120 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		Verify(marshaled["email"], "loginValidationSubmit", "login/validation", "", password, w, r)
	}
}

func LoginValidation(w http.ResponseWriter, r *http.Request) {
	payload := r.Header
	switch r.Method {
	case "POST":
		email := payload.Get("email")
		password := payload.Get("password")
		captchaD := payload.Get("captchaAnswer")
		if strings.ContainsAny(password, "@.\"'") {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if email == "" || captchaD == "" || password == "" || len(password) > 1000 || len(captchaD) > 100 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if len(password) < 8 || !strings.ContainsAny(password, "ABCDEFGHIKJLMNOPQRSTUVWXYZ") || !strings.ContainsAny(password, "abcdefghikjlmnopqrstuvwxyz") || !strings.ContainsAny(password, "0123456789") || !strings.ContainsAny(password, "!#$%^&*()-_+") {
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

		timee := marshaled["time"]
		token := marshaled["token"]

		if timee == "" || token == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		converted, err := strconv.Atoi(timee)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		if (time.Now().Unix() - int64(converted)) > 120 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		var captchaData map[string]string
		err = json.Unmarshal([]byte(captchaD), &captchaData)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		CaptchaToken(captchaData, "loginValidationJWT", "login/validation/jwt", email, "", password, token, w, r)
	}
}

func Login(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		CaptchaGeneration("loginValidation", "login", w, r)
	}
}

func RegisterValidationSubmit(w http.ResponseWriter, r *http.Request) {
	payload := r.Header
	switch r.Method {
	case "POST":
		verification := payload.Get("verification")
		if verification == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		answerCookie, erria := r.Cookie("registerValidationSubmit")
		if erria != nil {
			l("registerValidationSubmit")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		decoded, eroz := base64.StdEncoding.DecodeString(answerCookie.Value)
		if eroz != nil {
			l("decoded answerCookie")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		fmt.Println(decoded)
		var decodedMap map[string]string
		erriq := json.Unmarshal(decoded, &decodedMap)
		if erriq != nil {
			l("unmarshal", erriq)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		sig := decodedMap["signature"]
		answer := decodedMap["answer"]
		if sig == "" || answer == "" {
			l("sigo shit")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		summedJwt := sha256.Sum256([]byte(answer))
		decodedSig, erri := base64.StdEncoding.DecodeString(sig)
		if erri != nil {
			l("decoded issue")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		verErr := rsa.VerifyPKCS1v15(PublicKey, crypto.SHA256, summedJwt[:], decodedSig)

		if verErr != nil {
			l("verification problem")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		var marshaled map[string]string
		erro := json.Unmarshal([]byte(answer), &marshaled)
		if erro != nil {
			l("marshaled")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}

		timee := marshaled["time"]
		email := marshaled["email"]
		tok := marshaled["tok"]
		username := marshaled["username"]
		password := marshaled["password"]

		if timee == "" || email == "" || tok == "" || username == "" || password == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		converted, err := strconv.Atoi(timee)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if (time.Now().Unix() - int64(converted)) > 120 {
			l("time")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		count, era := Redis_client.Incr(r.Context(), "counter"+tok+email).Result()
		if era != nil {
			l("counter issue")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}

		if count == 1 {
			Redis_client.Expire(r.Context(), "counter"+tok+email, 2*time.Minute)
		}

		if vercode, erro := strconv.Atoi(verification); erro == nil {
			// in this case user received the code and its on the header now
			vc, err := Redis_client.Get(r.Context(), tok+email).Result()
			verificationCode, err1 := strconv.Atoi(vc)
			if err1 != nil {
				l("vercode error")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Server error")) // i dont think this happens, anyway
				return
			}
			if err != nil {
				if err == redis.Nil {
					l("redis Nili")
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte("Bad request")) // i dont think this happens, anyway
					return
				}
				l("serverili")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Server error"))
				return
			}
			if verificationCode == vercode {
				l("fucking same")
				_, errr := Redis_client.Del(r.Context(), tok+email).Result()
				_, era := Redis_client.Del(r.Context(), "counter"+tok+email).Result()
				if errr != nil || era != nil {
					l("fucking eeeerorrr")
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Server error")) // i dont think this happens, anyway
					return
				}
			} else {
				if count == 10 {
					l("goz jerk")
					_, errr := Redis_client.Del(r.Context(), tok+email).Result()
					_, era := Redis_client.Del(r.Context(), "counter"+tok+email).Result()
					if errr != nil || era != nil {
						l("fuckin errorrr in c 10")
						w.WriteHeader(http.StatusInternalServerError)
						w.Write([]byte("Server error")) // i dont think this happens, anyway
						return
					}
					l("blocked")
					w.WriteHeader(http.StatusBadGateway)
					w.Write([]byte("Blocked"))
					return
				}
				l("bad shito dick")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad request"))
			}
		} else {
			l("bad shito dick anyway")

			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		hashedPassword, erri := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if erri != nil {
			l("server shitty pussy")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}
		fmt.Println("IM here")
		refreshToken := make([]byte, 1000)
		rand.Read(refreshToken)
		refreshTokenHex := hex.EncodeToString(refreshToken)
		fmt.Println("IM here tooo")
		var userid string
		err = Postgres_client.QueryRow(r.Context(),
			"INSERT INTO users(email, username, password)"+
				" VALUES ($1, $2, $3) RETURNING userid", email, username, hashedPassword).Scan(&userid)
		if err == nil {
			l("no next")
			// give token and write success
			var sessionid string
			err = Postgres_client.QueryRow(r.Context(), "INSERT INTO sessions(userid, refreshToken, email) VALUES ($1, $2, $3) RETURNING sessionid", userid, refreshToken, marshaled["email"]).Scan(&sessionid)
			if err != nil {
				l("Its hereo!")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad request"))
				return
			}
			value := `{"userid": "` + userid + `","email": "` + email + `","refreshToken": "` + refreshTokenHex + `","sessionId": "` + sessionid + `"}`

			http.SetCookie(w, &http.Cookie{
				Name:     "userData",
				Value:    base64.StdEncoding.EncodeToString([]byte(value)),
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteStrictMode,
				Path:     "/",
				MaxAge:   0,
			})

			l(value)
			fmt.Println("Whole registeration is completed")
			w.WriteHeader(http.StatusAccepted)
			return
		} else {
			l("nooooooooooooo", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

func RegisterValidationJWT(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		l("So here we are")
		answerCookie, erria := r.Cookie("registerValidationJWT")
		if erria != nil {
			l("shashkir?")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		decoded, eroz := base64.StdEncoding.DecodeString(answerCookie.Value)
		if eroz != nil {
			l("ankir?")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		var decodedMap map[string]string
		erriq := json.Unmarshal(decoded, &decodedMap)
		if erriq != nil {
			l("sagjerk?")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		sig := decodedMap["signature"]
		answer := decodedMap["answer"]
		if sig == "" || answer == "" {
			l("kesafat?")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		summedJwt := sha256.Sum256([]byte(answer))
		decodedSig, erri := base64.StdEncoding.DecodeString(sig)
		if erri != nil {
			l("kosafjerk?")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		verErr := rsa.VerifyPKCS1v15(PublicKey, crypto.SHA256, summedJwt[:], decodedSig)

		if verErr != nil {
			l("sagejerk?")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		var marshaled map[string]string
		erro := json.Unmarshal([]byte(answer), &marshaled)
		if erro != nil {
			l("or lakejerk?")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}

		converted, err := strconv.Atoi(marshaled["time"])
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if (time.Now().Unix() - int64(converted)) > 120 {
			l("dangoz?")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		l("Ok its for verify functions seems like")
		Verify(marshaled["email"], "registerValidationSubmit", "register/validation", marshaled["username"], marshaled["password"], w, r)
	}
}

func RegisterValidation(w http.ResponseWriter, r *http.Request) {
	payload := r.Header
	switch r.Method {
	case "POST":
		log.Println("somethong for registerValidation need to be happen right?")
		captchaD := payload.Get("captchaAnswer")
		log.Println("The answer we found: ", captchaD)
		email := payload.Get("email")
		username := payload.Get("username")
		password := payload.Get("password")
		log.Println("Data we received from user,", email)
		log.Println("CaptchaD:")
		log.Println(captchaD)
		if username == "" || password == "" || email == "" || captchaD == "" || len(username) > 100 || len(password) > 1000 || len(captchaD) > 100 {
			l("userCSRF")
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

		if _, er := mail.ParseAddress(email); er != nil {
			l("email problem")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		answerCookie, erria := r.Cookie("registerValidation")
		if erria != nil {
			l("registerValidation")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		decoded, eroz := base64.StdEncoding.DecodeString(answerCookie.Value)
		if eroz != nil {
			l("decoded")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		fmt.Println(string(decoded))
		var decodedMap map[string]string
		erriq := json.Unmarshal(decoded, &decodedMap)
		if erriq != nil {
			l("decodedMap!", erriq)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		sig := decodedMap["signature"]
		fmt.Println("Signature:", sig)
		answer := decodedMap["answer"]
		fmt.Println("Answer:", answer)
		if sig == "" || answer == "" {
			l("sig or answer")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		summedJwt := sha256.Sum256([]byte(answer))
		decodedSig, erri := base64.StdEncoding.DecodeString(sig)
		if erri != nil {
			l("decoded sig")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		verErr := rsa.VerifyPKCS1v15(PublicKey, crypto.SHA256, summedJwt[:], decodedSig)

		if verErr != nil {
			l("verification problem")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		var marshaled map[string]string
		erra := json.Unmarshal([]byte(answer), &marshaled)
		if erra != nil {
			l("marshaled")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		timee := marshaled["time"]
		token := marshaled["token"]

		if timee == "" || token == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		converted, err := strconv.Atoi(timee)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		if (time.Now().Unix() - int64(converted)) > 120 {
			l("time problem")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		var captchaData map[string]string
		err = json.Unmarshal([]byte(captchaD), &captchaData)
		if err != nil {
			l("captchaData problem")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		CaptchaToken(captchaData, "registerValidationJWT", "register/validation/jwt", email, username, password, token, w, r)
	}
}

func Register(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		CaptchaGeneration("registerValidation", "register", w, r)
	}
}

func CaptchaGeneration(name string, endpoint string, w http.ResponseWriter, r *http.Request) {
	captData, err := Captcha.Generate()
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
	buffTok := make([]byte, 1000)
	rand.Read(buffTok)
	buffTokHex := hex.EncodeToString(buffTok)
	answer := `"{\"token\": \"` + buffTokHex + `\",\"time\": \"` + fmt.Sprintf("%d", time.Now().Unix()) + `\"}"`
	shouldSigned := `{"token": "` + buffTokHex + `","time": "` + fmt.Sprintf("%d", time.Now().Unix()) + `"}`
	fmt.Println("The thing i signed:", shouldSigned)
	summedAnswer := sha256.Sum256([]byte(shouldSigned))
	signature, erri := rsa.SignPKCS1v15(rand.Reader, PrivateKey, crypto.SHA256, summedAnswer[:])
	if erri != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server error"))
		return
	}

	// we should have store the answer in some storage,
	log.Println("X before:", captData.GetData().X)
	log.Println("This is the answer, X:", math.Round(float64(captData.GetData().X)/10)*10, ", Y:", captData.GetData().Y)
	log.Println("This is the width and height", captData.GetData().Width, captData.GetData().Height)
	log.Println("This is the dx and dy", captData.GetData().DX, captData.GetData().DY)

	pipe := Redis_client.Pipeline()
	pipe.Del(r.Context(), "captcha"+buffTokHex)

	strishit := fmt.Sprintf("%d,%d", int64(math.Round(float64(captData.GetData().X)/10)*10), captData.GetData().Y)
	l(strishit)
	pipe.Set(r.Context(), "captcha"+buffTokHex, strishit, 1*time.Minute)
	pipe.Exec(r.Context())
	http.SetCookie(w, &http.Cookie{
		Name: name,
		Value: base64.StdEncoding.EncodeToString([]byte(`{
			"signature": "` + base64.StdEncoding.EncodeToString(signature) + `",
			"answer": ` + answer + `
		}`)),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/" + endpoint, // Only sent to auth endpoints
		MaxAge:   60,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{
		"masterImage": "` + masterImage + `",
		"titleImage": "` + tileImage + `",
		"dx": ` + fmt.Sprintf("%d", captData.GetData().DX) + `,
		"dy": ` + fmt.Sprintf("%d", captData.GetData().DY) + `
	}`))
}

func CaptchaToken(captchaData map[string]string, name string, endpoint string, email string, username string, password string, token string, w http.ResponseWriter, r *http.Request) {
	fmt.Println("Captcha toooooken")
	x, err := strconv.Atoi(captchaData["x"])
	if err != nil { // this blocks are for testing and might be removed or above code might be changed
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad request"))
		return
	}

	str, err := Redis_client.GetDel(r.Context(), "captcha"+token).Result()
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
	xx, err := strconv.Atoi(ts[0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad request"))
		return
	}
	if captchaData["x"] == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad request"))
		return
	}
	l("The answeeers! x and XX:", x, xx)
	if x == xx {
		l("So we got reached here???")
		buff := make([]byte, 10)
		rand.Read(buff)
		tok := hex.EncodeToString(buff)
		//jwt
		var jsonAnswerShould string
		var jsonAnswer string
		if username != "" && password != "" {
			jsonAnswerShould = `{"tok":"` + tok + `","email":"` + email + `","username": "` + username + `","password": "` + password + `","time":"` + fmt.Sprintf("%d", time.Now().Unix()) + `"}`
			jsonAnswer = `"{\"tok\":\"` + tok + `\",\"email\":\"` + email + `\",\"username\": \"` + username + `\",\"password\": \"` + password + `\",\"time\":\"` + fmt.Sprintf("%d", time.Now().Unix()) + `\"}"`
		} else if username == "" {
			jsonAnswerShould = `{"tok":"` + tok + `","email":"` + email + `","password": "` + password + `","time":"` + fmt.Sprintf("%d", time.Now().Unix()) + `"}`
			jsonAnswer = `"{\"tok\":\"` + tok + `\",\"email\":\"` + email + `\",\"password\": \"` + password + `\",\"time\":\"` + fmt.Sprintf("%d", time.Now().Unix()) + `\"}"`
		} else if username == "" && password == "" {
			jsonAnswerShould = `{"tok":"` + tok + `","email":"` + email + `","time":"` + fmt.Sprintf("%d", time.Now().Unix()) + `"}`
			jsonAnswer = `"{\"tok\":\"` + tok + `\",\"email\":\"` + email + `\",\"time\":\"` + fmt.Sprintf("%d", time.Now().Unix()) + `\"}"`
		}
		summed := sha256.Sum256([]byte(jsonAnswerShould))
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
			Path:     "/" + endpoint,
			MaxAge:   120,
		})
		log.Println("Ok its accepted!")
		ass := base64.StdEncoding.EncodeToString([]byte(`{
				"answer": ` + jsonAnswer + `,
				"signature": "` + base64.StdEncoding.EncodeToString(signature) + `"
			}`))
		log.Println(ass)
		w.WriteHeader(http.StatusAccepted)
		return
	} else {
		l("wait but why???")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad request"))
		return
	}
}

func Verify(email string, name string, endpoint string, username string, password string, w http.ResponseWriter, r *http.Request) {
	vcode := rnd.IntN(90000) + 10000
	log.Println("The answer for verify is", vcode)
	go func() {
		msg := []byte("To: " + email + "\r\n" +
			"Subject: Shayegan's blog verification code\r\n" +
			"\r\n" +
			"Heres the code " + fmt.Sprint(vcode) + ".\r\n")
		fmt.Println("the user", Config["user"])
		fmt.Println("the password", Config["password"])
		err := smtp.SendMail("smtp.gmail.com:465", Auth, Config["username"], []string{email}, []byte(msg))
		if err != nil {
			log.Println("Problem with smtp server:", err)
		}
	}()
	buff := make([]byte, 32)
	rand.Read(buff)
	tok := hex.EncodeToString(buff)
	Redis_client.Set(r.Context(), tok+email, vcode, 2*time.Minute)
	//jwt
	var jsonAnswer string
	var jsonAnswerShould string
	if username != "" {
		jsonAnswerShould = `{"tok":"` + tok + `","email":"` + email + `","username": "` + username + `","password": "` + password + `","time":"` + fmt.Sprintf("%d", time.Now().Unix()) + `"}`
		jsonAnswer = `"{\"tok\":\"` + tok + `\",\"email\":\"` + email + `\",\"username\": \"` + username + `\",\"password\": \"` + password + `\",\"time\":\"` + fmt.Sprintf("%d", time.Now().Unix()) + `\"}"`
	} else {
		jsonAnswerShould = `{"tok":"` + tok + `","email":"` + email + `","password": "` + password + `","time":"` + fmt.Sprintf("%d", time.Now().Unix()) + `"}`
		jsonAnswer = `"{\"tok\":\"` + tok + `\",\"email\":\"` + email + `\",\"password\": \"` + password + `\",\"time\":\"` + fmt.Sprintf("%d", time.Now().Unix()) + `\"}"`
	}
	summed := sha256.Sum256([]byte(jsonAnswerShould))
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
		Path:     "/" + endpoint,
		MaxAge:   120,
	})
	w.WriteHeader(http.StatusAccepted)
}
