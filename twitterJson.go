package main

import (
	"encoding/json"
)

// TwitterJSON provides JSON-export capability for records read from twitterFile
type TwitterJSON struct {
	TweetList TweetRecordList
}

// CreateTwitterJSON creates and returns a JSON string of the given records
func CreateTwitterJSON(recs TweetRecordList) []byte {
	topLevel := TwitterJSON{
		TweetList: recs,
	}
	txt, _ := json.Marshal(topLevel)
	return txt
}
