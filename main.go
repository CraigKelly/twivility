// +build !test

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/coreos/pkg/flagutil"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
)

var buildDate string // Set by our build script

const (
	tweetStoreFile  = "tweetstore.gob"
	streamStoreFile = "stream.json"
)

/////////////////////////////////////////////////////////////////////////////
// Twitter client that ACTUALLY talked to Twitter

// WrappedTwitterClient is a thin wrapper around twitter.Client
type WrappedTwitterClient struct {
	client *twitter.Client
}

// RetrieveHomeTimeline delegates to twitter.Client's Timelines.HomeTimeline
func (cli *WrappedTwitterClient) RetrieveHomeTimeline(count int, since int64, max int64) ([]twitter.Tweet, error) {
	trimUser := false
	homeTimelineParams := &twitter.HomeTimelineParams{
		Count:    count,
		MaxID:    max,
		SinceID:  since,
		TrimUser: &trimUser,
	}
	log.Printf("GET Home Timeline => Count:%v, Max:%d, Since:%d\n",
		homeTimelineParams.Count,
		homeTimelineParams.MaxID,
		homeTimelineParams.SinceID)
	tweets, resp, tweetErr := cli.client.Timelines.HomeTimeline(homeTimelineParams)
	if tweetErr != nil {
		log.Printf("GET Home Timeline FAILED => Resp[%d]:%s Headers:%v\n", resp.StatusCode, resp.Status, resp.Header)
	}
	return tweets, tweetErr
}

/////////////////////////////////////////////////////////////////////////////
// Actual service running

func jsonResponse(w http.ResponseWriter, req *http.Request, jsonSrc interface{}) {
	js, err := json.Marshal(jsonSrc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// statResult is what we return for the stats API (and isn't used anywhere else)
type statResult struct {
	LastUpdateTime string
	LastStreamRecv string
	MentionCount   int64
	StoreSizeMB    float32
	StreamSizeMB   float32
	Accts          map[string]int
}

// fileSizeMB returns the size of the given file in MB. On any error (including
// file not found), 0.0 is returned
func fileSizeMB(filename string) float32 {
	st, err := os.Stat(filename)
	if err != nil || st.IsDir() {
		return 0.0
	}
	return float32(st.Size()) / 1048576.0
}

func runService(addrListen string, service *TwivilityService, mentions *TwitterMentions) {
	// Initial update
	service.UpdateTwitterFile(false)
	lastUpdate := time.Now()

	// Start the mention stream
	var lastMentionRecv time.Time

	var recentMentions struct {
		tweets []TweetRecord
		curr   int
	}
	recentMentions.curr = -1
	recentMentions.tweets = make([]TweetRecord, 100)

	mentions.Mention = func(tweet TweetRecord) {
		lastMentionRecv = time.Now()

		recentMentions.curr = (recentMentions.curr + 1) % 100
		recentMentions.tweets[recentMentions.curr] = tweet

		cnt := mentions.Count
		if cnt > 0 && cnt%1000 == 0 {
			log.Printf("Mentions: Seen %d\n", cnt)
		}
	}

	go mentions.Stream(service.GetAccounts())
	defer mentions.Stop()

	// Make sure to update the tweets every 5 minutes. We also take the
	// opportunity to stop and restart our stream gathering
	updateTicker := time.NewTicker(5 * time.Minute)
	updateQuit := make(chan struct{})
	defer close(updateQuit)
	go func() {
		for {
			select {
			case <-updateTicker.C:
				mentions.Stop()
				service.UpdateTwitterFile(false)
				lastUpdate = time.Now()
				go mentions.Stream(service.GetAccounts())
			case <-updateQuit:
				updateTicker.Stop()
				return
			}
		}
	}()

	// API endpoints

	http.HandleFunc("/api/stats", func(w http.ResponseWriter, req *http.Request) {
		stats := statResult{
			LastUpdateTime: lastUpdate.Format(time.RFC1123Z),
			LastStreamRecv: lastMentionRecv.Format(time.RFC1123Z),
			MentionCount:   mentions.Count,
			StoreSizeMB:    fileSizeMB(tweetStoreFile),
			StreamSizeMB:   fileSizeMB(streamStoreFile),
			Accts:          make(map[string]int),
		}
		for _, acct := range service.GetAccounts() {
			stats.Accts[acct] = service.GetTweets(acct).Len()
		}

		log.Printf("GET %s - returning stats\n", req.URL.Path)
		jsonResponse(w, req, stats)
	})

	http.HandleFunc("/api/accts", func(w http.ResponseWriter, req *http.Request) {
		accts := service.GetAccounts()
		log.Printf("GET %s - returning list of len %d\n", req.URL.Path, len(accts))
		jsonResponse(w, req, accts)
	})

	http.HandleFunc("/api/tweets/", func(w http.ResponseWriter, req *http.Request) {
		acct := strings.Replace(req.URL.Path, "/api/tweets/", "", 1)
		tweets := service.GetTweets(acct)
		log.Printf("GET %s - returning list of len %d for acct %s\n", req.URL.Path, len(tweets), acct)
		jsonResponse(w, req, tweets)
	})

	http.HandleFunc("/api/recent-stream", func(w http.ResponseWriter, req *http.Request) {
		tweets := TweetRecordList(make([]TweetRecord, 0, 100))
		for _, tw := range recentMentions.tweets {
			if tw.TweetID != 0 {
				tweets = append(tweets, tw)
			}
		}
		SortTwitterRecords(tweets)
		jsonResponse(w, req, tweets)
	})

	// API default and unspecified API end points
	http.HandleFunc("/api/", func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/" {
			http.Error(w, "Unknown API path "+req.URL.Path, 404)
			return
		}
		http.ServeFile(w, req, "./client/api-default.html")
	})

	// Our static HTML5 client
	fs := http.FileServer(http.Dir("./client"))
	http.Handle("/client/", http.StripPrefix("/client/", fs))

	// The default page if you just come to the root of the site (or an
	// unhandled endpoint)
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/" {
			http.Error(w, "Unknown API path "+req.URL.Path, 404)
			return
		}
		http.ServeFile(w, req, "./client/main.html")
	})

	if addrListen == "" {
		log.Printf("No host specified: using default\n")
		addrListen = "127.0.0.1:8484"
	}
	log.Printf("Starting listen on %s\n", addrListen)
	http.ListenAndServe(addrListen, nil)

	log.Printf("Exiting\n")
}

