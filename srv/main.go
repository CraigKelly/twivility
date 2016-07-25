package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/coreos/pkg/flagutil"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
)

// TODO: switch to some kind of dep mgr
// TODO: web service giving stats on file contents - extra file Options
// TODO: web service can be prompted to update, and also has a scheduled update
// TODO: concurrency: write to new file, take lock, replace old file, unlock
// TODO: honor lock in tweet update/dump functions
// TODO: switch from fmt.print to actual logging
// TODO: enough logging/backup to be able to restart/recover

/////////////////////////////////////////////////////////////////////////////
// Implementation of actions

// UpdateTweets - read our tweet store add tweet that we haven't seen
func UpdateTweets(client *twitter.Client) {
	filename := "tweetstore.gob"

	TouchFile(filename) // Make sure at least empty file exists
	existing := ReadFile(filename)
	SortRecords(existing)
	seen := existing.Seen()
	mnID, mxID := existing.MinMax()
	fmt.Printf("Found %d tweets in file %s - ID range %d<->%d\n", len(existing), filename, mnID, mxID)

	qCount := 190 // try to be good citizens
	qSince := int64(0)
	qMax := int64(0)
	if len(existing) > 0 {
		qSince = mxID
	}

	totalAdded := 0
	for {
		homeTimelineParams := &twitter.HomeTimelineParams{
			Count:   qCount,
			MaxID:   qMax,
			SinceID: qSince,
		}
		fmt.Printf("GET! Count:%v, Max:%d, Since:%d\n",
			homeTimelineParams.Count,
			homeTimelineParams.MaxID,
			homeTimelineParams.SinceID)
		tweets, _, tweetErr := client.Timelines.HomeTimeline(homeTimelineParams)
		if tweetErr != nil {
			fmt.Printf("Error getting user timeline: %v\n", tweetErr)
			return
		}

		addCount := 0
		preSeen := len(seen)
		batchMin := int64(0)
		for _, tweet := range tweets {
			tweetID := tweet.ID
			if _, inMap := seen[tweet.ID]; !inMap {
				// New ID!
				newRec := FileRecord{
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

				fmt.Printf("Added tweet: %v\n", newRec.TweetID)
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

	fmt.Printf("Added %d records: rewriting file %s\n", totalAdded, filename)
	existing.WriteFile(filename)
}

// DumpTweets - write all tweets in our tweet store
func DumpTweets() {
	filename := "tweetstore.gob"
	TouchFile(filename) // Make sure at least empty file exists
	records := ReadFile(filename)
	fmt.Printf("Read %d records from %s\n", len(records), filename)
	for i, tweet := range records {
		fmt.Printf("Rec #%12d: %v\n", i, tweet)
	}
}

/////////////////////////////////////////////////////////////////////////////
// Entry point

func main() {
	flags := flag.NewFlagSet("user-auth", flag.ExitOnError)
	consumerKey := flags.String("consumer-key", "", "Twitter Consumer Key")
	consumerSecret := flags.String("consumer-secret", "", "Twitter Consumer Secret")
	accessToken := flags.String("access-token", "", "Twitter Access Token")
	accessSecret := flags.String("access-secret", "", "Twitter Access Secret")

	pcheck(flags.Parse(os.Args[1:]))
	pcheck(flagutil.SetFlagsFromEnv(flags, "TWITTER"))

	if *consumerKey == "" || *consumerSecret == "" || *accessToken == "" || *accessSecret == "" {
		log.Fatal("Consumer key/secret and Access token/secret required")
	}

	cmd := flags.Arg(0)

	// Remember that OAuth1 http.Client will automatically authorize Requests
	config := oauth1.NewConfig(*consumerKey, *consumerSecret)
	token := oauth1.NewToken(*accessToken, *accessSecret)
	httpClient := config.Client(oauth1.NoContext, token)

	// Twitter client
	client := twitter.NewClient(httpClient)

	// One-timer/startup - Verify Credentials
	fmt.Println("Verifying user...")
	verifyParams := &twitter.AccountVerifyParams{
		SkipStatus:   twitter.Bool(true),
		IncludeEmail: twitter.Bool(true),
	}
	user, _, userError := client.Accounts.VerifyCredentials(verifyParams)
	pcheck(userError)
	fmt.Printf("User Verified:%v\n", user.Name)

	if cmd == "update" {
		UpdateTweets(client)
	} else if cmd == "dump" || cmd == "json" {
		DumpTweets()
	} else {
		fmt.Println("Options are update or dump")
	}
}
