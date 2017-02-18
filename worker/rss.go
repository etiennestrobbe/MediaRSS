package worker

import (
	"fmt"
	"time"

	"io"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/feeds"
	"github.com/teambrookie/showrss/db"
)

//RSS is a function that for a specified user generate his RSS feed and save it to Google Cloud storage
func RSS(userID string, db db.DB) {
	log.Info("Building rss feed for " + userID)
	now := time.Now()
	feed := &feeds.Feed{
		Title:       "ShowRSS by binou",
		Link:        &feeds.Link{Href: "https://github.com/TeamBrookie/showrss"},
		Description: "A list of torrent for your unseen Betaseries episodes",
		Author:      &feeds.Author{Name: "Fabien Foerster", Email: "fabienfoerster@gmail.com"},
		Created:     now,
	}
	episodes := db.GetUserEpisodes(userID)

	for _, ep := range episodes {
		torrent := db.GetTorrentInfo(ep)
		if torrent.MagnetLink == "" {
			continue
		}
		description := fmt.Sprintf("<a href='%s'>MagnetLink</a>", torrent.MagnetLink)
		item := &feeds.Item{
			Title:       torrent.Name,
			Link:        &feeds.Link{Href: torrent.MagnetLink},
			Description: description,
			Created:     torrent.LastModified,
		}
		feed.Add(item)
	}

	reader, writer := io.Pipe()
	//When using pipe you need to write in a goroutine because if you don't shit won't get done
	go func() {
		defer writer.Close()
		err := feed.WriteRss(writer)
		if err != nil {
			log.Error(err)
		}

	}()

	err := db.SaveUserFeed(userID, reader)
	if err != nil {
		log.Error(err)
	}
}
