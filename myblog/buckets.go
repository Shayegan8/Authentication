package myblog

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	rnd "math/rand/v2"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
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

func BucketHandlement(w http.ResponseWriter, r *http.Request) {
	dip := r.Header.Get("realip")
	token, err := Redis_client.LIndex(r.Context(), dip, -1).Result()
	l("shit", token)
	if err == redis.Nil {
		l("shit1")
		counter, err := Redis_client.Incr(r.Context(), "counter"+dip).Result()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}
		l("Whats the counter:", counter)
		if counter == 1 {
			Redis_client.Expire(r.Context(), "counter"+dip, 30*time.Second)
		}

		redis_pipe := Redis_client.Pipeline()
		buckdat := GenerateBucket(rnd.IntN(50) + 20)
		buckdatInterfaces := make([]any, len(buckdat))
		for k, v := range buckdat {
			buckdatInterfaces[k] = v
		}
		redis_pipe.LPush(r.Context(), dip, buckdatInterfaces...)
		redis_pipe.Expire(r.Context(), dip, 5*time.Minute)
		redis_pipe.Exec(r.Context())
		token, err := Redis_client.LIndex(r.Context(), dip, -1).Result()
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

		counter, err := Redis_client.Incr(r.Context(), "counter"+dip).Result()
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

		length, err := Redis_client.LLen(r.Context(), dip).Result()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}
		if length == 0 {
			Redis_client.LPush(r.Context(), dip, "in-queue")
		}

	}
}
