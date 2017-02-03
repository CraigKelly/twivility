package main

import (
	"errors"
	"io/ioutil"
	"os"
	"strconv"
	"testing"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/stretchr/testify/assert"
)

/////////////////////////////////////////////////////////////////////////////
// Testing with a mock client

type TestTwitterClient struct{}

func (cli *TestTwitterClient) RetrieveHomeTimeline(count int, since int64, max int64) ([]twitter.Tweet, error) {
	//SPECIAL: return an error when count < 0
	if count < 0 {
		return nil, errors.New("RetrieveHomeTimeline test requested an error")
	}

	users := map[int]string{
		101: "User1",
		202: "User2",
		42:  "CoolUser",
	}

	tweets := make([]twitter.Tweet, 0, 8)
	tcounter := 0

	one := func(tid int64, txt string, uid int, htcount int, umcount int) {
		// Book keeping
		if len(tweets) >= count {
			return
		} else if max != 0 && tid > max {
			return
		} else if max != 0 && tid < since {
			return
		}
		tcounter++

		// Hashtags
		for ht := 1; ht <= htcount; ht++ {
			txt += " " + "#ht" + strconv.Itoa(ht)
		}

		// User mentionsHashtags
		for m := 1; m <= umcount; m++ {
			txt += " " + "@Mention" + strconv.Itoa(m)
		}

		// Create and append tweet
		tweet := twitter.Tweet{
			ID:        tid,
			CreatedAt: "testTime+" + strconv.Itoa(tcounter),
			Text:      txt,
			User: &twitter.User{
				ID:         int64(uid),
				IDStr:      strconv.Itoa(uid),
				Name:       users[uid],
				ScreenName: "@" + users[uid],
			},
		}
		tweets = append(tweets, tweet)
	}

	one(1, "First tweet", 101, 0, 0)
	one(2, "Second tweet A", 202, 2, 2)
	one(3, "Second tweet B", 202, 1, 1)
	one(4, "Last Tweet", 42, 0, 0)

	return tweets, nil
}

func TestTwitterFileIO(t *testing.T) {
	assert := assert.New(t)

	tmpfile, err := ioutil.TempFile("", "twivility")
	pcheck(err)
	defer os.Remove(tmpfile.Name())

	client := &TestTwitterClient{}
	service := NewTwivilityService(client, tmpfile.Name())

	assertUpdate := func(expected int) {
		count, err := service.UpdateTwitterFile(false)
		assert.Nil(err)
		assert.Equal(expected, count)
	}

	assertUpdate(4)
	assertUpdate(0)

	tweets := service.ReadTwitterFile()
	assert.Equal(4, len(tweets))

	mn, mx := tweets.MinMax()
	assert.Equal(int64(1), mn)
	assert.Equal(int64(4), mx)
	assert.Contains(tweets.Seen(), int64(2))
	assert.Contains(tweets.Seen(), int64(3))

	// Make sure sorted tweet ID descending and array looks correct - necessary
	// for our assumptions about what we should have read
	assert.Equal(int64(4), tweets[0].TweetID)
	assert.Equal(int64(3), tweets[1].TweetID)
	assert.Equal(int64(2), tweets[2].TweetID)
	assert.Equal(int64(1), tweets[3].TweetID)

	// check hash tags came back OK
	assert.Equal(0, len(tweets[0].Hashtags))
	assert.Equal(1, len(tweets[1].Hashtags))
	assert.Equal(2, len(tweets[2].Hashtags))
	assert.Equal(0, len(tweets[3].Hashtags))

	assertHashtag := func(idx int, hidx int, expTag string) {
		assert.Equal(expTag, tweets[idx].Hashtags[hidx])
	}
	assertHashtag(2, 0, "#ht1")
	assertHashtag(2, 1, "#ht2")
	assertHashtag(1, 0, "#ht1")

	// check user mentions came back OK
	assert.Equal(0, len(tweets[0].Mentions))
	assert.Equal(1, len(tweets[1].Mentions))
	assert.Equal(2, len(tweets[2].Mentions))
	assert.Equal(0, len(tweets[3].Mentions))

	assertMention := func(idx int, midx int, expUsr string) {
		assert.Equal(expUsr, tweets[idx].Mentions[midx])
	}
	assertMention(2, 0, "@Mention1")
	assertMention(2, 1, "@Mention2")
	assertMention(1, 0, "@Mention1")
}

func TestTwitterAcct(t *testing.T) {
	assert := assert.New(t)

	tmpfile, err := ioutil.TempFile("", "twivility")
	pcheck(err)
	defer os.Remove(tmpfile.Name())

	client := &TestTwitterClient{}
	service := NewTwivilityService(client, tmpfile.Name())
	service.UpdateTwitterFile(false)

	accts := service.GetAccounts()
	assert.Equal(3, len(accts))
	assert.Contains(accts, "@User1")
	assert.Contains(accts, "@User2")
	assert.Contains(accts, "@CoolUser")
}

func TestTwitterAcctTweets(t *testing.T) {
	assert := assert.New(t)

	tmpfile, err := ioutil.TempFile("", "twivility")
	pcheck(err)
	defer os.Remove(tmpfile.Name())

	client := &TestTwitterClient{}
	service := NewTwivilityService(client, tmpfile.Name())
	service.UpdateTwitterFile(false)

	assert.Equal(0, len(service.GetTweets("not-an-account")))

	var tweets TweetRecordList

	tweets = service.GetTweets("@CoolUser")
	assert.Equal(1, len(tweets))
	assert.Equal(int64(4), tweets[0].TweetID)
	assert.Equal("Last Tweet", tweets[0].Text)

	tweets = service.GetTweets("@User2")
	assert.Equal(2, len(tweets))
}

/////////////////////////////////////////////////////////////////////////////
// Testing when the client fails - we shouldn't get results, but we shouldn't
// kill data either

type FailingTwitterClient struct{}

func (cli *FailingTwitterClient) RetrieveHomeTimeline(count int, since int64, max int64) ([]twitter.Tweet, error) {
	return nil, errors.New("I always fail.")
}

func TestTwitterFailingClient(t *testing.T) {
	assert := assert.New(t)

	tmpfile, err := ioutil.TempFile("", "twivility")
	pcheck(err)
	defer os.Remove(tmpfile.Name())

	badClient := &FailingTwitterClient{}
	goodClient := &TestTwitterClient{}

	assertUpdate := func(expected int, haveErr bool, s *TwivilityService) {
		count, err := s.UpdateTwitterFile(false)
		if haveErr {
			assert.NotNil(err)
		} else {
			assert.Nil(err)
		}
		assert.Equal(expected, count)
	}

	var service *TwivilityService

	// Need to fail some
	service = NewTwivilityService(badClient, tmpfile.Name())
	assertUpdate(0, true, service)
	assert.Equal(0, len(service.ReadTwitterFile()))

	// Now work correctly
	service = NewTwivilityService(goodClient, tmpfile.Name())
	assertUpdate(4, false, service)
	assertUpdate(0, false, service)
	assert.Equal(4, len(service.ReadTwitterFile()))

	// Fail updating, but succeed reading
	service = NewTwivilityService(badClient, tmpfile.Name())
	assertUpdate(0, true, service)
	assert.Equal(4, len(service.ReadTwitterFile()))

	// Work correctly again
	service = NewTwivilityService(goodClient, tmpfile.Name())
	assertUpdate(0, false, service)
	assert.Equal(4, len(service.ReadTwitterFile()))
}
