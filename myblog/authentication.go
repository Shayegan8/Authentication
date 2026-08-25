package myblog

import (
	"context"
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
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
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
		if token == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		dip := r.Header.Get("realip")

		removed, err := Redis_client.LRem(r.Context(), dip, 1, token).Result()

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		} else if removed == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		email := payload.Get("email")
		parts := strings.Split(email, "@")
		if len(parts) != 1 {
			if len(parts[0]) < 1 || !strings.Contains(parts[1], ".") {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad request"))
				return
			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		res := CaptchaGeneration(email, email, email, email, dip, w, r)
		if res {
			return
		}
		captchaD := payload.Get("captchaAnswer")
		var captchaData map[string]string
		if captchaD != "" {
			err := json.Unmarshal([]byte(captchaD), &captchaData)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("It should be all string"))
				return
			}
			CaptchaToken(captchaData, w, dip, r)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		sig := payload.Get("signature")
		tok := payload.Get("jwtAnswer")
		if sig == "" || tok == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		summedJwt := sha256.Sum256([]byte(tok))
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
		erro := json.Unmarshal([]byte(tok), &marshaled)
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

		rows, erra := Postgres_client.Query(context.Background(), "SELECT * FROM users WHERE email=$1", email)
		if erra != nil {
			rows.Close()
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error")) // Invalid user credintals
			return
		}

		if rows.Next() { // this means if the user with this details actually exist
			rows.Close()
			newHashedPass := make([]byte, 32)
			rand.Read(newHashedPass)
			encodedHashedPass := hex.EncodeToString(newHashedPass)
			_, err := Postgres_client.Exec(r.Context(), "UPDATE users SET password=$1 WHERE email=$2", newHashedPass, email)
			if err != nil {
				rows.Close()
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Server error")) // Invalid user credintals
				return
			}
			msg := []byte("To: " + email + "\r\n" +
				"Subject: Shayegan's blog\r\n" +
				"\r\n" +
				"Heres your new password " + encodedHashedPass + ".\r\n")
			err1 := smtp.SendMail("smtp.gmail.com:587", Auth, Config["user"], []string{email}, []byte(msg))
			if err1 != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Problem with server"))
				return
			}
			w.WriteHeader(http.StatusAccepted)
			return
		} else {
			rows.Close()
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request")) // Invalid user credintals
			return
		}
	}
}

func Login(w http.ResponseWriter, r *http.Request) {
	payload := r.Header
	switch r.Method {
	case "GET":
		BucketHandlement(w, r)
	case "POST":
		token := payload.Get("token") // generated token from bucket
		if token == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		dip := r.Header.Get("realip")

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

		verification := payload.Get("verification")
		password := payload.Get("password")
		email := payload.Get("email") // for login email field can be either username or email
		res := CaptchaGeneration(verification, email, password, email, dip, w, r)
		if res {
			return
		}
		captchaD := payload.Get("captchaAnswer")
		var captchaData map[string]string
		if captchaD != "" {
			err := json.Unmarshal([]byte(captchaD), &captchaData)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("It should be all string"))
				return
			}
			CaptchaToken(captchaData, w, dip, r)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		sig := payload.Get("signature")
		tok := payload.Get("jwtAnswer")
		if sig == "" || tok == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		summedJwt := sha256.Sum256([]byte(tok))
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
		erro := json.Unmarshal([]byte(tok), &marshaled)
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

		if email == "" || password == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		var rows pgx.Rows
		var erra error
		if strings.Contains(email, "@") {
			parts := strings.Split(email, "@")
			if len(parts[0]) < 1 || !strings.Contains(parts[1], ".") {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad request"))
				return
			}
			rows, erra = Postgres_client.Query(context.Background(), "SELECT password FROM users WHERE email=$1", email)
		} else {
			rows, erra = Postgres_client.Query(context.Background(), "SELECT password FROM users WHERE username=$1", email)
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
		Verify(verification, password, email, dip, w, r)
		Authing(email, email, password, verification, dip, w, r, false)
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
			return
		} else if removed == 0 {
			log.Println("Removed problem")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		verification := payload.Get("verification")
		username := payload.Get("username")
		password := payload.Get("password")
		email := payload.Get("email")

		res := CaptchaGeneration(verification, username, password, email, dip, w, r)
		if res {
			return
		}
		captchaD := payload.Get("captchaAnswer")
		var captchaData map[string]string
		if captchaD != "" {
			err := json.Unmarshal([]byte(captchaD), &captchaData)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("It should be all string"))
				return
			}
			CaptchaToken(captchaData, w, dip, r)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		sig := payload.Get("signature")
		tok := payload.Get("jwtAnswer")
		if sig == "" || tok == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		summedJwt := sha256.Sum256([]byte(tok))
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
		erro := json.Unmarshal([]byte(tok), &marshaled)
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

		parts := strings.Split(email, "@")
		if len(parts) != 1 {
			if len(parts[0]) < 1 || !strings.Contains(parts[1], ".") {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad request"))
				return
			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		rows, erra := Postgres_client.Query(context.Background(), "SELECT * FROM users WHERE email=$1 OR username=$2", email, username)
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

		Verify(verification, password, email, dip, w, r)
		Authing(username, email, password, verification, dip, w, r, true)
	}
}

func CaptchaGeneration(verification string, username string, password string, email string, dip string, w http.ResponseWriter, r *http.Request) bool {
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
			return true
		}

		masterImage, _ := captData.GetMasterImage().ToBase64()
		tileImage, _ := captData.GetTileImage().ToBase64()
		jsoned := `{
			"masterImage": "` + masterImage + `",
			"titleImage": "` + tileImage + `"
		}`
		// we should have store the answer in some storage,
		log.Println("This is the answer, X:", captData.GetData().X, ", Y:", captData.GetData().Y)
		Redis_client.Set(r.Context(), "captcha"+dip+fmt.Sprintf("%d,%d", captData.GetData().X, captData.GetData().Y), fmt.Sprintf("%d,%d", captData.GetData().X, captData.GetData().Y), 1*time.Minute)
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(jsoned))
		return true
	}
	return false
}

