package mal

import (
	"fmt"
	"time"
)

const (
	winter string = "winter"
	spring string = "spring"
	summer string = "summer"
	fall   string = "fall"
)

func getCurrentSeason() (year int, season string) {
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

func getPreviousSeason(season string, year int) (int, string) {
	if season == winter {
		return year - 1, fall
	}
	if season == spring {
		return year, winter
	}
	if season == summer {
		return year, spring
	}
	if season == fall {
		return year, summer
	}
	panic(fmt.Errorf("unknown season: %s", season))
}
