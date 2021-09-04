package radar

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
	errorRetryMax       = 5
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
			c.log.Info("[MAL] [Watcher] context done: stopping worker")
			return
		}
	}
}

func (c *Controller) batch() {
	start := time.Now()
	c.log.Info("[MAL] [Watcher] starting new batch")
	defer func() {
		c.log.Infof("[MAL] [Watcher] batch executed in %v", time.Since(start))
	}()
	var (
		err      error
		finished []*jikan.Anime
	)
	// first run or state update ?
	if c.watchList == nil {
		if finished, err = c.buildInitialList(); err != nil {
			c.watchList = nil
			c.log.Errorf("[MAL] [Watcher] failed to build initial list: %v", err)
			return
		}
	} else {
		// try to recover previously finished animes not notified
		finished = c.recoverOldFinished()
		// update state of known animes & process the finished one
		finished = append(finished, c.updateCurrentState()...)
		// try to find new ones
		c.findNewAnimes()
	}
	// notify
	c.batchNotifier(finished)
}

func (c *Controller) buildInitialList() (finished []*jikan.Anime, err error) {
	var notifinit string
	if c.notifyInit {
		notifinit = "backlog notifications activated"
	} else {
		notifinit = "no backlog notifications"
	}
	c.log.Infof("[MAL] [Watcher] building initial list with a %d season(s) backlog and %s",
		c.nbSeasons, notifinit)
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
		c.log.Infof("[MAL] [Watcher] building initial list: season %d/%d (%s %d): fetching details for %d animes...",
			i+1, c.nbSeasons, season, year, len(seasonList.Anime))
		if c.watchList == nil {
			c.watchList = make(map[int]string, c.nbSeasons*len(seasonList.Anime)*3/2) // ×1.5
		}
		// for each anime
		for index, anime := range seasonList.Anime {
			// do we have it from an earlier season ?
			if _, found = c.watchList[anime.MalID]; found {
				c.log.Debugf("[MAL] [Watcher] building initial list: season %d/%d (%s %d): anime %d/%d: '%s' (MalID %d): already in the list",
					i+1, c.nbSeasons, season, year, index, len(seasonList.Anime), anime.Title, anime.MalID, animeDetails.Status)
				continue
			}
			// get its details
			try := 0
			for {
				// sometime the Jikkan API can have issues, we will retry until errorRetryMax is reached
				try++
				c.rateLimiter()
				if animeDetails, err = jikan.GetAnime(anime.MalID); err == nil {
					// no error let's get out of the loop
					if try > 1 {
						c.log.Infof("[MAL] [Watcher] building initial list: season %d/%d (%s %d): anime %d details recovered at try %d/%d",
							i+1, c.nbSeasons, season, year, anime.MalID, try, errorRetryMax)
					}
					break
				}
				if try == errorRetryMax {
					err = fmt.Errorf("iteration %d (%s %d): failed to acquire anime %d details (try %d/%d): %w",
						i+1, season, year, anime.MalID, try, errorRetryMax, err)
					return
				}
				// let's retry when rateLimiter will allow us to
				c.log.Warningf("[MAL] [Watcher] building initial list: season %d/%d (%s %d): failed to acquire anime %d details (try %d/%d): %v",
					i+1, c.nbSeasons, season, year, anime.MalID, try, errorRetryMax, err)
			}
			// save data
			c.update.Lock()
			for _, genre := range animeDetails.Genres {
				c.genres.Add(genre.Name)
			}
			c.ratings.Add(animeDetails.Rating)
			c.types.Add(animeDetails.Type)
			if animeDetails.Status == animeStatusFinished {
				if c.notifyInit {
					finished = append(finished, animeDetails)
					c.watchList[anime.MalID] = animeDetails.Status
				}
				// else skip
			} else {
				c.watchList[anime.MalID] = animeDetails.Status
			}
			c.update.Unlock()
			c.log.Debugf("[MAL] [Watcher] building initial list: season %d/%d (%s %d): anime %d/%d: '%s' (MalID %d) with '%s' state",
				i+1, c.nbSeasons, season, year, index, len(seasonList.Anime), getTitle(animeDetails), animeDetails.MalID, animeDetails.Status)
		}
		// season done
		c.log.Infof("[MAL] [Watcher] building initial list: season %d/%d (%s %d): added %d/%d animes",
			i+1, c.nbSeasons, season, year, len(c.watchList)-previousLen, len(seasonList.Anime))
		// prepare for next run
		year, season = previousSeason(season, year)
	}
	// send all the finished animes discovered
	c.log.Infof("[MAL] [Watcher] building initial list: now tracking %d animes, %d '%s' to be processed",
		len(c.watchList)-len(finished), len(finished), animeStatusFinished)
	return
}

