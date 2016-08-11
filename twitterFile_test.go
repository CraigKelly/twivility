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
