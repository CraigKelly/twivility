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

// FileRecord is the struct we store in a file for a single tweet
type FileRecord struct {
	TweetID   int64
	UserID    int64
	UserName  string
	Text      string
	Timestamp string
}

// FileRecordSlice is a slice of FileRecords
type FileRecordSlice []FileRecord

// sort.Interface
func (frs FileRecordSlice) Len() int           { return len(frs) }
func (frs FileRecordSlice) Swap(i, j int)      { frs[i], frs[j] = frs[j], frs[i] }
func (frs FileRecordSlice) Less(i, j int) bool { return frs[i].TweetID < frs[j].TweetID }

// MinMax return the min and max tweet ID for the given slice. REQUIRES sorted slice
func (frs FileRecordSlice) MinMax() (mn int64, mx int64) {
	ln := len(frs)
	if ln < 1 {
		return 0, 0
	}
	return frs[ln-1].TweetID, frs[0].TweetID
}

// Seen returns a map of tweet ID's that have been seen
func (frs FileRecordSlice) Seen() map[int64]bool {
	seen := make(map[int64]bool)
	for _, tweet := range frs {
		seen[tweet.TweetID] = true
	}
	return seen
}

// SortRecords sorts the given FileRecordSlice INPLACE in our "canonical" order
func SortRecords(frs FileRecordSlice) {
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

// WriteFile writes the file - note that the slice is sorted (and therefore mutated)
func (frs FileRecordSlice) WriteFile(filename string) {
	output, err := os.Create(filename)
	pcheck(err)
	defer safeClose(output)

	SortRecords(frs)

	enc := gob.NewEncoder(output)
	for _, obj := range frs {
		err := enc.Encode(obj)
		pcheck(err)
	}
}

// ReadFile reads the specified file name for our twitter records
func ReadFile(filename string) FileRecordSlice {
	input, err := os.Open(filename)
	pcheck(err)
	defer safeClose(input)

	dec := gob.NewDecoder(input)
	records := make([]FileRecord, 0, 512)
	var rec FileRecord

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

	return FileRecordSlice(records)
}
