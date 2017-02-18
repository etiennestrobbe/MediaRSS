package worker

import (
	"log"
	"strconv"
	"time"

	"github.com/teambrookie/showrss/betaseries"
	"github.com/teambrookie/showrss/db"
	"github.com/teambrookie/showrss/torrent"
)

//Torrents is a function that take a channel of episode and for each get the torrent info
// using rarbg.to api and save that to Firebase
func Torrents(torrentJobs <-chan betaseries.Episode, database *db.DB) {
	// for fun rarbg limit to 1req/2secs
	time.Sleep(3 * time.Second)
	for episode := range torrentJobs {
		log.Println("Processing : " + episode.Name)
		episodeID := strconv.Itoa(episode.ShowID)
		to := database.GetTorrentInfo(episodeID)
		log.Println(to)
		if (to != db.Torrent{}) {
			log.Println("Torrent already exists")
			continue
		}

		torrentLink, err := torrent.Search(episodeID, episode.Code, "720p")
		log.Println("Result : " + torrentLink)
		if err != nil {
			log.Printf("Error processing %s : %s ...\n", episode.Name, err)
		}
		if torrentLink != "" {
			episode.MagnetLink = torrentLink
			episode.LastModified = time.Now()
			episodeID := strconv.Itoa(episode.ID)

			err = database.SaveTorrentInfo(episodeID, episode)
			if err != nil {
				log.Println("Error saving : " + episode.Name)
			}
			err = database.RemoveEpisode(episodeID)
			if err != nil {
				log.Println("Error removing " + episode.Name + " from queue ...")
			}
		}

		time.Sleep(2 * time.Second)
	}
}
