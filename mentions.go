package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/dghubble/go-twitter/twitter"
)

// TODO: need some testing up in here!
// TODO: test with George's list
// TODO: deploy

// TwitterMentions provides stream-to-file functionality
type TwitterMentions struct {
	Client   *twitter.Client
	Filename string
	Count    int64
	stream   *twitter.Stream
	Hashtags []string
	Mention  func(tweet TweetRecord)
}

// readHashtags reads whitespace-delimited hashtags from the given file and
// returns an array of strings. Any string that does not begin with either #
// or @ will have # prepended. The array returned is sorted and duplicate-free
func readHashtags(filename string) ([]string, error) {
	if filename == "" {
		return []string{}, nil
	}

	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return []string{}, err
	}

	tags := NewUniqueStrings()
	for _, one := range strings.Fields(string(buf)) {
		if !strings.HasPrefix(one, "#") && !strings.HasPrefix(one, "@") {
			one = "#" + one
		}
		tags.Add(one)
	}

	return tags.Strings(), nil
}

// NewTwitterMentions creates a new TwitterMentions instance
func NewTwitterMentions(client *twitter.Client, filename string, hashtagFile string) *TwitterMentions {
	tags, err := readHashtags(hashtagFile)
	pcheck(err)

	return &TwitterMentions{
		Client:   client,
		Filename: filename,
		Count:    0,
		Hashtags: tags,
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

	// Create our tracking array (and insure all accts are prefixed with @)
	gather := NewUniqueStrings()
	for _, acct := range accts {
		if !strings.HasPrefix(acct, "@") {
			acct = "@" + acct
		}
		gather.Add(acct)
	}
	for _, tag := range tm.Hashtags {
		gather.Add(tag)
	}
	trackQuery := gather.Strings()

	log.Printf("Mentions: starting stream on %v\n", trackQuery)

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

	// If we see a stream, someone else is running and we have a race condition
	if tm.stream != nil {
		log.Panicln("Stream found an already running instance: RACE CONDITION")
	}

	// Start our stream
	// TODO: Track should get list of accts AND list of hastags
	params := &twitter.StreamFilterParams{
		Track:         trackQuery,
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
