package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/feeds"
	"github.com/teambrookie/showrss/betaseries"
	"github.com/teambrookie/showrss/handlers"
	"github.com/teambrookie/showrss/torrent"
	"github.com/zabawaba99/firego"

	"flag"

	"syscall"

	"strconv"

	"cloud.google.com/go/storage"
	"github.com/braintree/manners"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
)

const version = "1.0.0"

const bucket = "showrss-64e4b.appspot.com"
const database = "https://showrss-64e4b.firebaseio.com"
const keyfile = "showrss_keyfile.json"

type Firebase struct {
	client *storage.Client
	*firego.Firebase
}

func NewFirebase(databaseSecret string) Firebase {
	//Init Firebase connection to database
	f := firego.New(database, nil)
	f.Auth(databaseSecret)

	//Init client to connect to google cloud storage
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithServiceAccountFile(keyfile))

	firebase := Firebase{client, f}
	return firebase
}

func torrentWorker(torrentJobs <-chan betaseries.Episode, firebase Firebase) {
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

func episodeWorker(users <-chan string, torrents chan<- betaseries.Episode, betaseries betaseries.EpisodeProvider, firebase Firebase) {
	for user := range users {
		tokenRef, _ := firebase.Ref(fmt.Sprintf("users/%s/token", user))
		var token string
		tokenRef.Value(&token)
		episodes, _ := betaseries.Episodes(token)
		var ids []int
		for _, episode := range episodes {

			var data interface{}
			torrentRef, _ := firebase.Ref(fmt.Sprintf("torrents/%d", episode.ID))
			torrentRef.Value(&data)
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

func (fire Firebase) getUserToken(userID string) string {
	tokenRef, _ := fire.Ref(fmt.Sprintf("/users/%s/token", userID))
	var token string
	tokenRef.Value(&token)
	return token
}

func (fire Firebase) getAllUsers() ([]string, error) {
	usersRef, err := fire.Ref("users")
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

func buildRSSFeed(userID string, firebase Firebase) {
	log.Println("Building rss feed for " + userID)
	now := time.Now()
	feed := &feeds.Feed{
		Title:       "ShowRSS by binou",
		Link:        &feeds.Link{Href: "https://github.com/TeamBrookie/showrss"},
		Description: "A list of torrent for your unseen Betaseries episodes",
		Author:      &feeds.Author{Name: "Fabien Foerster", Email: "fabienfoerster@gmail.com"},
		Created:     now,
	}
	var episodes []string
	episodesRef, _ := firebase.Ref("/users/" + userID + "/episodes")
	episodesRef.Value(&episodes)

	for _, ep := range episodes {
		torrentRef, _ := firebase.Ref("torrents/" + ep)
		var torrent betaseries.Episode
		torrentRef.Value(&torrent)
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
	f := NewFirebase(fireDatabaseSecret)
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
	go func() {
		for {
			<-rssLimiter
			usersRef, _ := f.Ref("users")
			var data map[string]interface{}
			usersRef.Value(&data)
			for k := range data {
				go buildRSSFeed(k, f)
			}
		}
	}()

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
