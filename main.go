// +build !test

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/coreos/pkg/flagutil"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
)

// TODO: we'll probably want to use extractEntitiesWithIndices from twitter-text on the client
// TODO: ADMINS - web service giving stats on file contents - extra file Options
// TODO: ADMINS - web service can be prompted to update, and also has a scheduled update

var buildDate string // Set by our build script

/////////////////////////////////////////////////////////////////////////////
// Twitter client that ACTUALLY talked to Twitter

// WrappedTwitterClient is a thin wrapper around twitter.Client
type WrappedTwitterClient struct {
	client *twitter.Client
}

// RetrieveHomeTimeline delegates to twitter.Client's Timelines.HomeTimeline
func (cli *WrappedTwitterClient) RetrieveHomeTimeline(count int, since int64, max int64) ([]twitter.Tweet, error) {
	homeTimelineParams := &twitter.HomeTimelineParams{
		Count:   count,
		MaxID:   max,
		SinceID: since,
	}
	log.Printf("GET Home Timeline => Count:%v, Max:%d, Since:%d\n",
		homeTimelineParams.Count,
		homeTimelineParams.MaxID,
		homeTimelineParams.SinceID)
	tweets, _, tweetErr := cli.client.Timelines.HomeTimeline(homeTimelineParams)
	return tweets, tweetErr
}

/////////////////////////////////////////////////////////////////////////////
// Helpers for serving

func jsonResponse(w http.ResponseWriter, req *http.Request, jsonSrc interface{}) {
	js, err := json.Marshal(jsonSrc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

/////////////////////////////////////////////////////////////////////////////
// Entry point

func main() {
	log.Printf("STARTING twivility - built %s\n", buildDate)

	flags := flag.NewFlagSet("user-auth", flag.ExitOnError)
	consumerKey := flags.String("consumer-key", "", "Twitter Consumer Key")
	consumerSecret := flags.String("consumer-secret", "", "Twitter Consumer Secret")
	accessToken := flags.String("access-token", "", "Twitter Access Token")
	accessSecret := flags.String("access-secret", "", "Twitter Access Secret")
	hostBinding := flags.String("host", "", "How to listen for service")

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
	service := NewTwivilityService(wrapped, "tweetstore.gob")

	if cmd == "service" {
		service.UpdateTwitterFile()

		http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			io.WriteString(w, "hello, world!\n") //TODO: application
		})
		http.HandleFunc("/accts", func(w http.ResponseWriter, req *http.Request) {
			accts := service.GetAccounts()
			log.Printf("GET %s - returning list of len %d\n", req.URL.Path, len(accts))
			jsonResponse(w, req, accts)
		})
		http.HandleFunc("/tweets/", func(w http.ResponseWriter, req *http.Request) {
			acct := strings.Replace(req.URL.Path, "/tweets/", "", 1)
			tweets := service.GetTweets(acct)
			log.Printf("GET %s - returning list of len %d for acct %s\n", req.URL.Path, len(tweets), acct)
			jsonResponse(w, req, service.GetTweets(acct))
		})

		// TODO: run update in the background

		addrListen := *hostBinding
		if addrListen == "" {
			log.Printf("No host specified: using default\n")
			addrListen = "0.0.0.0:8787"
		}
		log.Printf("Starting listen on %s\n", addrListen)
		http.ListenAndServe(addrListen, nil)

	} else if cmd == "update" {
		service.UpdateTwitterFile()

	} else if cmd == "dump" || cmd == "json" {
		records := service.ReadTwitterFile()
		fmt.Println(string(CreateTwitterJSON(records)))

	} else {
		log.Printf("Options are service, update, or dump\n")
	}
}
