package main

import (
	"bytes"
	"errors"
	"io"
	"log"
	"os"
)

// Helpers for error handling

// pcheck logs a detailed error and then panics with the same msg
func pcheck(err error) {
	if err != nil {
		log.Panicf("Fatal Error: %v\n", err)
	}
}

// SafeClose simple helper for closing closers
func SafeClose(target io.Closer) {
	err := target.Close()
	if err != nil {
		log.Printf("Error closing something - will continue\n")
	}
}

// TouchFile insures that our input/output file exists
func TouchFile(filename string) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		f, err := os.Create(filename)
		pcheck(err)
		SafeClose(f)
	}
}

// Count lines in the specified file
// Adapted from http://stackoverflow.com/questions/24562942/golang-how-do-i-determine-the-number-of-lines-in-a-file-efficiently
func lineCounter(filename string) (int, error) {
	// Make sure we have a file that exists with more than one byte
	stat, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	if stat.IsDir() {
		return 0, errors.New("No lines in a directory, genius")
	}
	if stat.Size() <= 1 {
		return 1, nil
	}

	// Actually open the file
	fd, err := os.Open(filename)
	defer SafeClose(fd)
	if err != nil {
		return 0, err
	}

	// Count line terminators in 32Kb chunks
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}
	for {
		c, err := fd.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}
