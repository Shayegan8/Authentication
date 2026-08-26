package myblog

import "net/http"

func Post(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		BucketHandlement("post", "post", w, r)
	case "POST":
		payload := r.Header
		dip := payload.Get("realip")
		cookie, ero := r.Cookie("post")
		if ero != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		token := cookie.Value
		removed, erro := Redis_client.LRem(r.Context(), dip+token, 1, token).Result()

		if erro != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		} else if removed == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		/*
			we check for their refresh token
			then we make a signed access token
			then they can use that access token
		*/
	}
}

func GetPosts(w http.ResponseWriter, r *http.Request) { // GetPosts dosent require
	switch r.Method {
	case "GET":
		BucketHandlement("getPosts", "getPosts", w, r)
	case "POST":
		payload := r.Header
		dip := payload.Get("realip")
		cookie, ero := r.Cookie("post")
		if ero != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		token := cookie.Value
		removed, erro := Redis_client.LRem(r.Context(), dip+token, 1, token).Result()

		if erro != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		} else if removed == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

	}
}
