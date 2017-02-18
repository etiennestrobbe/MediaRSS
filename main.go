package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/teambrookie/showrss/betaseries"
	"github.com/teambrookie/showrss/db"
	"github.com/teambrookie/showrss/handlers"
	"github.com/teambrookie/showrss/worker"

	"flag"

	"syscall"

	"github.com/braintree/manners"
)

const version = "1.0.0"

func main() {
	var httpAddr = flag.String("http", "0.0.0.0:8000", "HTTP service address")
	flag.Parse()

	apiKey := os.Getenv("BETASERIES_KEY")
	if apiKey == "" {
		log.Fatalln("BETASERIES_KEY must be set in env")
	}

	episodeProvider := betaseries.Betaseries{ApiKey: apiKey}

	log.Println("Starting server ...")
	log.Printf("HTTP service listening on %s", *httpAddr)

	log.Println("Initializing DB ...")
	db := db.Init()

	// Worker stuff
	log.Println("Starting worker ...")
	userQueue := make(chan string, 100)
	episodeQueue := make(chan betaseries.Episode, 1000)
	// go torrentWorker(torrentJobs, f)
	// go episodeWorker(betaseriesJobs, torrentJobs, episodeProvider, f)

	//Stuff for rss worker
	rssLimiter := make(chan time.Time, 1)
	go func() {
		for t := range time.Tick(time.Second * 60) {
			rssLimiter <- t
		}
	}()
	go func() {
		for {
			<-rssLimiter
			users, err := db.GetAllUsers()
			if err != nil {
				log.Println(err)
			}
			for _, user := range users {
				go worker.RSS(user, &db)
			}
		}
	}()

	//Stuff for episode worker
	go worker.Episodes(userQueue, episodeQueue, episodeProvider, &db)
	episodeLimiter := make(chan time.Time, 1)
	go func() {
		for t := range time.Tick(time.Second * 30) {
			episodeLimiter <- t
		}
	}()
	go func() {
		for {
			<-episodeLimiter
			users, err := db.GetAllUsers()
			if err != nil {
				log.Println(err)
			}
			for _, user := range users {
				userQueue <- user
			}
		}
	}()

	//Stuff for torrent worker
	go worker.Torrents(episodeQueue, &db)
	torrentLimiter := make(chan time.Time, 1)
	go func() {
		for t := range time.Tick(time.Second * 15) {
			torrentLimiter <- t
		}
	}()
	go func() {
		for {
			<-torrentLimiter
			episodes, err := db.GetNotFoundEpisodes()
			log.Printf("Passing %d episode to torrent worker ...", len(episodes))
			if err != nil {
				log.Println("Error getting not found episodes from Firebase")
			}
			for _, ep := range episodes {
				episodeQueue <- ep
			}
		}
	}()

	errChan := make(chan error, 10)

	mux := http.NewServeMux()
	mux.Handle("/auth", handlers.AuthHandler(episodeProvider, db))
	mux.Handle("/rss", handlers.RSSHandler(db))

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
