package mal

import (
	"encoding/json"
	"errors"
	"os"
)

const (
	file = "malwatcher_state.json"
)

func (c *Controller) load() (proceed bool) {
	// handle file descriptor
	stateFile, err := os.Open(file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			proceed = true
		} else {
			c.log.Errorf("[MAL] can't open state file: %v", err)
		}
		return
	}
	defer stateFile.Close()
	// handle content
	if err = json.NewDecoder(stateFile).Decode(&c.watchList); err != nil {
		c.log.Errorf("[MAL] can't parse state file: %v", err)
		return
	}
	proceed = true
	return
}

func (c *Controller) save() {
	if len(c.watchList) == 0 {
		return
	}
	// handle file descriptor
	stateFile, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
	if err != nil {
		c.log.Errorf("[MAL] can't open state file: %v", err)
		return
	}
	defer stateFile.Close()
	// handle content
	if err = json.NewEncoder(stateFile).Encode(c.watchList); err != nil {
		c.log.Errorf("[MAL] can't write state to file: %v", err)
		return
	}
	return
}
