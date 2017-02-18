package handlers

import (
	"net/http"
	"time"

	"io"

	"github.com/teambrookie/showrss/db"
)

type rssHandler struct {
	database db.DB
}

func (h *rssHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")

	if username == "" {
		http.Error(w, "empty login/password", http.StatusBadRequest)
		return
	}

	feed, err := h.database.GetUserFeed(username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cacheUntil := time.Now().Add(1 * time.Hour).Format(http.TimeFormat)
	w.Header().Set("Expires", cacheUntil)
	io.Copy(w, feed)
	return

}

//RSSHandler is responsible for returning the correct RSS feed to the user
func RSSHandler(database db.DB) http.Handler {
	return &rssHandler{
		database: database,
	}
}
