package handlers

import (
	"net/http"

	"github.com/teambrookie/showrss/betaseries"
	"github.com/zabawaba99/firego"
)

type refreshHandler struct {
	users    chan string
	episodes chan betaseries.Episode
	fire     *firego.Firebase
}

func (h *refreshHandler) getAllUsers() ([]string, error) {
	usersRef, err := h.fire.Ref("users")
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	if err := usersRef.Value(&data); err != nil {
		return nil, err
	}
	var users []string
	for k := range data {
		users = append(users, k)
	}
	return users, nil
}

func (h *refreshHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	users, err := h.getAllUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	for _, user := range users {
		h.users <- user
	}
	w.WriteHeader(http.StatusOK)
	return

}

func RefreshHandler(users chan string, episodes chan betaseries.Episode, fire *firego.Firebase) http.Handler {
	return &refreshHandler{
		users:    users,
		episodes: episodes,
		fire:     fire,
	}
}
