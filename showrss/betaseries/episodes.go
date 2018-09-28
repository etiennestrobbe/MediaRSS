package betaseries

import (
	"encoding/json"
	"net/http"
)

type Episode struct {
	ID        int    `json:"id"`
	TheTVDBID int    `json:"thetvdb_id"`
	Title     string `json:"title"`
	Season    int    `json:"season"`
	Episode   int    `json:"episode"`
	Show      struct {
		ID        int    `json:"id"`
		TheTVDBID int    `json:"thetvdb_id"`
		Title     string `json:"title"`
	} `json:"show"`
	Code string `json:"code"`
	User struct {
		Downloaded bool `json:"downloaded"`
	}
}

type betaseriesEpisodesResponse struct {
	Shows []struct {
		Unseen []Episode `json:"unseen"`
	} `json:"shows"`
	Errors []interface{} `json:"errors"`
}

//Episodes retrieve your unseen episode from betaseries
// and flatten the result so you have an array of Episode
func (b Betaseries) Episodes(token string) ([]Episode, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.betaseries.com/episodes/list", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-BetaSeries-Version", "2.4")
	req.Header.Add("X-BetaSeries-Key", b.APIKey)
	req.Header.Add("X-BetaSeries-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var res betaseriesEpisodesResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return nil, err
	}

	var episodes []Episode
	for _, show := range res.Shows {
		for _, ep := range show.Unseen {
			episodes = append(episodes, ep)
		}
	}

	return episodes, nil
}
