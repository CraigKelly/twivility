package main

import (
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"sort"
)

// Simple helper for closing closers
func safeClose(target io.Closer) {
	err := target.Close()
	if err != nil {
		fmt.Println("Error closing something - will continue")
	}
}

// TweetFileRecord is the struct we store in a file for a single tweet
type TweetFileRecord struct {
	TweetID   int64
	UserID    int64
	UserName  string
	Text      string
	Timestamp string
}

// TweetFileRecordSlice is a slice of TweetFileRecords
type TweetFileRecordSlice []TweetFileRecord

// sort.Interface
func (frs TweetFileRecordSlice) Len() int           { return len(frs) }
func (frs TweetFileRecordSlice) Swap(i, j int)      { frs[i], frs[j] = frs[j], frs[i] }
func (frs TweetFileRecordSlice) Less(i, j int) bool { return frs[i].TweetID < frs[j].TweetID }

// MinMax return the min and max tweet ID for the given slice. REQUIRES sorted slice
func (frs TweetFileRecordSlice) MinMax() (mn int64, mx int64) {
	ln := len(frs)
	if ln < 1 {
		return 0, 0
	}
	return frs[ln-1].TweetID, frs[0].TweetID
}

// Seen returns a map of tweet ID's that have been seen
func (frs TweetFileRecordSlice) Seen() map[int64]bool {
	seen := make(map[int64]bool)
	for _, tweet := range frs {
		seen[tweet.TweetID] = true
	}
	return seen
}

// SortTwitterRecords sorts the given TweetFileRecordSlice INPLACE in our "canonical" order
func SortTwitterRecords(frs TweetFileRecordSlice) {
	sort.Sort(sort.Reverse(frs))
}

// TouchFile insures that our input/output file exists
func TouchFile(filename string) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		f, err := os.Create(filename)
		pcheck(err)
		safeClose(f)
	}
}

// WriteTwitterFile writes the file - note that the slice is sorted (and therefore mutated)
func (frs TweetFileRecordSlice) WriteTwitterFile(filename string) {
	output, err := os.Create(filename)
	pcheck(err)
	defer safeClose(output)

	SortTwitterRecords(frs)

	enc := gob.NewEncoder(output)
	for _, obj := range frs {
		err := enc.Encode(obj)
		pcheck(err)
	}
}

// ReadTwitterFile reads the specified file name for our twitter records
func ReadTwitterFile(filename string) TweetFileRecordSlice {
	input, err := os.Open(filename)
	pcheck(err)
	defer safeClose(input)

	dec := gob.NewDecoder(input)
	records := make([]TweetFileRecord, 0, 512)
	var rec TweetFileRecord

	for {
		err := dec.Decode(&rec)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				pcheck(err)
			}
		}

		records = append(records, rec)
	}

	return TweetFileRecordSlice(records)
}
