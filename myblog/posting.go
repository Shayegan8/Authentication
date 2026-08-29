package myblog

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
)

func Post(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		BucketHandlement("post", "post", w, r)
	case "POST":
		payload := r.Header
		dip := payload.Get("realip")
		cookie, ero := r.Cookie("post")
		userCSRF := payload.Get("csrf-token")
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
		sideline := parts[2]
		if token == "" || csrf == "" || sideline == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if userCSRF != csrf {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		removed, ero := Redis_client.LRem(r.Context(), "post"+dip+sideline, 1, token).Result()

		if ero != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		} else if removed == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

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
	Id    string `json:"postId"`
	Info  string `json:"info"`
	Body  string `json:"body"`
}

func GetPost(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		BucketHandlement("getPost", "post/v/", w, r)
	case "POST":
		payload := r.Header
		cookie, ero := r.Cookie("getPost")
		postid := mux.Vars(r)["postId"]
		dip := payload.Get("realip")
		if postid == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		userCSRF := payload.Get("csrf-token")
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
		sideline := parts[2]
		if token == "" || csrf == "" || sideline == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if userCSRF != csrf {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		removed, erro := Redis_client.LRem(r.Context(), "getPost"+dip+sideline, 1, token).Result()

		if erro != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		} else if removed == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		rows, e := Postgres_client.Query(r.Context(), "SELECT title, info, body FROM posts where postid=$1", postid)
		if e != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		post := PostData{}
		if rows.Next() {
			rows.Close()
			rows.Scan(&post.Title, &post.Info, &post.Body)
		} else {
			rows.Close()
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{
			"title": "` + post.Title + `",
			"info": "` + post.Info + `",
			"body": "` + post.Body + `",
		}`))
	}
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
		userCSRF := payload.Get("csrf-token")
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
		sideline := parts[2]
		if token == "" || csrf == "" || sideline == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if userCSRF != csrf {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		removed, erro := Redis_client.LRem(r.Context(), "getPosts"+dip+sideline, 1, token).Result()

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
			rows, e = Postgres_client.Query(r.Context(), "SELECT postid, title, info FROM posts ORDER BY created_at DESC LIMIT 10")
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
			rows, e = Postgres_client.Query(r.Context(), "SELECT postid, title, info FROM posts OFFSET $1 ORDER BY created_at DESC LIMIT 10", numberPage*10)
		}
		if e != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Bad request"))
			return
		}

		posts := make([]PostData, 10)
		for i := 0; rows.Next(); i++ {
			rows.Scan(&posts[0].Id, &posts[0].Title, &posts[0].Info)
		}
		rows.Close()

		jsoni := make(map[int]any, 10)
		for i := range 10 {
			jsoni[i] = PostData{Title: posts[i].Title, Info: posts[i].Info, Id: posts[i].Id}
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
