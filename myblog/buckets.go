package myblog

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"log"
	rnd "math/rand/v2"
	"net/http"
	"time"
)

type BucketData struct {
	tokens    []string
	timestamp int64
}

func GenerateBucket(n int) []string {
	var arr []string = make([]string, n)
	for i := range len(arr) {
		var buffer []byte = make([]byte, 10)
		rand.Read(buffer)
		arr[i] = hex.EncodeToString(buffer)
	}
	return arr
}

var l = log.Println

/*
Ok we rate limit better
If just userData exist we do rate limit with userData, using userAgent make it more vulnerable
*/
func BucketHandlement(name string, endpoint string, w http.ResponseWriter, r *http.Request) {
	dip := r.Header.Get("realip")
	userData, erro := r.Cookie("userData")
	email := ""
	if erro == nil {
		decrypted, ere := base64.StdEncoding.DecodeString(userData.Value)
		if ere != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}
		var decryptedInMap map[string]string
		erse := json.Unmarshal(decrypted, &decryptedInMap)
		if erse != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}

		sig := decryptedInMap["signature"]
		answer := decryptedInMap["answer"]
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

		// then we check if the refresh token was exist
		rows, e := Postgres_client.Query(r.Context(), "SELECT refreshToken FROM users WHERE email=$1", marshaled["email"])
		if e != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}
		var refreshTokenBuffer []byte
		if rows.Next() {
			rows.Scan(&refreshTokenBuffer)
			refreshToken := hex.EncodeToString(refreshTokenBuffer)
			if refreshToken != marshaled["refreshToken"] {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad request"))
				return
			}
			email = marshaled["email"]
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

	}
	exists := false
	var count int64
	var o error
	if email == "" {
		count, o = Redis_client.Exists(r.Context(), name+dip).Result()
		if count != 0 {
			exists = true
		}
	} else {
		count, o = Redis_client.Exists(r.Context(), email).Result()
		if count != 0 {
			exists = true
		}
	}
	if o != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server error"))
		return
	}

	if !exists { // means such a cookie not exist
		l("shit1")

		var counter int64

		if email == "" {
			counter, _ = Redis_client.Incr(r.Context(), "counter"+name+dip).Result()
			Redis_client.Expire(r.Context(), "counter"+name+dip, 30*time.Second)
		} else {
			counter, _ = Redis_client.Incr(r.Context(), "counter"+email).Result()
			Redis_client.Expire(r.Context(), "counter"+email, 30*time.Second)
		}
		l("Whats the counter:", counter)
		if counter == 1 {
			if email == "" {
				Redis_client.Expire(r.Context(), name+dip, 30*time.Second)
			} else {
				Redis_client.Expire(r.Context(), email, 30*time.Second)
			}
		}

		buckdat := GenerateBucket(rnd.IntN(50) + 20)
		buckdatInterfaces := make([]any, len(buckdat))
		for k, v := range buckdat {
			buckdatInterfaces[k] = v
		}
		if email == "" {
			Redis_client.LPush(r.Context(), name+dip, buckdatInterfaces...)
			Redis_client.Expire(r.Context(), name+dip, 30*time.Second)
		} else {
			Redis_client.LPush(r.Context(), email, buckdatInterfaces...)
			Redis_client.Expire(r.Context(), email, 30*time.Second)
		}

		token, _ := Redis_client.LIndex(r.Context(), name+dip, -1).Result()
		randomShit := make([]byte, 16)
		rand.Read(randomShit)
		hexed := hex.EncodeToString(randomShit)
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    token + "," + hexed,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			Path:     "/auth/" + endpoint,
			MaxAge:   30,
		})
		w.Write([]byte(hexed))
		w.WriteHeader(http.StatusAccepted)
		return
	} else {
		var counter int64

		if email == "" {
			counter, _ = Redis_client.Incr(r.Context(), "counter"+name+dip).Result()
			Redis_client.Expire(r.Context(), "counter"+name+dip, 30*time.Second)
		} else {
			counter, _ = Redis_client.Incr(r.Context(), "counter"+email).Result()
			Redis_client.Expire(r.Context(), "counter"+email, 30*time.Second)
		}

		if counter > 10 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		var result string
		var length int64
		var err error
		if email == "" {
			result, err = Redis_client.LIndex(r.Context(), name+dip, -1).Result()
			length, err = Redis_client.LLen(r.Context(), name+dip).Result()
		} else {
			result, err = Redis_client.LIndex(r.Context(), email, -1).Result()
			length, err = Redis_client.LLen(r.Context(), email).Result()
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}
		if length == 0 {
			if email == "" {
				Redis_client.LPush(r.Context(), name+dip, "in-queue")
				Redis_client.Expire(r.Context(), name+dip, 30*time.Second)
			} else {
				Redis_client.LPush(r.Context(), email, "in-queue")
				Redis_client.Expire(r.Context(), email, 30*time.Second)
			}
		}
		if result == "in-queue" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		randomShit := make([]byte, 16)
		rand.Read(randomShit)
		hexed := hex.EncodeToString(randomShit)
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    result + "," + hexed,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			Path:     "/auth/" + endpoint,
			MaxAge:   30,
		})
		w.Write([]byte(hexed))
		w.WriteHeader(http.StatusAccepted)
	}
}
