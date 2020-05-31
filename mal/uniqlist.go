package mal

import (
	"encoding/json"
	"fmt"
)

type uniqList map[string]struct{}

func (ul uniqList) add(item string) {
	ul[item] = struct{}{}
}

func (ul *uniqList) MarshalJSON() (data []byte, err error) {
	// create the flat list
	flat := make([]string, len(*ul))
	index := 0
	for item := range *ul {
		flat[index] = item
		index++
	}
	// marshal it in place
	return json.Marshal(flat)
}

func (ul *uniqList) UnmarshalJSON(data []byte) (err error) {
	var flat []string
	if err := json.Unmarshal(data, &flat); err != nil {
		return fmt.Errorf("cannot unmarshal data wihtin the temporary flat list: %w", err)
	}
	*ul = make(uniqList, len(flat))
	for _, item := range flat {
		(*ul)[item] = struct{}{}
	}
	return
}
