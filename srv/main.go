package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"

	"github.com/coreos/pkg/flagutil"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
)

/////////////////////////////////////////////////////////////////////////////
// Helpers for error handling

func logPanic(msg string) {
	log.Fatal(msg)
	panic(msg)
}

func pcheck(err error) {
	if err != nil {
		logPanic(fmt.Sprintf("Fatal Error: %v\n", err))
	}
}

func safeClose(target io.Closer) {
	err := target.Close()
	if err != nil {
		fmt.Println("Error closing something - will continue")
	}
}

/////////////////////////////////////////////////////////////////////////////
// File storage handling

type fileRecord struct {
	TweetID   int64
	UserID    int64
	UserName  string
	Text      string
	Timestamp string
}

type fileRecordSlice []fileRecord

// sort.Interface
func (frs fileRecordSlice) Len() int           { return len(frs) }
func (frs fileRecordSlice) Swap(i, j int)      { frs[i], frs[j] = frs[j], frs[i] }
func (frs fileRecordSlice) Less(i, j int) bool { return frs[i].TweetID < frs[j].TweetID }

// Only works if we are already sorted
func (frs fileRecordSlice) MinMax() (mn int64, mx int64) {
	ln := len(frs)
	if ln < 1 {
		return 0, 0
	}
	return frs[ln-1].TweetID, frs[0].TweetID
}

func (frs fileRecordSlice) Seen() map[int64]bool {
	seen := make(map[int64]bool)
	for _, tweet := range frs {
		seen[tweet.TweetID] = true
	}
	return seen
}

func sortRecords(frs fileRecordSlice) {
	sort.Sort(sort.Reverse(frs))
}

func touchFile(filename string) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		f, err := os.Create(filename)
		pcheck(err)
		safeClose(f)
	}
}

// Write the file - note that the slice is sorted (and therefore mutated)
func (frs fileRecordSlice) writeFile(filename string) {
	output, err := os.Create(filename)
	pcheck(err)
	defer safeClose(output)

	sortRecords(frs)

	enc := gob.NewEncoder(output)
	for _, obj := range frs {
		err := enc.Encode(obj)
		pcheck(err)
	}
}

// Read the file
func readFile(filename string) fileRecordSlice {
	input, err := os.Open(filename)
	pcheck(err)
	defer safeClose(input)

	dec := gob.NewDecoder(input)
	records := make([]fileRecord, 0, 512)
	var rec fileRecord

	for {
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

	return fileRecordSlice(records)
}

/////////////////////////////////////////////////////////////////////////////
// Implementation of actions

// UpdateTweets - read our tweet store add tweet that we haven't seen
func UpdateTweets(client *twitter.Client) {
	filename := "tweetstore.gob"

	touchFile(filename) // Make sure at least empty file exists
	existing := readFile(filename)
	sortRecords(existing)
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
				newRec := fileRecord{
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
	existing.writeFile(filename)
}

// DumpTweets - write all tweets in our tweet store
func DumpTweets() {
	filename := "tweetstore.gob"
	touchFile(filename) // Make sure at least empty file exists
	records := readFile(filename)
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
	} else if cmd == "dump" {
		DumpTweets()
	} else {
		fmt.Println("Options are update or dump")
	}
}
