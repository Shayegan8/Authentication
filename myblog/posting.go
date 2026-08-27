package myblog

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

func Post(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		BucketHandlement("post", "post", w, r)
	case "POST":
		payload := r.Header
		cookie, ero := r.Cookie("post")
		userCSRF := payload.Get("csrf-Token")
		title := payload.Get("title")
		info := payload.Get("info")
		body := payload.Get("body")

		if userCSRF == "" || info == "" || title == "" || body == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		if ero != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		parts := strings.Split(cookie.Value, ",")
		token := parts[0]
		csrf := parts[1]
		if token == "" || csrf == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if userCSRF != csrf {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		email := ""

		if Validator(email, w, r) {
			return
		}

		if email == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		removed, ero := Redis_client.LRem(r.Context(), "post"+email, 1, token).Result()

		if ero != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		} else if removed == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		// titles are unique so if we insert with a title that exist we will get error created_at and postid is auto generated
		_, ero = Postgres_client.Exec(r.Context(), "INSERT INTO posts(title, info, body) VALUES ($1, $2, $3)", title, info, body)
		if ero != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

type PostData struct {
	Title string `json:"title"`
	Info  string `json:"info"`
}

func GetPosts(w http.ResponseWriter, r *http.Request) { // GetPosts dosent require refresh tokens
	switch r.Method {
	case "GET":
		BucketHandlement("getPosts", "getPosts", w, r)
	case "POST":
		payload := r.Header
		page := payload.Get("page")
		if page == "1" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		dip := payload.Get("realip")
		cookie, ero := r.Cookie("getPosts")
		userCSRF := payload.Get("csrf-Token")
		if userCSRF == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		if ero != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		parts := strings.Split(cookie.Value, ",")
		token := parts[0]
		csrf := parts[1]
		if token == "" || csrf == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if userCSRF != csrf {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		email := ""

		if Validator(email, w, r) {
			return
		}

		var removed int64
		var erro error

		if email == "" {
			removed, erro = Redis_client.LRem(r.Context(), dip+token, 1, token).Result()
		} else {
			removed, erro = Redis_client.LRem(r.Context(), "getPosts"+email, 1, token).Result()
		}

		if erro != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		} else if removed == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		var rows pgx.Rows
		var e error
		if page == "" {
			rows, e = Postgres_client.Query(r.Context(), "SELECT * FROM posts ORDER BY created_at DESC LIMIT 10")
		} else {
			numberPage, e := strconv.Atoi(page)
			if e != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Bad request"))
				return
			} else if numberPage < 0 {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Bad request"))
				return
			}
			rows, e = Postgres_client.Query(r.Context(), "SELECT * FROM posts OFFSET $1 ORDER BY created_at DESC LIMIT 10", numberPage*10)
		}
		if e != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Bad request"))
			return
		}

		posts := make([]PostData, 10)
		for i := 0; rows.Next(); i++ {
			rows.Scan(&posts[0].Title, &posts[0].Info)
		}

		jsoni := make(map[int]any, 10)
		for i := range 10 {
			jsoni[i] = PostData{Title: posts[i].Title, Info: posts[i].Info}
		}
		jsonResponse, eee := json.Marshal(jsoni)
		if eee != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Bad request"))
			return
		}
		w.WriteHeader(http.StatusAccepted)
		w.Write(jsonResponse)
	}
}
