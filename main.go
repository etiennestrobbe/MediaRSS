package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/teambrookie/showrss/betaseries"
	"github.com/teambrookie/showrss/handlers"
	"github.com/teambrookie/showrss/torrent"
	"github.com/zabawaba99/firego"

	"flag"

	"syscall"

	"strconv"

	"github.com/braintree/manners"
)

const version = "1.0.0"

func torrentWorker(torrentJobs <-chan betaseries.Episode, firebase *firego.Firebase) {
	for episode := range torrentJobs {
		time.Sleep(2 * time.Second)
		log.Println("Processing : " + episode.Name)
		torrentLink, err := torrent.Search(strconv.Itoa(episode.ShowID), episode.Code, "720p")
		log.Println("Result : " + torrentLink)
		if err != nil {
			log.Printf("Error processing %s : %s ...\n", episode.Name, err)
			continue
		}
		if torrentLink == "" {
			continue
		}
		episode.MagnetLink = torrentLink
		episode.LastModified = time.Now()
		torrentRef, _ := firebase.Ref(fmt.Sprintf("torrents/%d", episode.ID))
		torrentRef.Set(episode)
		episodeRef, _ := firebase.Ref(fmt.Sprintf("episodes/%d", episode.ID))
		episodeRef.Remove()

	}
}

func episodeWorker(users <-chan string, torrents chan<- betaseries.Episode, betaseries betaseries.EpisodeProvider, firebase *firego.Firebase) {
	for user := range users {
		tokenRef, _ := firebase.Ref(fmt.Sprintf("users/%s/token", user))
		var token string
		tokenRef.Value(&token)
		episodes, _ := betaseries.Episodes(token)
		var ids []int
		for _, episode := range episodes {

			var data interface{}
			torrentRef, _ := firebase.Ref(fmt.Sprintf("torrents/%d", episode.ID))
			err := torrentRef.Value(&data)
			if err != nil {
				log.Println(err)
			}
			if data == nil {
				log.Println(episode.Name + " don't exist yet")
				epRef, _ := firebase.Ref(fmt.Sprintf("episodes/%d", episode.ID))
				epRef.Set(episode)
				torrents <- episode
			} else {
				log.Println(episode.Name + " exist already")
			}

			ids = append(ids, episode.ID)

		}
		episodesRef, _ := firebase.Ref(fmt.Sprintf("users/%s/episodes", user))
		episodesRef.Set(ids)
	}
}

func rssWorker(limiter <-chan time.Time) {
	for {
		<-limiter
		log.Println("LOL")
	}
}

func main() {
	var httpAddr = flag.String("http", "0.0.0.0:8000", "HTTP service address")
	flag.Parse()

	apiKey := os.Getenv("BETASERIES_KEY")
	if apiKey == "" {
		log.Fatalln("BETASERIES_KEY must be set in env")
	}

	fireDatabaseSecret := os.Getenv("FIREBASE_DATABASE_SECRET")
	if fireDatabaseSecret == "" {
		log.Fatalln("FIREBASE_DATABASE_SECRET must be set in env")
	}

	episodeProvider := betaseries.Betaseries{ApiKey: apiKey}

	log.Println("Starting server ...")
	log.Printf("HTTP service listening on %s", *httpAddr)

	//Firebase initialization
	f := firego.New("https://showrss-64e4b.firebaseio.com", nil)
	f.Auth(fireDatabaseSecret)
	// Worker stuff
	log.Println("Starting worker ...")
	betaseriesJobs := make(chan string, 100)
	torrentJobs := make(chan betaseries.Episode, 1000)
	go torrentWorker(torrentJobs, f)
	go episodeWorker(betaseriesJobs, torrentJobs, episodeProvider, f)

	rssLimiter := make(chan time.Time, 1)
	go func() {
		for t := range time.Tick(time.Second * 5) {
			rssLimiter <- t
		}
	}()
	go rssWorker(rssLimiter)

	errChan := make(chan error, 10)

	mux := http.NewServeMux()
	mux.Handle("/auth", handlers.AuthHandler(episodeProvider, f))
	mux.Handle("/refresh", handlers.RefreshHandler(betaseriesJobs, torrentJobs, f))
	mux.Handle("/rss", handlers.RSSHandler(episodeProvider))

	httpServer := manners.NewServer()
	httpServer.Addr = *httpAddr
	httpServer.Handler = handlers.LoggingHandler(mux)

	go func() {
		errChan <- httpServer.ListenAndServe()
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case err := <-errChan:
			if err != nil {
				log.Fatal(err)
			}
		case s := <-signalChan:
			log.Println(fmt.Sprintf("Captured %v. Exiting...", s))
			httpServer.BlockingClose()
			os.Exit(0)
		}
	}

}
