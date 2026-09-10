package myblog

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
)

func Post(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		payload := r.Header
		title := payload.Get("title")
		info := payload.Get("info")
		body := payload.Get("body")
		userData, ero := r.Cookie("userData")
		if ero != nil {
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

		_, ero = Postgres_client.Exec(r.Context(), "CALL insert_post($1, $2, $3, $4)", userid, title, info, body)
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
	Id    string `json:"postid"`
	Info  string `json:"info"`
	Body  string `json:"body"`
}

func GetPost(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		postid := mux.Vars(r)["postid"]
		if postid == "" {
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
			rows, e = Postgres_client.Query(r.Context(), "SELECT postid, title, info FROM posts OFFSET $1 ORDER BY created_at DESC LIMIT 10", numberPage*30)
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
