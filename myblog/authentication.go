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
		BucketHandlement(w, r)
	case "POST":
		token := payload.Get("token") // generated token from bucket
		sig := payload.Get("signature")
		answer := payload.Get("answer")
		dip := payload.Get("realip")

		if token == "" || sig == "" || answer == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		removed, err := Redis_client.LRem(r.Context(), dip, 1, token).Result()

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
		} else if removed == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
		} else {
			CaptchaGeneration(dip, w, r)
		}
	}
}

func ForgetPasswordValidation(w http.ResponseWriter, r *http.Request) {
	payload := r.Header
	dip := r.Header.Get("realip")
	switch r.Method {
	case "GET":
		BucketHandlement(w, r)
	case "POST":
		email := payload.Get("email")
		token := payload.Get("token")
		sig := payload.Get("signature")
		answer := payload.Get("answer")
		if email == "" || token == "" || answer == "" || sig == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		removed, erro := Redis_client.LRem(r.Context(), dip, 1, token).Result()

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

		if (time.Now().Unix() - int64(converted)) > 300 {
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
		CaptchaToken(captchaData, email, w, dip, r)
	}
}

func ForgetPasswordValidationJWT(w http.ResponseWriter, r *http.Request) {
	payload := r.Header
	switch r.Method {
	case "GET":
		BucketHandlement(w, r)
	case "POST":
		dip := payload.Get("realip")
		sig := payload.Get("signature")
		answer := payload.Get("answer") // the same we gave them they should put it there
		token := payload.Get("token")
		if sig == "" || answer == "" || token == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		removed, err := Redis_client.LRem(r.Context(), dip, 1, token).Result()

		if err != nil { // dont care if its server error or client bad request
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		} else if removed == 0 {
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

		if (time.Now().Unix() - int64(converted)) > 300 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		rows, erra := Postgres_client.Query(r.Context(), "SELECT * FROM users WHERE email=$1", marshaled["email"])
		if erra != nil {
			if erra == redis.Nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Bad request")) // Invalid user credintals
				return
			}
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
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request")) // Invalid user credintals
		}
	}
}

func rateLimiter(dip string, w http.ResponseWriter, r *http.Request) bool {
	counter, err := Redis_client.Incr(r.Context(), "counter"+dip).Result()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server error"))
		return false
	}
	l("Whats the counter:", counter)
	if counter == 1 {
		Redis_client.Expire(r.Context(), "counter"+dip, 30*time.Second)
	}

	if counter >= 10 {
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
		if !rateLimiter(dip, w, r) {
			return
		}
		email, err := Redis_client.LPop(r.Context(), vars["token"]).Result()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
		} else if email == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
		} else {
			tokBuffer := make([]byte, 16)
			rand.Read(tokBuffer)
			jsonAnswer := `{
				"tok":"` + hex.EncodeToString(tokBuffer) + `",
				"email": "` + email + `",
				"time": "` + fmt.Sprintf("%d", time.Now().Unix()) + `"
			}`
			summed := sha256.Sum256([]byte(jsonAnswer))
			signature, _ := rsa.SignPKCS1v15(rand.Reader, PrivateKey, crypto.SHA256, summed[:])
			w.WriteHeader(http.StatusAccepted)
			w.Write([]byte(`{
				"answer": ` + jsonAnswer + `,
				"signature": "` + base64.StdEncoding.EncodeToString(signature) + `"
			}`))
		}
	case "POST":
		if !rateLimiter(dip, w, r) {
			return
		}
		password := payload.Get("password")
		sig := payload.Get("signature")
		answer := payload.Get("answer")
		if password == "" || sig == "" || answer == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if strings.ContainsAny(password, "@.\"'") {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if len(password) < 8 || !strings.ContainsAny(password, "ABCDEFGHIKJLMNOPQRSTUVWXYZ") || !strings.ContainsAny(password, "abcdefghikjlmnopqrstuvwxyz") || !strings.ContainsAny(password, "0123456789") || !strings.ContainsAny(password, "!@#$%^&*()-_+") {
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

		var answerMap map[string]string
		json.Unmarshal([]byte(answer), &answerMap)
		converted, _ := strconv.Atoi(answerMap["time"])

		if (time.Now().Unix() - int64(converted)) > 300 {
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

		oo, er := Postgres_client.Exec(r.Context(), "UPDATE users SET password=$1 WHERE email=$2", hashedPassword, answerMap["email"])
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
		BucketHandlement(w, r)
	case "POST":
		sig := payload.Get("signature")
		answer := payload.Get("answer")
		password := payload.Get("password")
		verification := payload.Get("verification")
		token := payload.Get("token")

		if sig == "" || answer == "" || password == "" || verification == "" || token == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		removed, err := Redis_client.LRem(r.Context(), dip, 1, token).Result()

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

		if len(password) < 8 || !strings.ContainsAny(password, "ABCDEFGHIKJLMNOPQRSTUVWXYZ") || !strings.ContainsAny(password, "abcdefghikjlmnopqrstuvwxyz") || !strings.ContainsAny(password, "0123456789") || !strings.ContainsAny(password, "!@#$%^&*()-_+") {
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

		if (time.Now().Unix() - int64(converted)) > 300 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if vercode, erro := strconv.Atoi(verification); erro == nil {
			// in this case user received the code and its on the header now
			vc, err := Redis_client.Get(r.Context(), "captcha"+dip).Result()
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
			if verificationCode != vercode {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad request"))
				return
			} else {
				vcc, errr := Redis_client.Del(r.Context(), "captcha"+dip).Result()
				if errr != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Server error")) // i dont think this happens, anyway
					return
				} else if vcc == 0 {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte("Bad request"))
					return
				}
			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		rows, errio := Postgres_client.Query(r.Context(), "SELECT refreshToken, userid, password FROM users WHERE email=$1", marshaled["email"])
		if errio != nil {
			if errio == redis.Nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad request"))
				return
			}
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

			_, erra := Postgres_client.Exec(r.Context(), "UPDATE users SET refreshToken = $1 WHERE email=$2 RETURNING refreshToken, userid, password", refreshTokenHex, marshaled["email"])
			if erra != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Server error"))
				return
			}

			response := `{
					"userid": "` + userid + `,
					"refreshToken": "` + refreshTokenHex + `"
				}`

			w.WriteHeader(http.StatusAccepted)
			w.Header().Set("content-type", "application/json")
			w.Write([]byte(response))
			return
		} else {
			rows.Close()
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Bad request"))
			return
		}
	}
}