/////////////////////////////////////////////////////////////////////////////
// Entry point

func main() {
	log.Printf("STARTING twivility - built %s\n", buildDate)

	flags := flag.NewFlagSet("twivility", flag.ExitOnError)
	consumerKey := flags.String("consumer-key", "", "Twitter Consumer Key")
	consumerSecret := flags.String("consumer-secret", "", "Twitter Consumer Secret")
	accessToken := flags.String("access-token", "", "Twitter Access Token")
	accessSecret := flags.String("access-secret", "", "Twitter Access Secret")
	hostBinding := flags.String("host", "", "How to listen for service")
	hashtagFile := flags.String("hashtags", "", "Filename with list of hashtags")

	pcheck(flags.Parse(os.Args[1:]))
	pcheck(flagutil.SetFlagsFromEnv(flags, "TWITTER"))

	if *consumerKey == "" || *consumerSecret == "" || *accessToken == "" || *accessSecret == "" {
		log.Panicf("Consumer key/secret and Access token/secret required\n")
	}

	cmd := flags.Arg(0)

	// Remember that OAuth1 http.Client will automatically authorize Requests
	config := oauth1.NewConfig(*consumerKey, *consumerSecret)
	token := oauth1.NewToken(*accessToken, *accessSecret)
	httpClient := config.Client(oauth1.NoContext, token)

	// Twitter client
	client := twitter.NewClient(httpClient)

	// One-timer/startup - Verify Credentials
	log.Printf("Verifying user...\n")
	verifyParams := &twitter.AccountVerifyParams{
		SkipStatus:   twitter.Bool(true),
		IncludeEmail: twitter.Bool(true),
	}
	user, _, userError := client.Accounts.VerifyCredentials(verifyParams)
	pcheck(userError)
	log.Printf("User Verified:%v\n", user.Name)

	wrapped := &WrappedTwitterClient{client: client}
	service := NewTwivilityService(wrapped, tweetStoreFile)

	if cmd == "update" {
		service.UpdateTwitterFile(false)
	} else if cmd == "backfill" {
		service.UpdateTwitterFile(true)
		service.UpdateTwitterFile(false)
	} else if cmd == "dump" || cmd == "json" {
		records := service.ReadTwitterFile()
		for _, rec := range records {
			txt, err := json.Marshal(rec)
			pcheck(err)
			fmt.Println(string(txt))
		}
	} else if cmd == "service" {
		log.Printf("Using hashtag file %s\n", *hashtagFile)
		mentions := NewTwitterMentions(client, streamStoreFile, *hashtagFile)
		runService(*hostBinding, service, mentions)
	} else if cmd == "stream" {
		// We need an accounts list to listen to
		log.Println("Outputting streamed mentions until CTRL+C")
		service.UpdateTwitterFile(false)
		accts := service.GetAccounts()

		log.Printf("Using hashtag file %s\n", *hashtagFile)
		mentions := NewTwitterMentions(client, streamStoreFile, *hashtagFile)
		mentions.Mention = func(tweet TweetRecord) {
			log.Printf("%d: %s\n", tweet.TweetID, tweet.Text)
		}
		go mentions.Stream(accts)

		ch := make(chan os.Signal)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		log.Println(<-ch)
		mentions.Stop()
	} else {
		log.Printf("Options are service, update, backfill, dump, or stream\n")
	}
}
