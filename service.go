package main

import (
	"log"

	"github.com/dghubble/go-twitter/twitter"
)

var dataFileName = "tweetstore.gob"

//TODO: unit test
//TODO: strategy for handling concurrency while contacting twitter

// TwitterClient is the generic interface we need for twitter (and is what we
// stub out for unit testing)
type TwitterClient interface {
	RetrieveHomeTimeline(count int, since int64, max int64) ([]twitter.Tweet, error)
}

// TwivilityService handles rest-ful requests for twivility. Someone else needs
// to map our functions
type TwivilityService struct {
	client TwitterClient
}

// NewTwivilityService - return a nice, new twitter service. See main.go for
// how a full client for Twitter is created
func NewTwivilityService(client TwitterClient) *TwivilityService {
	return &TwivilityService{client: client}
}

// ReadTwitterFile returns all records in our current twitter data store
func (service *TwivilityService) ReadTwitterFile() TweetFileRecordSlice {
	TouchFile(dataFileName) // Make sure at least empty file exists
	records := ReadTwitterFile(dataFileName)
	log.Printf("Read %d records from %s\n", len(records), dataFileName)
	return records
}

// UpdateTwitterFile updates our twitter store on disk
func (service *TwivilityService) UpdateTwitterFile() {
	TouchFile(dataFileName) // Make sure at least empty file exists
	existing := ReadTwitterFile(dataFileName)
	SortTwitterRecords(existing)
	seen := existing.Seen()
	mnID, mxID := existing.MinMax()
	log.Printf("Found %d tweets in file %s - ID range %d<->%d\n", len(existing), dataFileName, mnID, mxID)

	qCount := 190 // try to be good citizens
	qSince := int64(0)
	qMax := int64(0)
	if len(existing) > 0 {
		qSince = mxID
	}

	totalAdded := 0
	for {
		tweets, tweetErr := service.client.RetrieveHomeTimeline(qCount, qSince, qMax)
		if tweetErr != nil {
			log.Printf("Error getting user timeline: %v\n", tweetErr)
			return
		}

		addCount := 0
		preSeen := len(seen)
		batchMin := int64(0)
		for _, tweet := range tweets {
			tweetID := tweet.ID
			if _, inMap := seen[tweet.ID]; !inMap {
				// New ID!
				newRec := TweetFileRecord{
					TweetID:   tweetID,
					UserID:    tweet.User.ID,
					UserName:  tweet.User.Name,
					Text:      tweet.Text,
					Timestamp: tweet.CreatedAt,
				}
				existing = append(existing, newRec)
				seen[tweetID] = true
				addCount++

				if tweetID < batchMin || batchMin == 0 {
					batchMin = tweetID
				}

				log.Printf("Added tweet: %v\n", newRec.TweetID)
			}
		}

		totalAdded += addCount
		if len(seen) == preSeen {
			break // Nothing new or don't know how to continue
		}

		qMax = batchMin
		if qMax <= qSince+1 || qSince < 1 {
			break // Nothing left to find or only one query allowed
		}
	}

	log.Printf("Added %d records: rewriting file %s\n", totalAdded, dataFileName)
	existing.WriteTwitterFile(dataFileName)
}

// GetAccounts returns all accounts in our current twitter store
func (service *TwivilityService) GetAccounts() []string {
	accts := make([]string, 0, 4)
	//TODO: populate
	return accts
}

// GetTweets returns the sorted tweet list for the specified account
func (service *TwivilityService) GetTweets(acct string) TweetFileRecordSlice {
	results := make(TweetFileRecordSlice, 0, 512) //TODO: better guess on size
	//TODO: populate
	return results
}
