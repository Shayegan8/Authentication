package myblog

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

func Comment(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		BucketHandlement("comment", "comment", w, r)
	case "POST":
		payload := r.Header
		cookie, ero := r.Cookie("comment")
		userCSRF := payload.Get("csrf-Token")
		post := payload.Get("comment")
		comment := payload.Get("info")

		if userCSRF == "" || post == "" || comment == "" {
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

		removed, ero := Redis_client.LRem(r.Context(), "comment"+email, 1, token).Result()

		if ero != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		} else if removed == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		_, ero = Postgres_client.Exec(r.Context(), "INSERT INTO comments(post, body) VALUES ($1, $2)", post, comment)
		if ero != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

type CommentData struct {
	Commentid string `json:"commentId"`
	Body      string `json:"body"`
}

func GetComments(w http.ResponseWriter, r *http.Request) { // GetPosts dosent require refresh tokens
	switch r.Method {
	case "GET":
		BucketHandlement("getComments", "getComments", w, r)
	case "POST":
		payload := r.Header
		page := payload.Get("page")
		if page == "1" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		dip := payload.Get("realip")
		cookie, ero := r.Cookie("getComments")
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
			removed, erro = Redis_client.LRem(r.Context(), "getComments"+email, 1, token).Result()
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
			rows, e = Postgres_client.Query(r.Context(), "SELECT commentid, body FROM comments ORDER BY created_at DESC LIMIT 30")
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
			rows, e = Postgres_client.Query(r.Context(), "SELECT commentid, body FROM comments OFFSET $1 ORDER BY created_at DESC LIMIT 30", numberPage*10)
		}
		if e != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Bad request"))
			return
		}

		comments := make([]CommentData, 30)
		for i := 0; rows.Next(); i++ {
			rows.Scan(&comments[0].Commentid, &comments[0].Body)
		}
		rows.Close()

		jsoni := make(map[int]any, 30)
		for i := range 30 {
			jsoni[i] = CommentData{Commentid: comments[i].Commentid, Body: comments[i].Body}
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
