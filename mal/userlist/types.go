package userlist

// List handles Anime list with handfull methods
type List []Anime

// Contains return true if an mal anime ID is present within the list
func (l List) Contains(id int) bool {
	for _, anime := range l {
		if anime.AnimeID == id {
			return true
		}
	}
	return false
}

// FilterStatus returns a filtered list of animes with only one status
func (l List) FilterStatus(status Status) (filtered List) {
	filtered = make(List, 0, len(l))
	for _, anime := range l {
		if anime.Status == status {
			filtered = append(filtered, anime)
		}
	}
	return
}

// Anime describe an anime item within a user list
type Anime struct {
	Status             Status `json:"status"`
	Score              int    `json:"score"`
	Tags               string `json:"tags"`
	IsRewatching       int    `json:"is_rewatching"`
	NumWatchedEpisodes int    `json:"num_watched_episodes"`
	AnimeTitle         string `json:"anime_title"`
	AnimeNumEpisodes   int    `json:"anime_num_episodes"`
	AnimeAiringStatus  int    `json:"anime_airing_status"`
	AnimeID            int    `json:"anime_id"`
	// AnimeStudios          interface{} `json:"anime_studios"`
	// AnimeLicensors        interface{} `json:"anime_licensors"`
	// AnimeSeason           interface{} `json:"anime_season"`
	HasEpisodeVideo       bool   `json:"has_episode_video"`
	HasPromotionVideo     bool   `json:"has_promotion_video"`
	HasVideo              bool   `json:"has_video"`
	VideoURL              string `json:"video_url"`
	AnimeURL              string `json:"anime_url"`
	AnimeImagePath        string `json:"anime_image_path"`
	IsAddedToList         bool   `json:"is_added_to_list"`
	AnimeMediaTypeString  string `json:"anime_media_type_string"`
	AnimeMpaaRatingString string `json:"anime_mpaa_rating_string"`
	// StartDateString       interface{} `json:"start_date_string"`
	// FinishDateString      interface{} `json:"finish_date_string"`
	AnimeStartDateString string `json:"anime_start_date_string"`
	AnimeEndDateString   string `json:"anime_end_date_string"`
	// DaysString           interface{} `json:"days_string"`
	StorageString  string `json:"storage_string"`
	PriorityString string `json:"priority_string"`
}

// Status represents an anime status for a given user
type Status int

const (
	// StatusWatching represents the 'Watching' status for an anime in a user list
	StatusWatching Status = 1
	// StatusCompleted represents the 'Completed' status for an anime in a user list
	StatusCompleted Status = 2
	// StatusOnHold represents the 'On Hold' status for an anime in a user list
	StatusOnHold Status = 3
	// StatusDropped represents the 'Dropped' status for an anime in a user list
	StatusDropped Status = 4
	// StatusPlanToWatch represents the 'Plan to Watch' status for an anime in a user list
	StatusPlanToWatch Status = 6
	// StatusAll represents all the possible status for an anime in a user list (no filtering)
	StatusAll Status = 7
)
