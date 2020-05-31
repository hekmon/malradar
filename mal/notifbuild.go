package mal

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/darenliang/jikan-go"
	"github.com/hekmon/pushover/v2"
)

func (c *Controller) generateNotificationMsg(anime *jikan.Anime) pushover.Message {
	// download the image
	var attachment io.Reader
	if imgData, err := getHTTPFile(anime.ImageURL); err != nil {
		c.log.Errorf("[MAL] processing finished animes: can't download anime image: %v", err)
	} else {
		attachment = bytes.NewReader(imgData)
	}
	// extract list names
	studios := make([]string, len(anime.Studios))
	for index, studioItem := range anime.Studios {
		studios[index] = studioItem.Name
	}
	genres := make([]string, len(anime.Genres))
	for index, genreItem := range anime.Genres {
		genres[index] = genreItem.Name
	}
	// return the msg
	return pushover.Message{
		Message: fmt.Sprintf("<b>Score</b>\n%.2f (%d votes) ranked #%d\n<b>Episodes</b>\n%d %s (%s)\n<b>Studios</b>\n%s\n<b>Genres</b>\n%s\n<b>Rating</b>\n%s",
			anime.Score, anime.ScoredBy, anime.Rank,
			anime.Episodes, anime.Type, anime.Duration,
			strings.Join(studios, ", "),
			strings.Join(genres, ", "),
			anime.Rating,
		),
		Title:    getTitle(anime),
		Priority: pushover.PriorityNormal,
		URL:      anime.URL,
		URLTitle: "Check it on MyAnimeList",
		// Timestamp:  time.Now().Unix(),
		HTML:       true,
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
