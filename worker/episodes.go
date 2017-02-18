package worker

import (
	"log"
	"strconv"

	"github.com/teambrookie/showrss/betaseries"
	"github.com/teambrookie/showrss/db"
)

//Episodes is a function that takes a channel of users
// get for each user the list of unseen episodes using betaseries
// and save that list to Firebase
// and also send it to the Torrents function to get the torrent Info ( could change later on)
func Episodes(users <-chan string, torrents chan<- betaseries.Episode, betaseries betaseries.EpisodeProvider, database *db.DB) {

	for user := range users {
		log.Println("Retriving episode for " + user)
		token := database.GetUserToken(user)
		episodes, _ := betaseries.Episodes(token)
		var ids []string
		for _, episode := range episodes {
			episodeID := strconv.Itoa(episode.ID)
			torrent := database.GetTorrentInfo(episodeID)
			if (torrent == db.Torrent{}) {
				log.Println("Don't exists yet --> " + episode.Name)
				database.AddEpisode(episodeID, episode)
				torrents <- episode
			} else {
				log.Println("Already exists dummy !!! --> " + episode.Name)
			}

			ids = append(ids, episodeID)

		}
		err := database.SetUserEpisodes(user, ids)
		if err != nil {
			log.Println("Error setting episodes for " + user)
		}
	}
}
