package mal

import (
	"fmt"
	"time"
)

type malSeason string

const (
	winter malSeason = "winter"
	spring malSeason = "spring"
	summer malSeason = "summer"
	fall   malSeason = "fall"
)

func getCurrentSeason() (season malSeason, year int) {
	current := time.Now()
	year = current.Year()
	switch current.Month() {
	case 12, 1, 2:
		season = winter
	case 3, 4, 5:
		season = spring
	case 6, 7, 8:
		season = summer
	case 9, 10, 11:
		season = fall
	default:
		panic(fmt.Errorf("invalid month: %d", current.Month()))
	}
	return
}

func getPreviousSeason(season malSeason, year int) (malSeason, int) {
	if season == winter {
		return fall, year - 1
	}
	if season == spring {
		return winter, year
	}
	if season == summer {
		return spring, year
	}
	if season == fall {
		return summer, year
	}
	panic(fmt.Errorf("unknown season: %s", season))
}
