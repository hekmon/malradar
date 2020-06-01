package userlist

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	// MaxAnimesPerPage is the maximum number of items a single call to GetUserList() can return
	MaxAnimesPerPage   = 300
	malUserSafetyLimit = 100000
)

// GetAllUserAnimes is a wrapper around GetUserList() which will repeat calls while adapting the offset
// in order to build a complete list. It has an upper safety limit (see malUserSafetyLimit) to avoid
// potential infinite loops.
func GetAllUserAnimes(user string) (animes List, err error) {
	var pageAnimes []Anime
	for offset := 0; offset < malUserSafetyLimit; offset += MaxAnimesPerPage {
		if pageAnimes, err = GetUserList(user, StatusAll, offset); err != nil {
			err = fmt.Errorf("error while recovering the %d page (offset %d) of '%s' list: %w",
				(offset/MaxAnimesPerPage)+1, offset, user, err)
			return
		}
		if len(pageAnimes) == 0 {
			// we got them all
			return
		}
		animes = append(animes, pageAnimes...)
	}
	return
}

// GetUserList returns a single page (check MaxAnimesPerPage for maximum number of items per call) of a user personnal list.
// Use offset to request other pages and status to filter the results.
func GetUserList(user string, status Status, offset int) (pageAnimes List, err error) {
	url := fmt.Sprintf("https://myanimelist.net/animelist/Akarin/load.json?offset=%d&status=%d", offset, status)
	response, err := http.Get(url)
	if err != nil {
		err = fmt.Errorf("getting '%s' failed: %w", url, err)
		return
	}
	defer response.Body.Close()
	if err = json.NewDecoder(response.Body).Decode(&pageAnimes); err != nil {
		err = fmt.Errorf("decoding response from '%s' as JSON failed: %w", url, err)
	}
	return
}
