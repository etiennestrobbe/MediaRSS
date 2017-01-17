package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/teambrookie/showrss/betaseries"
	"github.com/zabawaba99/firego"
)

type AuthResponse struct {
	Username string `json:"username"`
	Token    string `json:"token"`
}

type authHandler struct {
	episodeProvider betaseries.EpisodeProvider
	firebase        *firego.Firebase
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
	usersRef, err := h.firebase.Ref("users/" + username)
	usersRef.Set(response)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	return
}

func AuthHandler(episodeProvider betaseries.EpisodeProvider, f *firego.Firebase) http.Handler {
	return &authHandler{
		episodeProvider: episodeProvider,
		firebase:        f,
	}
}