func CaptchaToken(captchaData map[string]string, w http.ResponseWriter, dip string, r *http.Request) {
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
				"time":"` + fmt.Sprintf("%d", time.Now().Unix()) + `"
			}`
		summed := sha256.Sum256([]byte(jsonAnswer))
		signature, _ := rsa.SignPKCS1v15(rand.Reader, PrivateKey, crypto.SHA256, summed[:])
		w.WriteHeader(http.StatusAccepted)
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

func Verify(verification string, password string, email string, dip string, w http.ResponseWriter, r *http.Request) {
	if verification == "" {
		if len(password) < 8 { // we just check length of password
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
		err := smtp.SendMail("smtp.gmail.com:587", Auth, Config["user"], []string{email}, []byte(msg))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Problem with sending email to you happened in server"))
			return
		}
		Redis_client.Set(r.Context(), "captcha"+dip, vcode, 1*time.Minute)
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, "%d", vcode)
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad request"))
		return
	}
}

func Authing(username string, email string, password string, verification string, dip string, w http.ResponseWriter, r *http.Request, registery bool) {
	if vercode, erro := strconv.Atoi(verification); erro != nil {
		// in this case user received the code and its on the header now
		vc, err := Redis_client.GetDel(r.Context(), "captcha"+dip).Result()
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
			query, err := Postgres_client.Query(context.Background(),
				"INSERT INTO users(email, username, password, refreshToken)"+
					" VALUES ($1, $2, $3, $4) RETURNING userid", email, username, hashedPassword, refreshTokenHex)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Server error"))
			}
			if query.Next() {
				var userid string
				query.Scan(&userid)
				response := `{
					"userid": "` + userid + `",
					"refreshToken": "` + refreshTokenHex + `"
				}`
				// give token and write success
				w.WriteHeader(http.StatusAccepted)
				w.Write([]byte(response))
			}
		} else {
			var rows pgx.Rows
			var erra error
			refreshToken := make([]byte, 32)
			rand.Read(refreshToken)
			refreshTokenHex := hex.EncodeToString(refreshToken)
			if strings.Contains(email, "@") {
				rows, erra = Postgres_client.Query(context.Background(), "UPDATE users SET refreshToken = $1 WHERE email=$2 RETURNING refreshToken, userid, password", refreshTokenHex, email)
			} else {
				rows, erra = Postgres_client.Query(context.Background(), "UPDATE users SET refreshToken = $1 WHERE username=$2 RETURNING refreshToken, userid, password", refreshTokenHex, username)
			}
			if erra != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Server error"))
				return
			}
			if rows.Next() {
				var refreshTokenHex string
				var userid string
				var hashedPassword []byte

				err1 := rows.Scan(&refreshTokenHex, &userid, &hashedPassword)
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
				response := `{
					"userid": "` + userid + `,
					"refreshToken": "` + refreshTokenHex + `"
				}`

				w.WriteHeader(http.StatusAccepted)
				w.Write([]byte(response))
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
