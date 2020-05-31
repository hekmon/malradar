package mal

import (
	"fmt"
	"strings"
	"time"

	"github.com/darenliang/jikan-go"
)

const (
	fetchFreq           = 24 * time.Hour
	animeStatusNotAired = "Not yet aired"
	animeStatusOnGoing  = "Currently Airing"
	animeStatusFinished = "Finished Airing"
)

func (c *Controller) watcher() {
	// create the ticker
	ticker := time.NewTicker(fetchFreq)
	defer ticker.Stop()
	// start the first batch
	c.batch()
	// reexecute batch at each tick
	for {
		select {
		case <-ticker.C:
			c.batch()
		case <-c.ctx.Done():
			c.log.Info("[MAL] context done: stopping worker")
			return
		}
	}
}

func (c *Controller) batch() {
	start := time.Now()
	c.log.Info("[MAL] starting new batch")
	defer func() {
		c.log.Infof("[MAL] batch executed in %v", time.Since(start))
	}()
	// first run ever ?
	if c.watchList == nil {
		c.log.Infof("[MAL] initializing watch list...")
		if err := c.buildInitialList(); err != nil {
			c.watchList = nil
			c.log.Errorf("[MAL] failed to build initial list: %v", err)
		}
		return
	}
	// update state of known anime
	oldFinished := c.updateCurrentState()
	// try to find new ones
	newFinished := c.findNewAnimes()
	// notify
	c.processFinished(append(oldFinished, newFinished...))
}

func (c *Controller) buildInitialList() (err error) {
	var (
		seasonList   *jikan.Season
		animeDetails *jikan.Anime
		previousLen  int
		found        bool
	)
	year, season := currentSeason()
	for i := 0; i < c.nbSeasons; i++ {
		previousLen = len(c.watchList)
		// get season list
		c.rateLimiter()
		if seasonList, err = jikan.GetSeason(year, season); err != nil {
			err = fmt.Errorf("iteration %d (%s %d): failing to acquire season animes: %w",
				i+1, season, year, err)
			return
		}
		c.log.Infof("[MAL] building initial list: season %d/%d (%s %d): fetching details for %d animes...",
			i+1, c.nbSeasons, season, year, len(seasonList.Anime))
		if c.watchList == nil {
			c.watchList = make(map[int]string, c.nbSeasons*len(seasonList.Anime)*3/2) // ×1.5
		}
		// for each anime
		for index, anime := range seasonList.Anime {
			// do we have it from an earlier season ?
			if _, found = c.watchList[anime.MalID]; found {
				c.log.Debugf("[MAL] building initial list: season %d/%d (%s %d): anime %d/%d: '%s' (MalID %d): already in the list",
					i+1, c.nbSeasons, season, year, index, len(seasonList.Anime), anime.Title, anime.MalID, animeDetails.Status)
				continue
			}
			// get its details
			c.rateLimiter()
			if animeDetails, err = jikan.GetAnime(anime.MalID); err != nil {
				err = fmt.Errorf("iteration %d (%s %d): failing to acquire anime %d details: %w",
					i+1, season, year, anime.MalID, err)
				return
			}
			// save data
			c.update.Lock()
			for _, genre := range animeDetails.Genres {
				c.genres.Add(genre.Name)
			}
			c.ratings.Add(animeDetails.Rating)
			c.watchList[anime.MalID] = animeDetails.Status
			c.update.Unlock()
			c.log.Debugf("[MAL] building initial list: season %d/%d (%s %d): anime %d/%d: '%s' (MalID %d) with '%s' state",
				i+1, c.nbSeasons, season, year, index, len(seasonList.Anime), getTitle(animeDetails), animeDetails.MalID, animeDetails.Status)
		}
		// season done
		c.log.Infof("[MAL] building initial list: season %d/%d (%s %d): added %d/%d animes",
			i+1, c.nbSeasons, season, year, len(c.watchList)-previousLen, len(seasonList.Anime))
		// prepare for next run
		year, season = previousSeason(season, year)
	}
	return
}

func (c *Controller) updateCurrentState() (finished []*jikan.Anime) {
	var (
		err          error
		animeDetails *jikan.Anime
	)
	finished = make([]*jikan.Anime, 0, len(c.watchList))
	index := 1
	for malID, oldStatus := range c.watchList {
		// only update the ones which need to
		if oldStatus == animeStatusFinished {
			continue
		}
		// get current details
		c.rateLimiter()
		if animeDetails, err = jikan.GetAnime(malID); err != nil {
			c.log.Errorf("[MAL] updating state: [%d/%d] can't check current status of MalID %d: %s",
				index, len(c.watchList), malID, err)
			continue
		}
		// save filters data
		c.update.Lock()
		for _, genre := range animeDetails.Genres {
			c.genres.Add(genre.Name)
		}
		c.ratings.Add(animeDetails.Rating)
		c.update.Unlock()
		// has status changed ?
		if animeDetails.Status != oldStatus {
			if animeDetails.Status == animeStatusFinished {
				// do not update internal state as the successfull notification will delete the key
				// by keeping the previous state this will act as a recovery mechanism if the program
				// is interupted before being able to send the notification
				finished = append(finished, animeDetails)
				c.log.Infof("[MAL] updating state: [%d/%d] '%s' (MalID %d) is now finished",
					index, len(c.watchList), getTitle(animeDetails), malID)
			} else {
				c.update.Lock()
				c.watchList[malID] = animeDetails.Status
				c.update.Unlock()
				c.log.Debugf("[MAL] updating state: [%d/%d] '%s' (MalID %d) status was '%s' and now is '%s'",
					index, len(c.watchList), getTitle(animeDetails), malID, oldStatus, animeDetails.Status)
			}
		} else {
			c.log.Debugf("[MAL] updating state: [%d/%d] '%s' (MalID %d) status '%s' is unchanged",
				index, len(c.watchList), getTitle(animeDetails), malID, oldStatus)
		}
		index++
	}
	return
}

