package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// Configuration holds the user configuration
type Configuration struct {
	MAL struct {
		MinScore   float64 `json:"minimum_score"`
		User       string  `json:"user_to_check_against"`
		Blacklists struct {
			Genres []string `json:"genres"`
			Types  []string `json:"types"`
		} `json:"blacklists"`
		Init struct {
			NbSeasons int  `json:"nb_of_seasons_to_scrape"`
			Notify    bool `json:"notify_on_first_run"`
		} `json:"initialization"`
	} `json:"myanimelist"`
	Pushover struct {
		UserKey        string `json:"user_key"`
		ApplicationKey string `json:"application_key"`
	} `json:"pushover"`
}

func getConfig(path string) (conf Configuration, err error) {
	// Open file
	var configFile *os.File
	if configFile, err = os.Open(path); err != nil {
		err = fmt.Errorf("can't open '%s' for reading: %w", path, err)
		return
	}
	defer configFile.Close()
	// Parse it
	if err = json.NewDecoder(configFile).Decode(&conf); err != nil {
		err = fmt.Errorf("can't decode '%s' as JSON: %w", path, err)
		return
	}
	// Check values
	if conf.Pushover.ApplicationKey == "" {
		err = errors.New("pushover application key must be set")
		return
	}
	if conf.Pushover.UserKey == "" {
		err = errors.New("pushover user key must be set")
		return
	}
	return
}
