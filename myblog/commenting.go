package myblog

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
)

func Comment(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		payload := r.Header
		body := payload.Get("body")
		postid := payload.Get("postid")
		userData, ero := r.Cookie("userData")
		if ero != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if postid == "" || body == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		var userDataMap map[string]string
		ero = json.Unmarshal([]byte(userData.Value), &userDataMap)
		if ero != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		userid := userDataMap["userid"]

		if userid == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		_, ero = Postgres_client.Exec(r.Context(), "CALL insert_comment($1, $2, $3)", postid, userid, body)
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
	case "POST":
		payload := r.Header
		page := payload.Get("page")
		if page == "1" {
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
			rows, e = Postgres_client.Query(r.Context(), "SELECT commentid, body FROM comments OFFSET $1 ORDER BY created_at DESC LIMIT 30", numberPage*30)
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
