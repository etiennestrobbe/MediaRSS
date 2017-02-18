package handlers

import (
	"encoding/json"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/teambrookie/showrss/betaseries"
	"github.com/teambrookie/showrss/db"
)

// AuthResponse describe the respond issue after an authentification
type AuthResponse struct {
	Username string `json:"username"`
	Token    string `json:"token"`
}

type authHandler struct {
	episodeProvider betaseries.EpisodeProvider
	db              db.DB
}

func (h *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		http.Error(w, "empty login/password", http.StatusUnauthorized)
		return
	}
	token, err := h.episodeProvider.Auth(username, password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	response := AuthResponse{
		Username: username,
		Token:    token,
	}

	// Save the user and his token to Firebase RealTimeDatabase
	go func() {
		err := h.db.SaveUser(username, token)
		if err != nil {
			log.Errorf("Error saving %s to the database : %s", username, err)
		}
	}()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	return
}

//AuthHandler handle the connection of the user to Betaseries
func AuthHandler(episodeProvider betaseries.EpisodeProvider, db db.DB) http.Handler {
	return &authHandler{
		episodeProvider: episodeProvider,
		db:              db,
	}
}
