package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"strings"

	"github.com/dghubble/go-twitter/twitter"
)

// TODO: need some testing up in here!

// TwitterMentions provides stream-to-file functionality
type TwitterMentions struct {
	Client   *twitter.Client
	Filename string
	Count    int64
	stream   *twitter.Stream
	Mention  func(tweet TweetRecord)
}

// NewTwitterMentions creates a new TwitterMentions instance
func NewTwitterMentions(client *twitter.Client, filename string) *TwitterMentions {
	return &TwitterMentions{
		Client:   client,
		Filename: filename,
		Count:    0,
		stream:   nil,
	}
}

// WriteTweet writes the given tweet to the Writer as a line of JSON
func (tm *TwitterMentions) WriteTweet(tweet *twitter.Tweet, target io.Writer) error {
	record := NewTweetRecord(tweet)
	tm.Count++

	txt, err := json.Marshal(record)
	if err != nil {
		return err
	}

	postfix := "\n"
	txt = append(txt, postfix...)

	_, err = target.Write(txt)
	if err != nil {
		return err
	}

	if tm.Mention != nil {
		tm.Mention(record)
	}

	return nil
}

// Stream starts listening for mentions of accts and writing to the file.
// Supports running in a goroutine.
func (tm *TwitterMentions) Stream(accts []string) error {
	// Restarts should work
	if err := tm.Stop(); err != nil {
		return err
	}

	// Insure all accts are prefixed with @
	for i, acct := range accts {
		if !strings.HasPrefix(acct, "@") {
			accts[i] = "@" + acct
		}
	}

	log.Printf("Mentions: starting stream on accts %v\n", accts)

	// Our file should exist, even if it's empty
	TouchFile(tm.Filename)

	// If we've never seen a count, start with the line count in the data file
	if tm.Count < 1 {
		initCount, err := lineCounter(tm.Filename)
		pcheck(err) // Yes, panic - because we can't stream at all
		tm.Count = int64(initCount)
	}

	// Open the data file
	output, err := os.OpenFile(tm.Filename, os.O_APPEND|os.O_WRONLY, 0600)
	pcheck(err)
	defer SafeClose(output)

	// Get tweets via search to backfill our stream
	// TODO: only five @s at a time - will need 5 calls for startup on list of 25
	backfill := &twitter.SearchTweetParams{
		Query: strings.Join(accts, " OR "), // note: OR is casesensitive
	}
	search, _, err := tm.Client.Search.Tweets(backfill)
	if err != nil {
		// Note that we'll continue if we see an error
		log.Printf("Mentions: there was an error getting backfills: %v\n", err)
	}
	if search != nil {
		log.Printf("Mentions: backfill on %v received %d responses\n", backfill.Query, len(search.Statuses))
		for _, t := range search.Statuses {
			err = tm.WriteTweet(&t, output)
			if err != nil {
				log.Printf("Mentions: could not write backfill tweet to %s: %v\n", tm.Filename, err)
			}
		}
	}

	// If we see a stream, someone else is running and we have a race condition
	if tm.stream != nil {
		log.Panicln("Stream found an already running instance: RACE CONDITION")
	}

	// Start our stream
	params := &twitter.StreamFilterParams{
		Track:         accts,
		StallWarnings: twitter.Bool(true),
	}
	stream, err := tm.Client.Streams.Filter(params)
	if err != nil {
		log.Printf("Could not start Mention stream: %v\n", err)
		return err
	}

	// Have a stream!
	tm.stream = stream

	// Set up a demux to receive tweets
	demux := twitter.NewSwitchDemux()

	// Our main action: write the tweet to the file as a JSON record on a line
	demux.Tweet = func(tweet *twitter.Tweet) {
		err := tm.WriteTweet(tweet, output)
		if err != nil {
			log.Printf("Mentions: could not write stream tweet to %s: %v\n", tm.Filename, err)
		}
	}

	// We currently just log these warnings and disconnects...
	demux.StreamLimit = func(limit *twitter.StreamLimit) {
		log.Printf("Mentions: stream limit - %d undelivered matches\n", limit.Track)
	}
	demux.StreamDisconnect = func(dis *twitter.StreamDisconnect) {
		log.Printf("Mentions: Disconnect [%d] %s\n", dis.Code, dis.Reason)
	}
	demux.Warning = func(warn *twitter.StallWarning) {
		log.Printf("Mentions: Stall Warning (%d%%) [%s] %s\n", warn.PercentFull, warn.Code, warn.Message)
	}

	// Loop until the stream is no more
	for msg := range stream.Messages {
		demux.Handle(msg)
	}

	return nil
}

// Stop stop listening for mentions of accts
func (tm *TwitterMentions) Stop() error {
	if tm.stream == nil {
		return nil // Nothing to do
	}
	tm.stream.Stop()
	tm.stream = nil
	log.Printf("Mentions: stopped stream\n")
	return nil
}
