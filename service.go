package main

import (
	"log"
	"strings"
	"sync"

	"github.com/dghubble/go-twitter/twitter"
)

// TwitterClient is the generic interface we need for twitter (and is what we
// stub out for unit testing)
type TwitterClient interface {
	RetrieveHomeTimeline(count int, since int64, max int64) ([]twitter.Tweet, error)
}

// TwivilityService handles rest-ful requests for twivility. Someone else needs
// to map our functions
type TwivilityService struct {
	client        TwitterClient
	dataFileName  string
	currentTweets TweetFileRecordSlice
	tweetMap      map[string]TweetFileRecordSlice
	tweetStoreMtx sync.RWMutex
}

// NewTwivilityService - return a nice, new twitter service. See main.go for
// how a full client for Twitter is created
func NewTwivilityService(client TwitterClient, dataFileName string) *TwivilityService {
	return &TwivilityService{client: client, dataFileName: dataFileName}
}

// updateTweetMap recreates service.tweetMap
// IMPORTANT! This function assumes that it does NOT need to worry about
// tweetStoreMtx. Only call while service.tweetStoreMtx.Lock() is active
func (service *TwivilityService) updateTweetMap() {
	service.tweetMap = make(map[string]TweetFileRecordSlice)
	for _, tweet := range service.currentTweets {
		list, inMap := service.tweetMap[tweet.UserScreenName]
		if !inMap {
			list = make(TweetFileRecordSlice, 0, len(service.currentTweets)/4)
		}
		service.tweetMap[tweet.UserScreenName] = append(list, tweet)
	}
}

func firstNonBlank(candidates ...string) string {
	for _, s := range candidates {
		t := strings.Trim(s, " ")
		if len(t) > 0 {
			return t
		}
	}
	return ""
}

func extractTweet(tweet twitter.Tweet) TweetFileRecord {
	hashtags := make([]string, 0, len(tweet.Entities.Hashtags))
	for _, ht := range tweet.Entities.Hashtags {
		hashtags = append(hashtags, ht.Text)
	}

	mentions := make([]string, 0, len(tweet.Entities.UserMentions))
	for _, m := range tweet.Entities.UserMentions {
		txt := firstNonBlank(m.ScreenName, m.Name, m.IDStr)
		if len(txt) > 0 {
			mentions = append(mentions, txt)
		}
	}

	return TweetFileRecord{
		TweetID:        tweet.ID,
		UserID:         tweet.User.ID,
		UserName:       tweet.User.Name,
		UserScreenName: tweet.User.ScreenName,
		Text:           tweet.Text,
		Timestamp:      tweet.CreatedAt,
		FavoriteCount:  tweet.FavoriteCount,
		RetweetCount:   tweet.RetweetCount,
		Hashtags:       hashtags,
		Mentions:       mentions,
	}
}

// ReadTwitterFile returns all records in our current twitter data store
func (service *TwivilityService) ReadTwitterFile() TweetFileRecordSlice {
	// Yes: writer lock since we touch the file and update currentTweets
	service.tweetStoreMtx.Lock()
	defer service.tweetStoreMtx.Unlock()

	TouchFile(service.dataFileName) // Make sure at least empty file exists
	service.currentTweets = ReadTwitterFile(service.dataFileName)
	service.updateTweetMap()

	log.Printf("Read %d records from %s\n", len(service.currentTweets), service.dataFileName)
	return service.currentTweets
}

// UpdateTwitterFile updates our twitter store on disk
func (service *TwivilityService) UpdateTwitterFile() (int, error) {
	service.tweetStoreMtx.Lock()
	defer service.tweetStoreMtx.Unlock()

	TouchFile(service.dataFileName) // Make sure at least empty file exists
	existing := ReadTwitterFile(service.dataFileName)
	SortTwitterRecords(existing)
	seen := existing.Seen()
	mnID, mxID := existing.MinMax()
	log.Printf("Found %d tweets in file %s - ID range %d<->%d\n", len(existing), service.dataFileName, mnID, mxID)

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
			return 0, tweetErr
		}

		addCount := 0
		preSeen := len(seen)
		batchMin := int64(0)
		for _, tweet := range tweets {
			tweetID := tweet.ID
			if _, inMap := seen[tweetID]; !inMap {
				// New ID!
				newRec := extractTweet(tweet)
				existing = append(existing, newRec)
				seen[tweetID] = true
				addCount++
				if tweetID < batchMin || batchMin == 0 {
					batchMin = tweetID
				}
			}
		}

		totalAdded += addCount
		if totalAdded > 2000 {
			break // Totaly arbitrary - don't get more than 2K at a time
		}

		if len(seen) == preSeen {
			break // Nothing new or don't know how to continue
		}

		qMax = batchMin
		if qMax <= qSince+1 {
			break // Nothing left to find
		}
	}

	log.Printf("Added %d records: rewriting file %s\n", totalAdded, service.dataFileName)
	existing.WriteTwitterFile(service.dataFileName)
	service.currentTweets = existing
	service.updateTweetMap()
	return totalAdded, nil
}

// GetAccounts returns all accounts in our current twitter store
func (service *TwivilityService) GetAccounts() []string {
	service.tweetStoreMtx.RLock()
	defer service.tweetStoreMtx.RUnlock()

	accts := make([]string, 0, 4)
	for acct := range service.tweetMap {
		accts = append(accts, acct)
	}

	return accts
}

// GetTweets returns the sorted tweet list for the specified account
func (service *TwivilityService) GetTweets(acct string) TweetFileRecordSlice {
	service.tweetStoreMtx.RLock()
	defer service.tweetStoreMtx.RUnlock()

	list, inMap := service.tweetMap[acct]
	if !inMap {
		log.Printf("No map entry found for acct %v", acct)
		return make(TweetFileRecordSlice, 0, 0)
	}

	return list
}
