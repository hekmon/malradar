package mal

import (
	"fmt"
	"time"

	"github.com/darenliang/jikan-go"
)

const (
	fetchFreq          = 24 * time.Hour
	animeStatusOnGoing = "TBA"
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
			c.log.Info("[MAL] [Fetcher] context done: stopping worker")
			return
		}
	}
}

func (c *Controller) batch() {
	start := time.Now()
	c.log.Debug("[MAL] [Fetcher] starting new batch")
	// first run ever ?
	if c.watchList == nil {
		c.log.Infof("[MAL] [Fetcher] initializing watch list")
		if err := c.buildInitialList(); err != nil {
			c.watchList = nil
			c.log.Errorf("[MAL] [Fetcher] failed to build initial list: %v", err)
		}
		return
	}
	// fetch current state
	// TODO
	// compare
	// TODO
	c.log.Infof("[MAL] [Fetcher] batch executed in %v", time.Since(start))
}

func (c *Controller) buildInitialList() (err error) {
	season, year := getCurrentSeason()
	var (
		seasonList   *jikan.Season
		animeDetails *jikan.Anime
		added        int
	)
	for i := 0; i < c.nbSeasons; i++ {
		c.log.Debugf("[MAL] [Fetcher] building initial list: season %d/%d (%s %d)",
			i+1, c.nbSeasons, season, year)
		// get season list
		c.rateLimiter()
		if seasonList, err = jikan.GetSeason(year, string(season)); err != nil {
			err = fmt.Errorf("iteration %d (%s %d): failing to acquire season animes: %w",
				i+1, season, year, err)
			return
		}
		if c.watchList == nil {
			c.watchList = make(map[int]*jikan.Anime, c.nbSeasons*len(seasonList.Anime))
		}
		// get details
		added = 0
		for index, anime := range seasonList.Anime {
			c.rateLimiter()
			if animeDetails, err = jikan.GetAnime(anime.MalID); err != nil {
				err = fmt.Errorf("iteration %d (%s %d): failing to acquire anime %d details: %w",
					i+1, season, year, anime.MalID, err)
				return
			}
			switch animeDetails.Status {
			case animeStatusOnGoing:
				c.log.Debugf("[MAL] [Fetcher] building initial list: season %d/%d (%s %d): anime %d/%d: '%s' (ID %d) is %s: adding it to the list",
					i+1, c.nbSeasons, season, year, index, len(seasonList.Anime), animeDetails.TitleEnglish, animeDetails.MalID, animeDetails.Status)
				c.watchList[anime.MalID] = animeDetails
				added++
			default:
				c.log.Debugf("[MAL] [Fetcher] building initial list: season %d/%d (%s %d): anime %d/%d: '%s' (ID %d) is %s: skipping",
					i+1, c.nbSeasons, season, year, index, len(seasonList.Anime), animeDetails.TitleEnglish, animeDetails.MalID, animeDetails.Status)
			}
		}
		c.log.Infof("[MAL] [Fetcher] building initial list: season %d/%d (%s %d): added %d/%d animes",
			i+1, c.nbSeasons, season, year, added, len(seasonList.Anime))
		// prepare for next run
		season, year = getPreviousSeason(season, year)
	}
	return
}
