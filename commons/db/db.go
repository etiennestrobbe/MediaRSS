package db

import (
	"time"
)

//Torrent is what i want it to be
type Torrent struct {
	Name       string `json:"name"`
	MagnetLink string `json:"magnet_link"`
	Seeds      int    `json:"seeds"`
	Leechs     int    `json:"leechs"`
}

//Media is a generic type
type Media struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Torrent    Torrent     `json:"torrent"`
	LastUpdate time.Time   `json:"last_update"`
	Metadata   interface{} `json:"metadata"`
}

//MediaStore define the interface for retriving media
type MediaStore interface {
	GetCollection(collection string) ([]Media, error)
	GetMedia(mediaID string, collection string) (Media, error)
	AddMedia(media Media, collection string) error
	UpdateMedia(media Media, collection string) error
	DeleteMedia(mediaID string, collection string) error
	Close() error
}
