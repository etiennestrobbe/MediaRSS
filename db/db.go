package db

import (
	"fmt"
	"os"
	"time"

	"io"

	"cloud.google.com/go/storage"
	log "github.com/Sirupsen/logrus"
	"github.com/teambrookie/showrss/betaseries"
	"github.com/zabawaba99/firego"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
)

const bucket = "showrss-64e4b.appspot.com"
const database = "https://showrss-64e4b.firebaseio.com"
const keyfile = "showrss_keyfile.json"

//Torrent is a type representing a torrent
type Torrent struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Code         string    `json:"code"`
	ShowID       int       `json:"show_id"`
	MagnetLink   string    `json:"magnet_link"`
	LastModified time.Time `json:"last_modified"`
}

// DB is a type wrapping both the Firebase RealTime Database and the Google Cloud storage
// and expose convenient api for me
type DB struct {
	ggs      *storage.Client
	firebase *firego.Firebase
}

//Init return a new instance of DB
func Init() DB {
	fireDatabaseSecret := os.Getenv("FIREBASE_DATABASE_SECRET")
	if fireDatabaseSecret == "" {
		log.Errorln("FIREBASE_DATABASE_SECRET must be set in env")
	}
	//Init Firebase connection to database
	f := firego.New(database, nil)
	f.Auth(fireDatabaseSecret)

	//Init client to connect to google cloud storage
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithServiceAccountFile(keyfile))
	if err != nil {
		log.Error(err)
	}
	db := DB{client, f}
	return db
}

//GetUserToken return the token for the specified user
func (db DB) GetUserToken(userID string) string {
	tokenRef, _ := db.firebase.Ref(fmt.Sprintf("/users/%s/token", userID))
	var token string
	tokenRef.Value(&token)
	return token
}

//GetAllUsers return an array of all the users
func (db DB) GetAllUsers() ([]string, error) {
	usersRef, err := db.firebase.Ref("users")
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

//GetNotFoundEpisodes return all the currently not found episode
func (db DB) GetNotFoundEpisodes() ([]betaseries.Episode, error) {
	episodesRef, err := db.firebase.Ref("episodes")
	if err != nil {
		return nil, err
	}
	var data map[string]betaseries.Episode
	if err := episodesRef.Value(&data); err != nil {
		return nil, err
	}
	var episodes []betaseries.Episode
	for _, v := range data {
		episodes = append(episodes, v)
	}
	return episodes, nil
}

//GetUserEpisodes return all the ids for the unseen episodes of a user
func (db DB) GetUserEpisodes(userID string) []string {
	var episodes []string
	episodesRef, _ := db.firebase.Ref("/users/" + userID + "/episodes")
	episodesRef.Value(&episodes)
	return episodes
}

//SetUserEpisodes set the unseen episodes for a the specified user
func (db DB) SetUserEpisodes(userID string, episodesIDS []string) error {
	episodesRef, err := db.firebase.Ref(fmt.Sprintf("users/%s/episodes", userID))
	if err != nil {
		return err
	}
	err = episodesRef.Set(episodesIDS)
	return err
}

//AddEpisode save the episode to Firebase
func (db DB) AddEpisode(episodeID string, episode interface{}) error {
	epRef, err := db.firebase.Ref(fmt.Sprintf("episodes/%s", episodeID))
	if err != nil {
		return err
	}
	err = epRef.Set(episode)
	return err
}

//RemoveEpisode remove the episode from Firebase
func (db DB) RemoveEpisode(episodeID string) error {
	epRef, err := db.firebase.Ref(fmt.Sprintf("episodes/%s", episodeID))
	if err != nil {
		return err
	}
	err = epRef.Remove()
	return err
}

//GetTorrentInfo return the torrent information for an episodes
func (db DB) GetTorrentInfo(episodeID string) Torrent {
	torrentRef, _ := db.firebase.Ref("torrents/" + episodeID)
	var torrent Torrent
	torrentRef.Value(&torrent)
	return torrent
}

//SaveTorrentInfo save the torrent info for an episode into Firebase
func (db DB) SaveTorrentInfo(episodeID string, torrentInfo interface{}) error {
	torrentRef, err := db.firebase.Ref(fmt.Sprintf("torrents/%s", episodeID))
	if err != nil {
		return err
	}
	err = torrentRef.Set(torrentInfo)
	return err
}

// SaveUser save a user Data to firebase
func (db DB) SaveUser(username, token string) error {
	usersRef, err := db.firebase.Ref("users/" + username)
	if err != nil {
		return err
	}
	userData := map[string]string{"username": username, "token": token}
	err = usersRef.Set(userData)
	return err
}

//SaveUserFeed save a user feed to Google Cloud Storage using Firebase Bucket
func (db DB) SaveUserFeed(username string, feed io.Reader) error {
	filename := fmt.Sprintf("%s.rss", username)
	ctx := context.Background()
	wc := db.ggs.Bucket(bucket).Object(filename).NewWriter(ctx)
	_, err := io.Copy(wc, feed)
	if err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}
	return nil
}

//GetUserFeed return the rss feed as saved to Google Cloud Storage
func (db DB) GetUserFeed(username string) (io.Reader, error) {
	filename := fmt.Sprintf("%s.rss", username)
	ctx := context.Background()
	rc, err := db.ggs.Bucket(bucket).Object(filename).NewReader(ctx)
	return rc, err

}
