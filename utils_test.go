package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSieve - simple unit testing for our prime factoring function
func TestPcheck(t *testing.T) {
	assert := assert.New(t)

	assert.NotPanics(func() {
		pcheck(nil)
	})
	assert.Panics(func() {
		pcheck(errors.New("I should panic"))
	})
}

func TestUniqueStrings(t *testing.T) {
	assert := assert.New(t)

	us := NewUniqueStrings()
	assert.Len(us.Seen, 0)

	us.Add("b")
	us.Add("a")
	assert.Len(us.Seen, 2)

	us.Add("b")
	us.Add("a")
	assert.Len(us.Seen, 2)

	// Test twice
	assert.Equal([]string{"a", "b"}, us.Strings())
	assert.Equal([]string{"a", "b"}, us.Strings())
}
