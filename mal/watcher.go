package mal

import (
	"fmt"
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
	// update state of known animes & process the finished one
	c.updateCurrentState()
	// try to find new ones
	c.findNewAnimes()
	// try to recover old finished to sent yet
	c.recoverOldFinished()
}

func (c *Controller) buildInitialList() (err error) {
	var (
		seasonList   *jikan.Season
		animeDetails *jikan.Anime
		previousLen  int
		found        bool
		finished     []*jikan.Anime
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
			if animeDetails.Status == animeStatusFinished {
				finished = append(finished, animeDetails)
			}
			c.log.Debugf("[MAL] building initial list: season %d/%d (%s %d): anime %d/%d: '%s' (MalID %d) with '%s' state",
				i+1, c.nbSeasons, season, year, index, len(seasonList.Anime), getTitle(animeDetails), animeDetails.MalID, animeDetails.Status)
		}
		// season done
		c.log.Infof("[MAL] building initial list: season %d/%d (%s %d): added %d/%d animes",
			i+1, c.nbSeasons, season, year, len(c.watchList)-previousLen, len(seasonList.Anime))
		// prepare for next run
		year, season = previousSeason(season, year)
	}
	// send all the finished animes discovered
	c.log.Infof("[MAL] building initial list: now tracking %s animes, %d ready to be notified",
		len(c.watchList)-len(finished), len(finished))
	for _, anime := range finished {
		c.pipeline <- anime
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
			c.update.Lock()
			c.watchList[malID] = animeDetails.Status
			c.update.Unlock()
			if animeDetails.Status == animeStatusFinished {
				c.log.Infof("[MAL] updating state: [%d/%d] '%s' (MalID %d) is now finished",
					index, len(c.watchList), getTitle(animeDetails), malID)
				// send it to the notifier
				go func() {
					c.pipeline <- animeDetails
				}()
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

func (c *Controller) findNewAnimes() {
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
		c.log.Errorf("[MAL] finding new animes (current season): can't get current season animes: %v", err)
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
			c.log.Errorf("[MAL] finding new animes (current season): can't get details of a new anime ('%s' [%d]): %v",
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
		if animeDetails.Status != animeStatusFinished {
			c.update.Lock()
			c.watchList[animeDetails.MalID] = animeDetails.Status
			c.update.Unlock()
			c.log.Debugf("[MAL] finding new animes (current season): a new (%s) anime has been found: '%s' (MalID %d)",
				animeDetails.Status, getTitle(animeDetails), animeDetails.MalID)
		} else {
			c.log.Infof("[MAL] finding new animes (current season): skipping an already finished anime: '%s' (MalID %d)",
				getTitle(animeDetails), animeDetails.MalID)
		}
		new++
	}
	c.log.Infof("[MAL] finding new animes (current season): %d new anime(s) added to the watch list", new)
	return
}

func (c *Controller) recoverOldFinished() {
	// // try to recover of unprocessed finished animes
	// c.update.Lock()
	// finishedIDs := make([]int, 0, len(c.watchList))
}
