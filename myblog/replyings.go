package myblog

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
)

func Reply(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		payload := r.Header
		body := payload.Get("body")
		postid := payload.Get("postid")
		commentid := payload.Get("commentid")
		userData, ero := r.Cookie("userData")
		if ero != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		if postid == "" || body == "" || commentid == "" {
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

		_, ero = Postgres_client.Exec(r.Context(), "CALL insert_reply($1, $2, $3, $4)", postid, userid, commentid, body)
		if ero != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

type ReplyData struct {
	Replyid string `json:"replyid"`
	Body    string `json:"body"`
}

func GetReplies(w http.ResponseWriter, r *http.Request) { // GetPosts dosent require refresh tokens
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
			rows, e = Postgres_client.Query(r.Context(), "SELECT replyid, body FROM replies ORDER BY created_at DESC LIMIT 30")
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
			rows, e = Postgres_client.Query(r.Context(), "SELECT replyid, body FROM replies OFFSET $1 ORDER BY created_at DESC LIMIT 30", numberPage*30)
		}
		if e != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Bad request"))
			return
		}

		replies := make([]ReplyData, 30)
		for i := 0; rows.Next(); i++ {
			rows.Scan(&replies[0].Replyid, &replies[0].Body)
		}
		rows.Close()

		jsoni := make(map[int]any, 30)
		for i := range 30 {
			jsoni[i] = ReplyData{Replyid: replies[i].Replyid, Body: replies[i].Body}
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
