package torrent

import torrentapi "github.com/qopher/go-torrentapi"

type Torrent struct {
	Name   string `json:"name"`
	Magnet string `json:"magnet"`
	Leechs int    `json:"leechs"`
	Seeds  int    `json:"seeds"`
}

func goodEnoughTorrent(results torrentapi.TorrentResults) Torrent {
	for _, t := range results {
		if t.Seeders > 0 || t.Leechers > 0 {
			return Torrent{Name: t.Title, Magnet: t.Download, Leechs: t.Leechers, Seeds: t.Seeders}
		}
	}
	return Torrent{}
}

func SearchEpisode(showID string, epCode string) (Torrent, error) {
	api, err := torrentapi.New("SHOWRSS")
	if err != nil {
		return Torrent{}, err
	}
	api.Format("json_extended")
	api.SearchTVDB(showID)
	api.SearchString(epCode + "720p")
	results, err := api.Search()
	if err != nil || len(results) == 0 {
		return Torrent{}, err
	}
	return goodEnoughTorrent(results), nil
}