func (c *Controller) findNewAnimes() (finished []*jikan.Anime) {
	var (
		seasonList   *jikan.Season
		animeDetails *jikan.Anime
		err          error
		found        bool
		new          int
	)
	// Get current season
	c.rateLimiter()
	if seasonList, err = jikan.GetSeason(currentSeason()); err != nil {
		c.log.Errorf("[MAL] finding new animes: can't get current season animes: %v", err)
		return
	}
	finished = make([]*jikan.Anime, 0, len(seasonList.Anime))
	// for each anime
	for _, anime := range seasonList.Anime {
		if _, found = c.watchList[anime.MalID]; found {
			continue
		}
		// new anime: get its status
		c.rateLimiter()
		if animeDetails, err = jikan.GetAnime(anime.MalID); err != nil {
			c.log.Errorf("[MAL] finding new animes: can't get details of a new anime ('%s' [%d]): %v",
				anime.Title, anime.MalID, err)
			continue
		}
		// save filters data
		c.update.Lock()
		for _, genre := range animeDetails.Genres {
			c.genres.Add(genre.Name)
		}
		c.ratings.Add(animeDetails.Rating)
		c.update.Unlock()
		// handle status
		if animeDetails.Status == animeStatusFinished {
			// we are cheating here as a fail safe: only process finished have the power to mark an anime finished (once notified or discarded)
			c.update.Lock()
			c.watchList[animeDetails.MalID] = animeStatusOnGoing
			c.update.Unlock()
			// add it to the finished tmp list for further processing
			finished = append(finished, animeDetails)
			c.log.Infof("[MAL] finding new animes: found an already finished anime: '%s' (MalID %d)",
				getTitle(animeDetails), animeDetails.MalID)
		} else {
			c.update.Lock()
			c.watchList[animeDetails.MalID] = animeDetails.Status
			c.update.Unlock()
			c.log.Debugf("[MAL] finding new animes: a new (%s) anime has been found: '%s' (MalID %d)",
				animeDetails.Status, getTitle(animeDetails), animeDetails.MalID)
		}
		new++
	}
	c.log.Infof("[MAL] finding new animes: %d new anime(s) added to the watch list", new)
	return
}

func (c *Controller) processFinished(finished []*jikan.Anime) {
	var (
		err error
		bl  []string
	)
	for _, anime := range finished {
		// filter out based on score
		if anime.Score < c.minScore {
			c.log.Infof("[MAL] processing finished animes: '%s' (MalID %d) does not have the require score (%.2f/%.2f): skipping",
				getTitle(anime), anime.MalID, anime.Score, c.minScore)
			c.update.Lock()
			c.watchList[anime.MalID] = animeStatusFinished
			c.update.Unlock()
			continue
		}
		// filter out based on genres
		if bl = c.getBlacklistedGenres(anime); len(bl) > 0 {
			c.log.Infof("[MAL] processing finished animes: '%s' (MalID %d) has the required score (%.2f/%.2f) but contains blacklisted genr(s): %s",
				getTitle(anime), anime.MalID, anime.Score, c.minScore, strings.Join(bl, ", "))
			c.update.Lock()
			c.watchList[anime.MalID] = animeStatusFinished
			c.update.Unlock()
			continue
		}
		// send the notification
		if err = c.pushover.SendCustomMsg(c.generateNotificationMsg(anime)); err != nil {
			c.log.Errorf("[MAL] processing finished animes: sending pushover notification failed for '%s' (MalID %d): %v",
				getTitle(anime), anime.MalID, err)
			// do not set its status to finish in order to have a chance to notify it again
		} else {
			c.log.Infof("[MAL] processing finished animes: pushover notification sent for '%s' (MalID %d)",
				getTitle(anime), anime.MalID)
			// notification sent successfully, we can mark it finished within the state
			c.update.Lock()
			c.watchList[anime.MalID] = animeStatusFinished
			c.update.Unlock()
		}
	}
}

func getTitle(anime *jikan.Anime) string {
	if anime.TitleEnglish != "" {
		return anime.TitleEnglish
	}
	return anime.Title
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
