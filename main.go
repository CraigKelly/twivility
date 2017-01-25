// +build !test

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
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

// TODO: let's just make this a REST-only service for now: can always write a "real" frontend later
// TODO: need to actually keep parsed entities

var buildDate string // Set by our build script

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
	tweets, _, tweetErr := cli.client.Timelines.HomeTimeline(homeTimelineParams)
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

func runService(addrListen string, service *TwivilityService) {
	// Initial update
	service.UpdateTwitterFile()

	// Make sure to update the tweets every 5 minutes
	updateTicker := time.NewTicker(5 * time.Minute)
	updateQuit := make(chan struct{})
	defer close(updateQuit)
	go func() {
		for {
			select {
			case <-updateTicker.C:
				service.UpdateTwitterFile()
			case <-updateQuit:
				updateTicker.Stop()
				return
			}
		}
	}()

	// Service functions
	http.HandleFunc("/accts", func(w http.ResponseWriter, req *http.Request) {
		accts := service.GetAccounts()
		log.Printf("GET %s - returning list of len %d\n", req.URL.Path, len(accts))
		jsonResponse(w, req, accts)
	})
	http.HandleFunc("/tweets/", func(w http.ResponseWriter, req *http.Request) {
		acct := strings.Replace(req.URL.Path, "/tweets/", "", 1)
		tweets := service.GetTweets(acct)
		log.Printf("GET %s - returning list of len %d for acct %s\n", req.URL.Path, len(tweets), acct)
		jsonResponse(w, req, tweets)
	})

	// Serve static files
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// provide some functions to our templates
	funcMap := template.FuncMap{
		"Year": func() string { return time.Now().Format("2006") },
	}

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		//TODO: move out after debugging
		templates := template.Must(template.New("ui").Funcs(funcMap).ParseFiles("main.html"))

		err := templates.ExecuteTemplate(w, "main.html", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
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

	if cmd == "update" {
		service.UpdateTwitterFile()
	} else if cmd == "dump" || cmd == "json" {
		records := service.ReadTwitterFile()
		fmt.Println(string(CreateTwitterJSON(records)))
	} else if cmd == "service" {
		runService(*hostBinding, service)
	} else if cmd == "stream" {
		// TODO: write to file in JSON (so don't need a dump option)
		// TODO: use the twitter demux from the lib
		// TODO: restart every n minutes no matter what (and also handle errors gracefully)
		// TODO: insure can run simultaneously with REST stuff in service

		// TODO: only five @s at a time - will need 5 calls for startup on list of 25
		// TODO: backfill should read previous output and use max count
		backfill := &twitter.SearchTweetParams{
			Query: "@memphispython OR @binarydolphin", // note: OR is casesensitive
			// TODO: specify at lease some of Count, SinceID, MaxID
		}
		search, _, err := client.Search.Tweets(backfill)
		pcheck(err)
		for _, t := range search.Statuses {
			fmt.Println(t)
		}
		fmt.Println("END OF BACKFILL-----------------------------------------------------")

		params := &twitter.StreamFilterParams{
			Track:         []string{"@memphispython", "@binarydolphin"},
			StallWarnings: twitter.Bool(true),
		}
		stream, err := client.Streams.Filter(params)
		pcheck(err)
		go func() {
			for sm := range stream.Messages {
				fmt.Println(sm)
			}
		}()
		ch := make(chan os.Signal)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		log.Println(<-ch)
		stream.Stop()
	} else {
		log.Printf("Options are service, update, or dump\n")
	}
}
