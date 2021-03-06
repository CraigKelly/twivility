package main

import (
	"encoding/gob"
	"io"
	"os"
	"sort"

	"github.com/dghubble/go-twitter/twitter"
)

// TweetRecord is the struct we store in a file for a single tweet
type TweetRecord struct {
	TweetID        int64
	UserID         int64
	UserScreenName string
	UserName       string
	Text           string
	Timestamp      string
	FavoriteCount  int
	RetweetCount   int
	Hashtags       []string
	Mentions       []string
	IsRetweet      bool
}

// NewTweetRecord builds our nice record from the 'actual' API record
func NewTweetRecord(tweet *twitter.Tweet) TweetRecord {
	txt := tweet.Text
	isRetweet := false
	if tweet.RetweetedStatus != nil && len(tweet.RetweetedStatus.Text) > 0 {
		// Use the actual retweed text since twitter likes to trunc the text
		// in the RT
		txt = tweet.RetweetedStatus.Text
		isRetweet = true
	}

	// Note that we manually match the entities we want
	return TweetRecord{
		TweetID:        tweet.ID,
		UserID:         tweet.User.ID,
		UserName:       tweet.User.Name,
		UserScreenName: tweet.User.ScreenName,
		Text:           txt,
		Timestamp:      tweet.CreatedAt,
		FavoriteCount:  tweet.FavoriteCount,
		RetweetCount:   tweet.RetweetCount,
		Hashtags:       allNonBlank(hashtagMatch.FindAllString(txt, -1)),
		Mentions:       allNonBlank(userMatch.FindAllString(txt, -1)),
		IsRetweet:      isRetweet,
	}
}

// TweetRecordList is a slice of TweetFileRecords
type TweetRecordList []TweetRecord

// sort.Interface
func (frs TweetRecordList) Len() int           { return len(frs) }
func (frs TweetRecordList) Swap(i, j int)      { frs[i], frs[j] = frs[j], frs[i] }
func (frs TweetRecordList) Less(i, j int) bool { return frs[i].TweetID < frs[j].TweetID }

// MinMax return the min and max tweet ID for the given slice. REQUIRES sorted slice
func (frs TweetRecordList) MinMax() (mn int64, mx int64) {
	ln := len(frs)
	if ln < 1 {
		return 0, 0
	}
	return frs[ln-1].TweetID, frs[0].TweetID
}

// Seen returns a map of tweet ID's that have been seen
func (frs TweetRecordList) Seen() map[int64]bool {
	seen := make(map[int64]bool)
	for _, tweet := range frs {
		seen[tweet.TweetID] = true
	}
	return seen
}

// SortTwitterRecords sorts the given TweetFileRecordSlice INPLACE in our "canonical" order
func SortTwitterRecords(frs TweetRecordList) {
	sort.Sort(sort.Reverse(frs))
}

// WriteTwitterFile writes the file - note that the slice is sorted (and therefore mutated)
func (frs TweetRecordList) WriteTwitterFile(filename string) {
	output, err := os.Create(filename)
	pcheck(err)
	defer SafeClose(output)

	SortTwitterRecords(frs)

	enc := gob.NewEncoder(output)
	for _, obj := range frs {
		err := enc.Encode(obj)
		pcheck(err)
	}
}

// ReadTwitterFile reads the specified file name for our twitter records
func ReadTwitterFile(filename string) TweetRecordList {
	input, err := os.Open(filename)
	pcheck(err)
	defer SafeClose(input)

	dec := gob.NewDecoder(input)
	records := make([]TweetRecord, 0, 512)

	for {
		rec := TweetRecord{}
		err := dec.Decode(&rec)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				pcheck(err)
			}
		}

		records = append(records, rec)
	}

	return TweetRecordList(records)
}
