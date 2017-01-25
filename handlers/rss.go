package handlers

import (
	"net/http"
	"time"

	"github.com/gorilla/feeds"
	"github.com/teambrookie/showrss/betaseries"
)

type rssHandler struct {
	episodeProvider betaseries.EpisodeProvider
}

func (h *rssHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "token must be set in query params", http.StatusNotAcceptable)
		return
	}

	now := time.Now()
	feed := &feeds.Feed{
		Title:       "ShowRSS by binou",
		Link:        &feeds.Link{Href: "https://github.com/TeamBrookie/showrss"},
		Description: "A list of torrent for your unseen Betaseries episodes",
		Author:      &feeds.Author{Name: "Fabien Foerster", Email: "fabienfoerster@gmail.com"},
		Created:     now,
	}
	// episodes, err := h.episodeProvider.Episodes(token)
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// }
	// for _, ep := range episodes {
	// 	episode, err := h.store.GetEpisode(ep.Name)
	// 	if episode.MagnetLink == "" || err != nil {
	// 		continue
	// 	}
	// 	description := fmt.Sprintf("<a href='%s'>MagnetLink</a>", episode.MagnetLink)
	// 	item := &feeds.Item{
	// 		Title:       episode.Name,
	// 		Link:        &feeds.Link{Href: episode.MagnetLink},
	// 		Description: description,
	// 		Created:     episode.LastModified,
	// 	}
	// 	feed.Add(item)
	// }

	w.Header().Set("Content-Type", "text/xml")
	feed.WriteRss(w)
	return

}

func RSSHandler(episodeProvider betaseries.EpisodeProvider) http.Handler {
	return &rssHandler{
		episodeProvider: episodeProvider,
	}
}
