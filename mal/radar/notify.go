package radar

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/hekmon/malwatcher/mal/userlist"

	"github.com/darenliang/jikan-go"
	"github.com/hekmon/pushover/v2"
)

func (c *Controller) batchNotifier(animes []*jikan.Anime) {
	// do we actually have work to do ?
	if len(animes) == 0 {
		return
	}
	c.log.Infof("[MAL] [Notify] got %d potential animes, applying filters...", len(animes))
	// get user list if any
	var userAnimes userlist.List
	if c.user != "" {
		var err error
		if userAnimes, err = userlist.GetAllUserAnimes(c.user); err != nil {
			c.log.Errorf("[MAL] [Notify] user list filtering: can't get '%s' animes list: %v",
				c.user, err)
		} else {
			c.log.Infof("[MAL] [Notify] user list filtering: recovered %d anime(s) for user '%s'",
				len(userAnimes), c.user)
		}
	} else {
		c.log.Debug("[MAL] [Notify] user list filtering: user unset: skipping")
	}
	// process animes
	for _, anime := range animes {
		c.notify(anime, userAnimes)
	}
}

func (c *Controller) notify(anime *jikan.Anime, userAnimes userlist.List) {
	// filter out based on types
	if bl := c.isBlacklistedType(anime); bl != "" {
		c.log.Infof("[MAL] [Notify] '%s' (MalID %d) has a blacklisted type: %s: skipping",
			getTitle(anime), anime.MalID, bl)
		c.update.Lock()
		delete(c.watchList, anime.MalID)
		c.update.Unlock()
		return
	}
	// filter out based on genres
	if bl := c.getBlacklistedGenres(anime); len(bl) > 0 {
		c.log.Infof("[MAL] [Notify] '%s' (MalID %d) contains blacklisted genre(s): %s: skipping",
			getTitle(anime), anime.MalID, strings.Join(bl, ", "))
		c.update.Lock()
		delete(c.watchList, anime.MalID)
		c.update.Unlock()
		return
	}
	// filter out based on score
	if anime.Score < c.minScore {
		c.log.Infof("[MAL] [Notify] '%s' (MalID %d) does not have the require score (%.2f/%.2f): skipping",
			getTitle(anime), anime.MalID, anime.Score, c.minScore)
		c.update.Lock()
		delete(c.watchList, anime.MalID)
		c.update.Unlock()
		return
	}
	// filter out based on user list if any
	if len(userAnimes) != 0 {
		if userAnimes.Contains(anime.MalID) {
			c.log.Infof("[MAL] [Notify] '%s' (MalID %d) is already present on '%s' user list: skipping",
				getTitle(anime), anime.MalID, c.user)
			c.update.Lock()
			delete(c.watchList, anime.MalID)
			c.update.Unlock()
			return
		}
	}
	// send the notification
	if err := c.pushover.SendCustomMsg(c.generateNotificationMsg(anime)); err != nil {
		c.log.Errorf("[MAL] [Notify] '%s' (MalID %d) does not have the require score (%.2f/%.2f): pushover notification failed: %v",
			getTitle(anime), anime.MalID, anime.Score, c.minScore, err)
		// do not delete its status in order to have a chance to notify it again later
	} else {
		c.log.Infof("[MAL] [Notify] '%s' (MalID %d) does not have the require score (%.2f/%.2f): pushover notification sent",
			getTitle(anime), anime.MalID, anime.Score, c.minScore)
		// notification sent successfully, we can remove it from the state
		c.update.Lock()
		delete(c.watchList, anime.MalID)
		c.update.Unlock()
	}
}

func (c *Controller) isBlacklistedType(anime *jikan.Anime) (blacklisted string) {
	for _, blacklisted := range c.blTypes {
		if anime.Type == blacklisted {
			return blacklisted
		}
	}
	return
}

func (c *Controller) getBlacklistedGenres(anime *jikan.Anime) (matches []string) {
	matches = make([]string, 0, len(c.blGenres))
	for _, blacklisted := range c.blGenres {
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
		c.log.Errorf("[MAL] [Notify] can't download anime image: %v", err)
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
		Title:      getTitle(anime),
		Priority:   pushover.PriorityNormal,
		URL:        anime.URL,
		URLTitle:   "Check it on MyAnimeList",
		Timestamp:  anime.Aired.To.Unix(),
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
