package myblog

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
)

func Search(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		payload := r.Header
		page := payload.Get("page")
		ilike := payload.Get("ilike")
		if page == "1" || ilike == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request"))
			return
		}
		var rows pgx.Rows
		var e error
		if page == "" {
			rows, e = Postgres_client.Query(r.Context(), "SELECT postid, title, info FROM posts WHERE title ILIKE '%' || $1 || '%' ORDER BY created_at DESC LIMIT 10", ilike)
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
			rows, e = Postgres_client.Query(r.Context(), "SELECT postid, title, info FROM posts WHERE title ILIKE '%' || $1 || '%' OFFSET $2 ORDER BY created_at DESC LIMIT 10", ilike, numberPage*10)
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
