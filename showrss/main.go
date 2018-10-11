package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/teambrookie/MediaRSS/commons/db"
	"github.com/teambrookie/MediaRSS/showrss/betaseries"
	"github.com/teambrookie/MediaRSS/showrss/dao"
	"github.com/teambrookie/MediaRSS/showrss/handlers"
	"github.com/teambrookie/MediaRSS/showrss/torrent"

	"flag"

	"syscall"
)

const version = "1.0.0"

func worker(jobs <-chan db.Media, store db.MediaStore) {
	for episode := range jobs {
		time.Sleep(2 * time.Second)
		log.Println("Processing : " + episode.Name)
		torrentLink, err := torrent.Search(episode.ID, episode.SearchTerm)
		log.Println("Result : " + torrentLink)
		if err != nil {
			log.Printf("Error processing %s : %s ...\n", episode.Name, err)
			continue
		}
		if torrentLink == "" {
			continue
		}
		episode.Magnet = torrentLink
		episode.LastUpdate = time.Now()
		err = store.UpdateMedia(episode, db.FOUND)
		if err != nil {
			log.Printf("Error saving %s to DB ...\n", episode.Name)
			continue
		}
		store.DeleteMedia(episode.ID, db.NOTFOUND)

	}
}

func main() {
	var httpAddr = flag.String("http", "0.0.0.0:8000", "HTTP service address")
	var dbAddr = flag.String("db", "showrss.db", "DB address")
	flag.Parse()

	apiKey := os.Getenv("BETASERIES_KEY")
	if apiKey == "" {
		log.Fatalln("BETASERIES_KEY must be set in env")
	}

	episodeProvider := betaseries.Betaseries{APIKey: apiKey}

	log.Println("Starting server ...")
	log.Printf("HTTP service listening on %s", *httpAddr)
	log.Println("Connecting to db ...")

	//DB stuff
	store, err := db.Open("showrss")
	if err != nil {
		log.Fatalln("Error connecting to DB")
	}

	// Worker stuff
	log.Println("Starting worker ...")
	jobs := make(chan dao.Episode, 1000)
	go worker(jobs, store)

	errChan := make(chan error, 10)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handlers.HelloHandler)
	mux.Handle("/auth", handlers.AuthHandler(episodeProvider))
	mux.Handle("/refresh", handlers.RefreshHandler(store, episodeProvider, jobs))
	mux.Handle("/episodes", handlers.EpisodeHandler(store))
	mux.Handle("/rss", handlers.RSSHandler(store, episodeProvider))

	httpServer := http.Server{}
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
			httpServer.Shutdown(context.Background())
			os.Exit(0)
		}
	}

}
