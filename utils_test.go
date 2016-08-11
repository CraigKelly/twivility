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
