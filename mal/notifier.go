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

func (c *Controller) notifier() {
	var anime *jikan.Anime
	for {
		select {
		case anime = <-c.pipeline:
			c.notify(anime)
		case <-c.ctx.Done():
			c.log.Info("[MAL] [Notifier] context done: stopping worker")
			return
		}
	}
}

func (c *Controller) notify(anime *jikan.Anime) {
	// filter out based on score
	if anime.Score < c.minScore {
		c.log.Infof("[MAL] [Notifier] '%s' (MalID %d) does not have the require score (%.2f/%.2f): skipping",
			getTitle(anime), anime.MalID, anime.Score, c.minScore)
		c.update.Lock()
		delete(c.watchList, anime.MalID)
		c.update.Unlock()
		return
	}
	// filter out based on genres
	if bl := c.getBlacklistedGenres(anime); len(bl) > 0 {
		c.log.Infof("[MAL] [Notifier] '%s' (MalID %d) has the required score (%.2f/%.2f) but contains blacklisted genr(s): %s",
			getTitle(anime), anime.MalID, anime.Score, c.minScore, strings.Join(bl, ", "))
		c.update.Lock()
		delete(c.watchList, anime.MalID)
		c.update.Unlock()
		return
	}
	// send the notification
	if err := c.pushover.SendCustomMsg(c.generateNotificationMsg(anime)); err != nil {
		c.log.Errorf("[MAL] [Notifier] sending pushover notification failed for '%s' (MalID %d): %v",
			getTitle(anime), anime.MalID, err)
		// do not delete its status in order to have a chance to notify it again later
	} else {
		c.log.Infof("[MAL] [Notifier] pushover notification sent for '%s' (MalID %d)",
			getTitle(anime), anime.MalID)
		// notification sent successfully, we can mark it finished within the state
		c.update.Lock()
		delete(c.watchList, anime.MalID)
		c.update.Unlock()
	}
}

func (c *Controller) getBlacklistedGenres(anime *jikan.Anime) (matches []string) {
	matches = make([]string, 0, len(c.blGenres))
	for blacklisted := range c.blGenres {
		for _, genre := range anime.Genres {
			if genre.Name == blacklisted {
				matches = append(matches, blacklisted)
			}
		}
	}
	return
}

func (c *Controller) generateNotificationMsg(anime *jikan.Anime) pushover.Message {
	// download the image
	var attachment io.Reader
	if imgData, err := getHTTPFile(anime.ImageURL); err != nil {
		c.log.Errorf("[MAL] [Notifier] can't download anime image: %v", err)
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

func getTitle(anime *jikan.Anime) string {
	if anime.TitleEnglish != "" {
		return anime.TitleEnglish
	}
	return anime.Title
}
