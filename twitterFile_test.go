package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSieve - simple unit testing for our prime factoring function
func TestEmptyTwitterFile(t *testing.T) {
	assert := assert.New(t)

	tmpfile, err := ioutil.TempFile("", "twivility")
	pcheck(err)
	defer os.Remove(tmpfile.Name())

	// Touch file should create a file (duh)
	TouchFile(tmpfile.Name())
	var st os.FileInfo
	st, err = tmpfile.Stat()
	assert.Nil(err)
	assert.False(st.IsDir())

	// We should be able to read an empty file and use our "usual" ops
	data := ReadTwitterFile(tmpfile.Name())
	assert.Empty(data)
	mn, mx := data.MinMax()
	assert.Equal(int64(0), mn)
	assert.Equal(int64(0), mx)
	assert.Empty(data.Seen())

	// Writing should also work (we do a shortcut check)
	data.WriteTwitterFile(tmpfile.Name())
	assert.Empty(ReadTwitterFile(tmpfile.Name()))
}

func TestSortingTwitterFileRecords(t *testing.T) {
	assert := assert.New(t)

	tmpfile, err := ioutil.TempFile("", "twivility")
	pcheck(err)
	defer os.Remove(tmpfile.Name())

	makeOne := func(tid int64, uid int64, s string) TweetFileRecord {
		return TweetFileRecord{
			TweetID:   tid,
			UserID:    uid,
			UserName:  s,
			Text:      s,
			Timestamp: s,
		}
	}

	input := TweetFileRecordSlice{
		makeOne(1, 3, "C"),
		makeOne(2, 2, "B"),
		makeOne(3, 1, "A"),
	}
	input.WriteTwitterFile(tmpfile.Name())

	output := ReadTwitterFile(tmpfile.Name())
	assert.Equal(len(input), len(output))
	seen := output.Seen()
	assert.Contains(seen, int64(1))
	assert.Contains(seen, int64(2))
	assert.Contains(seen, int64(3))
	assert.Equal(3, len(seen))

	// MinMax only works if the sort worked - which leaves 2
	mn, mx := output.MinMax()
	assert.Equal(int64(1), mn)
	assert.Equal(int64(2), output[1].TweetID)
	assert.Equal(int64(3), mx)
}
