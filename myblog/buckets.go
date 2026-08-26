package myblog

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
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

func BucketHandlement(name string, endpoint string, w http.ResponseWriter, r *http.Request) {
	dip := r.Header.Get("realip")
	_, err := r.Cookie(name)
	if err != nil { // means such a cookie not exist
		l("shit1")
		counter, err := Redis_client.Incr(r.Context(), name+dip).Result()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}
		l("Whats the counter:", counter)
		if counter == 1 {
			Redis_client.Expire(r.Context(), name+dip, 30*time.Second)
		}

		redis_pipe := Redis_client.Pipeline()
		buckdat := GenerateBucket(rnd.IntN(50) + 20)
		buckdatInterfaces := make([]any, len(buckdat))
		for k, v := range buckdat {
			buckdatInterfaces[k] = v
		}
		sessionId := rnd.IntN(90000) + 10000
		redis_pipe.LPush(r.Context(), dip+fmt.Sprintf("%d", sessionId), buckdatInterfaces...)
		redis_pipe.Expire(r.Context(), dip+fmt.Sprintf("%d", sessionId), 30*time.Second)
		redis_pipe.Exec(r.Context())
		token, err := Redis_client.LIndex(r.Context(), dip, -1).Result()
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    token,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			Path:     "/auth/" + endpoint,
			MaxAge:   30,
		})
		w.WriteHeader(http.StatusAccepted)
		return
	} else {
		counter, err := Redis_client.Incr(r.Context(), name+dip).Result()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}

		if counter > 10 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		result, err := Redis_client.LIndex(r.Context(), dip, -1).Result()
		length, err := Redis_client.LLen(r.Context(), dip).Result()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}
		if length == 0 {
			pipeline := Redis_client.Pipeline()
			pipeline.LPush(r.Context(), dip+result, "in-queue")
			pipeline.Expire(r.Context(), dip+result, 5*time.Minute)
			pipeline.Exec(r.Context())
		}
		if result == "in-queue" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    result,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			Path:     "/auth/" + endpoint,
			MaxAge:   30,
		})
		w.WriteHeader(http.StatusAccepted)
	}
}