func LoginValidationJWT(w http.ResponseWriter, r *http.Request) {
	payload := r.Header
	dip := payload.Get("realip")
	switch r.Method {
	case "GET":
		BucketHandlement(w, r)
	case "POST":
		sig := payload.Get("signature")
		answer := payload.Get("answer")
		password := payload.Get("password")
		token := payload.Get("token")

		if sig == "" || answer == "" || password == "" || token == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		removed, err := Redis_client.LRem(r.Context(), dip, 1, token).Result()

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

		if len(password) < 8 || !strings.ContainsAny(password, "ABCDEFGHIKJLMNOPQRSTUVWXYZ") || !strings.ContainsAny(password, "abcdefghikjlmnopqrstuvwxyz") || !strings.ContainsAny(password, "0123456789") || !strings.ContainsAny(password, "!@#$%^&*()-_+") {
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

		if (time.Now().Unix() - int64(converted)) > 300 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		rows, erra := Postgres_client.Query(r.Context(), "SELECT password FROM users WHERE email=$1", marshaled["email"])

		if erra != nil {
			if erra == redis.Nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad request")) // Invalid user credintals
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error")) // Invalid user credintals
			return
		}
		if rows.Next() {
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
		Verify(marshaled["email"], dip, w, r)
	}
}

func LoginValidation(w http.ResponseWriter, r *http.Request) {
	payload := r.Header
	switch r.Method {
	case "GET":
		BucketHandlement(w, r)
	case "POST":
		email := payload.Get("email")
		captchaD := payload.Get("captchaAnswer")
		sig := payload.Get("signature")
		answer := payload.Get("answer")
		dip := payload.Get("realip")
		token := payload.Get("token")

		if email == "" || captchaD == "" || sig == "" || answer == "" || token == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		removed, erro := Redis_client.LRem(r.Context(), dip, 1, token).Result()

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

		if (time.Now().Unix() - int64(converted)) > 300 {
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
		CaptchaToken(captchaData, email, w, dip, r)
	}
}

func Login(w http.ResponseWriter, r *http.Request) {
	payload := r.Header
	switch r.Method {
	case "GET":
		BucketHandlement(w, r)
	case "POST":
		dip := r.Header.Get("realip")
		token := payload.Get("token") // generated token from bucket
		if token == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		removed, err := Redis_client.LRem(r.Context(), dip, 1, token).Result()

		if err != nil { // dont care if its server error or client bad request
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		} else if removed == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		CaptchaGeneration(dip, w, r)
	}
}

func RegisterValidationSubmit(w http.ResponseWriter, r *http.Request) {
	payload := r.Header
	switch r.Method {
	case "GET":
		BucketHandlement(w, r)
	case "POST":
		token := payload.Get("token") // generated token from bucket
		sig := payload.Get("signature")
		answer := payload.Get("answer")
		verification := payload.Get("verification")
		dip := payload.Get("realip")
		password := payload.Get("password")
		username := payload.Get("username")
		if verification == "" || username == "" || password == "" || answer == "" || sig == "" || token == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		removed, err := Redis_client.LRem(r.Context(), dip, 1, token).Result()

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

		if len(password) < 8 || !strings.ContainsAny(password, "ABCDEFGHIKJLMNOPQRSTUVWXYZ") || !strings.ContainsAny(password, "abcdefghikjlmnopqrstuvwxyz") || !strings.ContainsAny(password, "0123456789") || !strings.ContainsAny(password, "!@#$%^&*()-_+") {
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

		if (time.Now().Unix() - int64(converted)) > 300 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if vercode, erro := strconv.Atoi(verification); erro == nil {
			// in this case user received the code and its on the header now
			vc, err := Redis_client.Get(r.Context(), "captcha"+dip).Result()
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
			if verificationCode != vercode {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad request"))
				return
			} else {
				vcc, errr := Redis_client.Del(r.Context(), "captcha"+dip).Result()
				if errr != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Server error")) // i dont think this happens, anyway
					return
				} else if vcc == 0 {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte("Bad request"))
					return
				}
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
				" VALUES ($1, $2, $3, $4) RETURNING userid", marshaled["email"], username, hashedPassword, refreshTokenHex)
		if err != nil {
			// this isnt possible but anyway
			if err == redis.Nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Bad request"))
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
		}
		if query.Next() {
			var userid string
			query.Scan(&userid)
			query.Close()
			response := `{
					"userid": "` + userid + `",
					"refreshToken": "` + refreshTokenHex + `"
				}`
			// give token and write success
			w.WriteHeader(http.StatusAccepted)
			w.Header().Set("content-type", "application/json")
			w.Write([]byte(response))
		}
	}
}

func RegisterValidationJWT(w http.ResponseWriter, r *http.Request) {
	payload := r.Header
	switch r.Method {
	case "GET":
		BucketHandlement(w, r)
	case "POST":
		token := payload.Get("token")
		sig := payload.Get("signature")
		answer := payload.Get("answer")
		dip := payload.Get("realip")
		username := payload.Get("username")
		password := payload.Get("password")
		if sig == "" || answer == "" || username == "" || password == "" || token == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		removed, err := Redis_client.LRem(r.Context(), dip, 1, token).Result()

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

		if len(password) < 8 || !strings.ContainsAny(password, "ABCDEFGHIKJLMNOPQRSTUVWXYZ") || !strings.ContainsAny(password, "abcdefghikjlmnopqrstuvwxyz") || !strings.ContainsAny(password, "0123456789") || !strings.ContainsAny(password, "!@#$%^&*()-_+") {
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

		if (time.Now().Unix() - int64(converted)) > 300 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		rows, erra := Postgres_client.Query(r.Context(), "SELECT * FROM users WHERE email=$1 OR username=$2", marshaled["email"], username)
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

		Verify(marshaled["email"], dip, w, r)

	}
}

func RegisterValidation(w http.ResponseWriter, r *http.Request) {
	payload := r.Header
	switch r.Method {
	case "GET":
		BucketHandlement(w, r)
	case "POST":
		captchaD := payload.Get("captchaAnswer")
		sig := payload.Get("signature")
		answer := payload.Get("answer")
		email := payload.Get("email")
		dip := payload.Get("realip")
		token := payload.Get("token")
		if captchaD == "" || sig == "" || answer == "" || email == "" || token == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
		}

		removed, erro := Redis_client.LRem(r.Context(), dip, 1, token).Result()

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

		if (time.Now().Unix() - int64(converted)) > 300 {
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
		CaptchaToken(captchaData, email, w, dip, r)
	}
}

func Register(w http.ResponseWriter, r *http.Request) {
	payload := r.Header
	switch r.Method {
	case "GET":
		BucketHandlement(w, r)
	case "POST":
		token := payload.Get("token") // generated token from bucket
		log.Println("Token seems obtained")
		if token == "" {
			log.Println("What the fuck?")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		dip := r.Header.Get("realip")
		log.Println("Ip of that guy", dip)

		removed, err := Redis_client.LRem(r.Context(), dip, 1, token).Result()

		if err != nil { // dont care if its server error or client bad request
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
		} else if removed == 0 {
			log.Println("Removed problem")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
		} else {
			CaptchaGeneration(dip, w, r)
		}
	}
}

func CaptchaGeneration(dip string, w http.ResponseWriter, r *http.Request) {
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
	answer := `{
				"token": "` + buffTokHex + `",
				"time": "` + fmt.Sprintf("%d", time.Now().Unix()) + `"
			}`
	summedAnswer := sha256.Sum256([]byte(answer))
	signature, erri := rsa.SignPKCS1v15(rand.Reader, PrivateKey, crypto.SHA256, summedAnswer[:])
	if erri != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server error"))
		return
	}

	jsoned := `{
			"masterImage": "` + masterImage + `",
			"titleImage": "` + tileImage + `",
			"signature": "` + base64.StdEncoding.EncodeToString(signature) + `",
			"answer": ` + answer + `
		}`
	// we should have store the answer in some storage,
	log.Println("This is the answer, X:", captData.GetData().X, ", Y:", captData.GetData().Y)
	Redis_client.Set(r.Context(), "captcha"+dip+fmt.Sprintf("%d,%d", captData.GetData().X, captData.GetData().Y), fmt.Sprintf("%d,%d", captData.GetData().X, captData.GetData().Y), 1*time.Minute)
	w.WriteHeader(http.StatusAccepted)
	w.Header().Set("content-type", "application/json")
	w.Write([]byte(jsoned))
}

func CaptchaToken(captchaData map[string]string, email string, w http.ResponseWriter, dip string, r *http.Request) {
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

	str, err := Redis_client.GetDel(r.Context(), "captcha"+dip+fmt.Sprintf("%d,%d", x, y)).Result()
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
		jsonAnswer := `{
				"tok":"` + tok + `",
				"email":"` + email + `",
				"time":"` + fmt.Sprintf("%d", time.Now().Unix()) + `"
			}`
		summed := sha256.Sum256([]byte(jsonAnswer))
		signature, _ := rsa.SignPKCS1v15(rand.Reader, PrivateKey, crypto.SHA256, summed[:])
		w.WriteHeader(http.StatusAccepted)
		w.Header().Set("content-type", "application/json")
		w.Write([]byte(`{
				"answer": ` + jsonAnswer + `,
				"signature": "` + base64.StdEncoding.EncodeToString(signature) + `"
			}`))
		return
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad request"))
		return
	}
}

func Verify(email string, dip string, w http.ResponseWriter, r *http.Request) {
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
	Redis_client.Set(r.Context(), "captcha"+dip, vcode, 3*time.Minute)
	w.WriteHeader(http.StatusAccepted)
}
