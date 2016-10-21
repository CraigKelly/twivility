package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSieve - simple unit testing for our prime factoring function
func TestTwitterJSON(t *testing.T) {
	assert := assert.New(t)

	makeOne := func(tid int64, uid int64, s string) TweetRecord {
		return TweetRecord{
			TweetID:   tid,
			UserID:    uid,
			UserName:  s,
			Text:      s,
			Timestamp: s,
		}
	}

	input := TweetRecordList{
		makeOne(1, 3, "C"),
		makeOne(2, 2, "B"),
		makeOne(3, 1, "A"),
	}

	data := CreateTwitterJSON(input)

	output := TwitterJSON{TweetList: make([]TweetRecord, 0)}
	json.Unmarshal(data, &output)

	assert.Equal(3, len(output.TweetList))

	assert.Equal(input, output.TweetList)
}
