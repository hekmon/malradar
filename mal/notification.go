package mal

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/darenliang/jikan-go"
	"github.com/hekmon/pushover/v2"
)

const (
	notifMsgTemplate = ``
)

func (c *Controller) generateNotificationMsg(anime *jikan.Anime) (msg pushover.Message) {
	// download the image
	var attachment io.Reader
	if imgData, err := getHTTPFile(anime.ImageURL); err != nil {
		c.log.Errorf("[MAL] processing finished animes:", err)
	} else {
		attachment = bytes.NewReader(imgData)
	}
	// forge the msg
	return pushover.Message{
		Message:    fmt.Sprintf("Score: %f (%d votes) #%d", anime.Score, anime.ScoredBy, anime.Rank),
		Title:      getTitle(anime),
		Priority:   pushover.PriorityNormal,
		URL:        anime.URL,
		URLTitle:   "Check it on MyAnimeList",
		Timestamp:  anime.Aired.To.Unix(),
		Attachment: attachment,
	}
}

func getHTTPFile(url string) (file []byte, err error) {
	response, err := http.Get(url)
	if err != nil {
		return
	}
	defer response.Body.Close()
	return ioutil.ReadAll(response.Body)
}
