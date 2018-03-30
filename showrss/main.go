package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/teambrookie/hermes/showrss/betaseries"
	"github.com/teambrookie/hermes/showrss/dao"
	"github.com/teambrookie/hermes/showrss/handlers"
	"github.com/teambrookie/hermes/showrss/torrent"

	"flag"

	"syscall"

	"strconv"
)

const version = "1.0.0"

func worker(jobs <-chan dao.Episode, store dao.EpisodeStore) {
	for episode := range jobs {
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
		err = store.UpdateEpisode(episode)
		if err != nil {
			log.Printf("Error saving %s to DB ...\n", episode.Name)
		}

	}
}

func main() {
	var dbAddr = flag.String("db", "showrss.db", "DB address")
	flag.Parse()

	apiKey := os.Getenv("BETASERIES_KEY")
	if apiKey == "" {
		log.Fatalln("BETASERIES_KEY must be set in env")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	episodeProvider := betaseries.Betaseries{APIKey: apiKey}

	log.Println("Starting server ...")
	log.Printf("HTTP service listening on %s", port)
	log.Println("Connecting to db ...")

	//DB stuff
	store, err := dao.InitDB(*dbAddr)
	if err != nil {
		log.Fatalln("Error connecting to DB")
	}

	err = store.CreateBucket("episodes")
	if err != nil {
		log.Fatalln("Error when creating bucket")
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
	httpServer.Addr = ":" + port
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
