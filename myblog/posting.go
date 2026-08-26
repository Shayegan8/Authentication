package myblog

import "net/http"

func Post(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		BucketHandlement("post", "post", w, r)
	case "POST":
		cookie, ero := r.Cookie("post")
		if ero != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		token := cookie.Value

	}
}

func GetPosts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		BucketHandlement("getPosts", "getPosts", w, r)
	case "POST":
		cookie, ero := r.Cookie("forgetPassword")
		if ero != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		token := cookie.Value

	}
}
