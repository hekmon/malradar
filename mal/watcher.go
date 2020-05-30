package mal

import (
	"fmt"
	"time"

	"github.com/darenliang/jikan-go"
)

const (
	fetchFreq           = 24 * time.Hour
	animeStatusOnGoing  = "Currently Airing"
	animeStatusFinished = "Finished Airing"
	animeStatusNotAired = "Not yet aired"
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
	// // update state of known anime
	// oldFinished := c.updateCurrentState()
	// // try to find new ones
	// newFinished := c.findNewAnimes()
	// // notify
	// c.processFinished(append(oldFinished, newFinished...))
	c.processFinished(nil)
}

func (c *Controller) buildInitialList() (err error) {
	var (
		seasonList   *jikan.Season
		animeDetails *jikan.Anime
		previousLen  int
		ok           bool
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
			// get its details
			c.rateLimiter()
			if animeDetails, err = jikan.GetAnime(anime.MalID); err != nil {
				err = fmt.Errorf("iteration %d (%s %d): failing to acquire anime %d details: %w",
					i+1, season, year, anime.MalID, err)
				return
			}
			// act depending on status
			switch animeDetails.Status {
			case animeStatusNotAired:
				fallthrough
			case animeStatusOnGoing:
				if _, ok = c.watchList[anime.MalID]; ok {
					c.log.Debugf("[MAL] building initial list: season %d/%d (%s %d): anime %d/%d: '%s' (MalID %d) state is '%s': already in the list",
						i+1, c.nbSeasons, season, year, index, len(seasonList.Anime), getTitle(animeDetails), animeDetails.MalID, animeDetails.Status)
				} else {
					c.log.Debugf("[MAL] building initial list: season %d/%d (%s %d): anime %d/%d: '%s' (MalID %d) state is '%s': adding it to the list",
						i+1, c.nbSeasons, season, year, index, len(seasonList.Anime), getTitle(animeDetails), animeDetails.MalID, animeDetails.Status)
					c.watchList[anime.MalID] = animeDetails.Status
				}
			case animeStatusFinished:
				c.log.Debugf("[MAL] building initial list: season %d/%d (%s %d): anime %d/%d: '%s' (MalID %d) state is '%s': skipping",
					i+1, c.nbSeasons, season, year, index, len(seasonList.Anime), getTitle(animeDetails), animeDetails.MalID, animeDetails.Status)
			default:
				c.log.Warningf("[MAL] building initial list: season %d/%d (%s %d): anime %d/%d: '%s' (MalID %d) state is unknown ('%s'): skipping",
					i+1, c.nbSeasons, season, year, index, len(seasonList.Anime), getTitle(animeDetails), animeDetails.MalID, animeDetails.Status)
			}
		}
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
		// get current details
		c.rateLimiter()
		if animeDetails, err = jikan.GetAnime(malID); err != nil {
			c.log.Errorf("[MAL] updating state: [%d/%d] can't check current status of MalID %d: %s",
				index, len(c.watchList), malID, err)
			continue
		}
		// has status changed ?
		if animeDetails.Status != oldStatus {
			// save the new status
			c.watchList[malID] = animeDetails.Status
			// and act on it
			if animeDetails.Status == animeStatusFinished {
				finished = append(finished, animeDetails)
				c.log.Debugf("[MAL] updating state: [%d/%d] '%s' (MalID %d) is now finished",
					index, len(c.watchList), getTitle(animeDetails), malID)
			} else {
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
		// add it
		if animeDetails.Status == animeStatusFinished {
			c.log.Debugf("[MAL] finding new animes: skipped an already finished anime: '%s' (MalID %d)",
				getTitle(animeDetails), animeDetails.MalID)
		} else {
			c.watchList[animeDetails.MalID] = animeDetails.Status
			c.log.Debugf("[MAL] finding new animes: a new (%s) anime has been found: '%s' (MalID %d)",
				animeDetails.Status, getTitle(animeDetails), animeDetails.MalID)
		}
		new++
	}
	c.log.Infof("[MAL] finding new animes: %d new anime(s) added to the watch list", new)
	return
}

func (c *Controller) processFinished(finished []*jikan.Anime) {
	// tmp for debug
	if len(finished) == 0 {
		if animeDetails, err := jikan.GetAnime(777); err != nil {
			c.log.Errorf("[MAL] processFinished: DEBUG: can't get details of anime: %v",
				err)
		} else {
			finished = append(finished, animeDetails)
		}
	}
	var err error
	for _, anime := range finished {
		// send the notification
		if err = c.pushover.SendCustomMsg(c.generateNotificationMsg(anime)); err != nil {
			c.log.Errorf("[MAL] processing finished animes: sending pushover notification failed for '%s' (MalID %d): %v",
				getTitle(anime), anime.MalID, err)
		} else {
			c.log.Infof("[MAL] processing finished animes: pushover notification sent for '%s' (MalID %d)",
				getTitle(anime), anime.MalID)
			// notification sent successfully, we can remove it from the state
			delete(c.watchList, anime.MalID)
		}
	}
}

func getTitle(anime *jikan.Anime) string {
	if anime.TitleEnglish != "" {
		return anime.TitleEnglish
	}
	return anime.Title
}
