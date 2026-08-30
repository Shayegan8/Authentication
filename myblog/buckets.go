package myblog

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"log"
	rnd "math/rand/v2"
	"net/http"
	"strings"
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

// protected by WAF
func GetRandomToken(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		randomShit := make([]byte, 16)
		rand.Read(randomShit)
		hexed := hex.EncodeToString(randomShit)
		summed := sha256.Sum256([]byte(hexed))
		signature, _ := rsa.SignPKCS1v15(rand.Reader, PrivateKey, crypto.SHA256, summed[:])
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(hex.EncodeToString(signature) + "," + hexed))
	}
}

func BucketHandlement(name string, endpoint string, w http.ResponseWriter, r *http.Request) {
	dip := r.Header.Get("realip")
	sideshash := r.Header.Get("sideline") // signature,hexed
	if sideshash == "" {
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("Bad request"))
		return
	}
	portions := strings.Split(sideshash, ",")
	signatureForSideline := portions[0]
	sideline := portions[1]
	hashed := sha256.Sum256([]byte(sideline))
	err := rsa.VerifyPKCS1v15(PublicKey, crypto.SHA256, hashed[:], []byte(signatureForSideline))
	if err != nil {
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("Bad request"))
		return
	}
	exists := false
	count, o := Redis_client.Exists(r.Context(), name+dip+sideline).Result()
	if count != 0 {
		exists = true
	}

	if o != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server error"))
		return
	}

	if !exists { // means such a cookie not exist
		l("shit1")

		counter, _ := Redis_client.Incr(r.Context(), "counter"+name+dip+sideline).Result()
		Redis_client.Expire(r.Context(), "counter"+name+dip+sideline, 30*time.Second)

		l("Whats the counter:", counter)
		if counter == 1 {
			Redis_client.Expire(r.Context(), name+dip+sideline, 30*time.Second)
		}

		buckdat := GenerateBucket(rnd.IntN(50) + 20)
		buckdatInterfaces := make([]any, len(buckdat))
		for k, v := range buckdat {
			buckdatInterfaces[k] = v
		}
		Redis_client.LPush(r.Context(), name+dip+sideline, buckdatInterfaces...)
		Redis_client.Expire(r.Context(), name+dip+sideline, 30*time.Second)

		token, _ := Redis_client.LIndex(r.Context(), name+dip+sideline, -1).Result()
		randomShit := make([]byte, 16)
		rand.Read(randomShit)
		hexed := hex.EncodeToString(randomShit)
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    token + "," + hexed + "," + sideline,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			Path:     "/auth/" + endpoint,
			MaxAge:   30,
		})
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(hexed))
		return
	} else {

		counter, _ := Redis_client.Incr(r.Context(), "counter"+name+dip+sideline).Result()
		Redis_client.Expire(r.Context(), "counter"+name+dip+sideline, 30*time.Second)

		if counter > 10 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		result, err := Redis_client.LIndex(r.Context(), name+dip+sideline, -1).Result()
		length, err := Redis_client.LLen(r.Context(), name+dip+sideline).Result()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}
		if length == 0 {
			Redis_client.LPush(r.Context(), name+dip+sideline, "in-queue")
			Redis_client.Expire(r.Context(), name+dip+sideline, 30*time.Second)
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
			Value:    result + "," + hexed + "," + sideline,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			Path:     "/auth/" + endpoint,
			MaxAge:   30,
		})
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(hexed))
	}
}
