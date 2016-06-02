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

func logPanic(msg string) {
	log.Fatal(msg)
	panic(msg)
}

func pcheck(err error) {
	if err != nil {
		logPanic(fmt.Sprintf("Fatal Error: %v\n", err))
	}
}

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

	// Remember that OAuth1 http.Client will automatically authorize Requests
	config := oauth1.NewConfig(*consumerKey, *consumerSecret)
	token := oauth1.NewToken(*accessToken, *accessSecret)
	httpClient := config.Client(oauth1.NoContext, token)

	// Twitter client
	client := twitter.NewClient(httpClient)

	// One-timer/startup - Verify Credentials
	verifyParams := &twitter.AccountVerifyParams{
		SkipStatus:   twitter.Bool(true),
		IncludeEmail: twitter.Bool(true),
	}
	user, _, userError := client.Accounts.VerifyCredentials(verifyParams)
	pcheck(userError)
	fmt.Printf("User Verified:%v\n", user.Name)

	fmt.Printf("\nTIMELINE SAMPLE\n\n")

	// Home Timeline
	homeTimelineParams := &twitter.HomeTimelineParams{Count: 20}
	tweets, _, tweetErr := client.Timelines.HomeTimeline(homeTimelineParams)
	if tweetErr != nil {
		fmt.Printf("Error getting user timeline: %v\n", tweetErr)
	} else {
		for _, tweet := range tweets {
			fmt.Printf("%s: %s\n\n", tweet.User.Name, tweet.Text)
		}
	}
}
