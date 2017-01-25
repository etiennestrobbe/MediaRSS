package betaseries

import "time"

type Episode struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Code         string    `json:"code"`
	ShowID       int       `json:"show_id"`
	MagnetLink   string    `json:"magnet_link"`
	LastModified time.Time `json:"last_modified"`
}

type EpisodeProvider interface {
	Auth(string, string) (string, error)
	Episodes(string) ([]Episode, error)
}

type Betaseries struct {
	ApiKey string
}
