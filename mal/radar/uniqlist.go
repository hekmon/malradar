package radar

import (
	"encoding/json"
	"fmt"
)

// UniqList allows to uniquely store values while translating from and to JSON as a regular list
type UniqList map[string]struct{}

// Add allow to add an item on the list
func (ul UniqList) Add(item string) {
	ul[item] = struct{}{}
}

// MarshalJSON transform the list as a regular JSON array
func (ul UniqList) MarshalJSON() (data []byte, err error) {
	// create the flat list
	flat := make([]string, len(ul))
	index := 0
	for item := range ul {
		flat[index] = item
		index++
	}
	// marshal it in place
	return json.Marshal(flat)
}

// UnmarshalJSON allows to transform a regular JSON array as a uniq lsit
func (ul UniqList) UnmarshalJSON(data []byte) (err error) {
	var flat []string
	if err := json.Unmarshal(data, &flat); err != nil {
		return fmt.Errorf("cannot unmarshal data wihtin the temporary flat list: %w", err)
	}
	ul = make(UniqList, len(flat))
	for _, item := range flat {
		ul[item] = struct{}{}
	}
	return
}