func (c *Controller) recoverOldFinished() (finished []*jikan.Anime) {
	c.log.Debugf("[MAL] [Watcher] recover old finished: checking %d animes...", len(c.watchList))
	var (
		err          error
		animeDetails *jikan.Anime
	)
	finished = make([]*jikan.Anime, 0, len(c.watchList))
	index := 1
	// try to recover of notified finished animes
anime:
	for malID, status := range c.watchList {
		if status == animeStatusFinished {
			// Get details
			try := 0
			for {
				// sometime the Jikkan API can have issues, we will retry until errorRetryMax is reached
				try++
				c.rateLimiter()
				if animeDetails, err = jikan.GetAnime(malID); err == nil {
					// no error let's get out of the loop
					if try > 1 {
						c.log.Infof("[MAL] [Watcher] recover old finished: [%d/%d] anime %d details recovered at try %d/%d",
							index, len(c.watchList), malID, try, errorRetryMax)
					}
					break
				}
				if try == errorRetryMax {
					c.log.Errorf("[MAL] [Watcher] recover old finished: [%d/%d] can't check current status of MalID %d (try %d/%d): %s",
						index, len(c.watchList), malID, try, errorRetryMax, err)
					continue anime
				}
				// let's retry when rateLimiter will allow us to
				c.log.Warningf("[MAL] [Watcher] recover old finished: [%d/%d] can't check current status of MalID %d (try %d/%d): %s",
					index, len(c.watchList), malID, try, errorRetryMax, err)
			}
			// save it for notification
			finished = append(finished, animeDetails)
		}
		index++
	}
	if len(finished) == 0 {
		c.log.Debug("[MAL] [Watcher] recover old finished: no tracked animes need recovery")
	} else {
		c.log.Infof("[MAL] [Watcher] recover old finished: recovered %d finished animes not yet notified", len(finished))
	}
	return
}

func (c *Controller) updateCurrentState() (finished []*jikan.Anime) {
	c.log.Infof("[MAL] [Watcher] updating state: refreshing %d animes...", len(c.watchList))
	var (
		err          error
		animeDetails *jikan.Anime
	)
	finished = make([]*jikan.Anime, 0, len(c.watchList))
	index := 1
anime:
	for malID, oldStatus := range c.watchList {
		// only update the ones which need to
		if oldStatus == animeStatusFinished {
			continue
		}
		// get current details
		try := 0
		for {
			// sometime the Jikkan API can have issues, we will retry until errorRetryMax is reached
			try++
			c.rateLimiter()
			if animeDetails, err = jikan.GetAnime(malID); err == nil {
				// no error let's get out of the loop
				if try > 1 {
					c.log.Infof("[MAL] [Watcher] updating state: [%d/%d] anime %d details recovered at try %d/%d",
						index, len(c.watchList), malID, try, errorRetryMax)
				}
				break
			}
			if try == errorRetryMax {
				c.log.Errorf("[MAL] [Watcher] updating state: [%d/%d] can't check current status of MalID %d (try %d/%d): %s",
					index, len(c.watchList), malID, try, errorRetryMax, err)
				continue anime
			}
			// let's retry when rateLimiter will allow us to
			c.log.Warningf("[MAL] [Watcher] updating state: [%d/%d] can't check current status of MalID %d (try %d/%d): %s",
				index, len(c.watchList), malID, try, errorRetryMax, err)
		}
		// save filters data
		c.update.Lock()
		for _, genre := range animeDetails.Genres {
			c.genres.Add(genre.Name)
		}
		c.ratings.Add(animeDetails.Rating)
		c.types.Add(animeDetails.Type)
		c.update.Unlock()
		// has status changed ?
		if animeDetails.Status != oldStatus {
			c.update.Lock()
			c.watchList[malID] = animeDetails.Status
			c.update.Unlock()
			if animeDetails.Status == animeStatusFinished {
				finished = append(finished, animeDetails)
				c.log.Infof("[MAL] [Watcher] updating state: [%d/%d] '%s' (MalID %d) is now finished",
					index, len(c.watchList), getTitle(animeDetails), malID)
			} else {
				c.log.Debugf("[MAL] [Watcher] updating state: [%d/%d] '%s' (MalID %d) status was '%s' and now is '%s'",
					index, len(c.watchList), getTitle(animeDetails), malID, oldStatus, animeDetails.Status)
			}
		} else {
			c.log.Debugf("[MAL] [Watcher] updating state: [%d/%d] '%s' (MalID %d) status '%s' is unchanged",
				index, len(c.watchList), getTitle(animeDetails), malID, oldStatus)
		}
		index++
	}
	return
}

func (c *Controller) findNewAnimes() {
	c.log.Info("[MAL] [Watcher] finding new animes (current season)...")
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
		c.log.Errorf("[MAL] [Watcher] finding new animes (current season): can't get current season animes: %v", err)
		return
	}
	// for each anime for this season
	for _, anime := range seasonList.Anime {
		if _, found = c.watchList[anime.MalID]; found {
			continue
		}
		// get its status
		c.rateLimiter()
		if animeDetails, err = jikan.GetAnime(anime.MalID); err != nil {
			c.log.Errorf("[MAL] [Watcher] finding new animes (current season): can't get details of a new anime ('%s' [%d]): %v",
				anime.Title, anime.MalID, err)
			continue
		}
		// save filters data
		c.update.Lock()
		for _, genre := range animeDetails.Genres {
			c.genres.Add(genre.Name)
		}
		c.ratings.Add(animeDetails.Rating)
		c.types.Add(animeDetails.Type)
		c.update.Unlock()
		// handle status
		if animeDetails.Status != animeStatusFinished {
			c.update.Lock()
			c.watchList[animeDetails.MalID] = animeDetails.Status
			c.update.Unlock()
			new++
			c.log.Infof("[MAL] [Watcher] finding new animes (current season): a new (%s) anime has been found: '%s' (MalID %d)",
				animeDetails.Status, getTitle(animeDetails), animeDetails.MalID)
		} else {
			c.log.Infof("[MAL] [Watcher] finding new animes (current season): skipping an already finished anime: '%s' (MalID %d)",
				getTitle(animeDetails), animeDetails.MalID)
		}
	}
	c.log.Infof("[MAL] [Watcher] finding new animes (current season): %d/%d new anime(s) added to the watch list",
		new, len(seasonList.Anime))
}
