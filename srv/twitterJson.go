package main

import (
	"encoding/json"
)

// Provide JSON-export capability for records read from twitterFile
type twitterJSON struct {
	TweetList FileRecordSlice
}

// CreateTwitterJSON creates and returns a JSON string of the given records
func CreateTwitterJSON(recs FileRecordSlice) []byte {
	topLevel := twitterJSON{
		TweetList: recs,
	}
	txt, _ := json.Marshal(topLevel)
	return txt
}
