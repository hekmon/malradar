package radar

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

const (
	stateFile   = "animes_state.json"
	genresFile  = "encountered_genres.json"
	ratingsFile = "encountered_ratings.json"
	typesFile   = "encountered_types.json"
)

func (c *Controller) load(file string) (proceed bool) {
	// prepare
	var (
		log    string
		target interface{}
	)
	switch file {
	case stateFile:
		log = "state"
		// do not make the map here as nil is used to start the initial building
		target = &c.watchList
	case genresFile:
		log = "genres"
		c.genres = make(UniqList)
		target = &c.genres
	case ratingsFile:
		log = "ratings"
		c.ratings = make(UniqList)
		target = &c.ratings
	case typesFile:
		log = "types"
		c.types = make(UniqList)
		target = &c.types
	default:
		panic(fmt.Sprintf("persistent save received an unknown file: %s", file))
	}
	// handle file descriptor
	fd, err := os.Open(file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			proceed = true
		} else {
			c.log.Errorf("[MAL] can't open %s file: %v", log, err)
		}
		return
	}
	defer fd.Close()
	// handle content
	if err = json.NewDecoder(fd).Decode(target); err != nil {
		c.log.Errorf("[MAL] can't parse %s file: %v", log, err)
		return
	}
	c.log.Infof("[MAL] %s loaded from %s", log, file)
	proceed = true
	return
}

func (c *Controller) save(file string) {
	// prepare
	var (
		log    string
		source interface{}
	)
	switch file {
	case stateFile:
		if len(c.watchList) == 0 {
			// next run will need to build the initial list
			c.log.Debug("[MAL] saving state skipped: next start must initial list building")
			return
		}
		log = "state"
		source = c.watchList
	case genresFile:
		log = "genres"
		source = c.genres
	case ratingsFile:
		log = "ratings"
		source = c.ratings
	case typesFile:
		log = "types"
		source = c.types
	default:
		panic(fmt.Sprintf("persistent load received an unknown file: %s", file))
	}
	// handle file descriptor
	fd, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
	if err != nil {
		c.log.Errorf("[MAL] can't open %s file: %v", log, err)
		return
	}
	defer fd.Close()
	// handle content
	if err = json.NewEncoder(fd).Encode(source); err != nil {
		c.log.Errorf("[MAL] can't write %s to file: %v", log, err)
		return
	}
	c.log.Infof("[MAL] %s saved to %s", log, file)
	return
}
