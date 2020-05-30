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

func (c *Controller) fetcher() {
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
	c.log.Debug("[MAL] starting new batch")
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
	oldFinished := c.updateState()
	// try to find new ones
	newFinished := c.findNewAnimes()
	// notify
	c.processFinished(append(oldFinished, newFinished...))
}

func (c *Controller) buildInitialList() (err error) {
	year, season := getCurrentSeason()
	var (
		seasonList   *jikan.Season
		animeDetails *jikan.Anime
		title        string
		previousLen  int
		ok           bool
	)
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
			// prefer english title if possible
			if animeDetails.TitleEnglish != "" {
				title = animeDetails.TitleEnglish
			} else {
				title = animeDetails.Title
			}
			// act depending on status
			switch animeDetails.Status {
			case animeStatusNotAired:
				fallthrough
			case animeStatusOnGoing:
				if _, ok = c.watchList[anime.MalID]; ok {
					c.log.Debugf("[MAL] building initial list: season %d/%d (%s %d): anime %d/%d: '%s' (MalID %d) state is '%s': already in the list",
						i+1, c.nbSeasons, season, year, index, len(seasonList.Anime), title, animeDetails.MalID, animeDetails.Status)
				} else {
					c.log.Debugf("[MAL] building initial list: season %d/%d (%s %d): anime %d/%d: '%s' (MalID %d) state is '%s': adding it to the list",
						i+1, c.nbSeasons, season, year, index, len(seasonList.Anime), title, animeDetails.MalID, animeDetails.Status)
					c.watchList[anime.MalID] = animeDetails.Status
				}
			case animeStatusFinished:
				c.log.Debugf("[MAL] building initial list: season %d/%d (%s %d): anime %d/%d: '%s' (MalID %d) state is '%s': skipping",
					i+1, c.nbSeasons, season, year, index, len(seasonList.Anime), title, animeDetails.MalID, animeDetails.Status)
			default:
				c.log.Warningf("[MAL] building initial list: season %d/%d (%s %d): anime %d/%d: '%s' (MalID %d) state is unknown ('%s'): skipping",
					i+1, c.nbSeasons, season, year, index, len(seasonList.Anime), title, animeDetails.MalID, animeDetails.Status)
			}
		}
		c.log.Infof("[MAL] building initial list: season %d/%d (%s %d): added %d/%d animes",
			i+1, c.nbSeasons, season, year, len(c.watchList)-previousLen, len(seasonList.Anime))
		// prepare for next run
		year, season = getPreviousSeason(season, year)
	}
	return
}

func (c *Controller) updateState() (finished []*jikan.Anime) {
	var (
		err          error
		animeDetails *jikan.Anime
		title        string
	)
	finished = make([]*jikan.Anime, 0, len(c.watchList))
	for malID, oldStatus := range c.watchList {
		// get current details
		c.rateLimiter()
		if animeDetails, err = jikan.GetAnime(malID); err != nil {
			c.log.Errorf("[MAL] updating state: can't check current status of MalID %d: %s", malID, err)
			continue
		}
		// prefer english title if possible
		if animeDetails.TitleEnglish != "" {
			title = animeDetails.TitleEnglish
		} else {
			title = animeDetails.Title
		}
		// has status changed ?
		if animeDetails.Status != oldStatus {
			// save the new status
			c.watchList[malID] = animeDetails.Status
			// and act on it
			if animeDetails.Status == animeStatusFinished {
				finished = append(finished, animeDetails)
				c.log.Debugf("[MAL] updating state: '%s' (MalID %d) is now finished",
					title, malID, oldStatus, animeDetails.Status)
			} else {
				c.log.Debugf("[MAL] updating state: '%s' (MalID %d) status was '%s' and now is '%s'",
					title, malID, oldStatus, animeDetails.Status)
			}
		} else {
			c.log.Debugf("[MAL] updating state: '%s' (MalID %d) status '%s' is unchanged",
				title, malID, oldStatus)
		}
	}
	return
}

func (c *Controller) findNewAnimes() (finished []*jikan.Anime) {
	return
}

func (c *Controller) processFinished(finished []*jikan.Anime) {

}
