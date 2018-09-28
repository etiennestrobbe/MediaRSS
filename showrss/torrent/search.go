package torrent

import (
	"github.com/qopher/go-torrentapi"
)

func goodEnoughTorrent(results torrentapi.TorrentResults) string {
	for _, t := range results {
		if t.Seeders > 0 || t.Leechers > 0 {
			return t.Download
		}
	}
	return ""
}

func Search(showID string, searchTerm string) (string, error) {
	api, err := torrentapi.New("SHOWRSS")
	if err != nil {
		return "", err
	}
	api.Format("json_extended")
	api.SearchTVDB(showID)
	api.SearchString(searchTerm)
	results, err := api.Search()
	if err != nil {
		return "", err
	}

	if len(results) == 0 {
		return "", nil
	}
	return goodEnoughTorrent(results), nil
}
