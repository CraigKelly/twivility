package main

import (
	"fmt"
	"log"
)

// Helpers for error handling

// pcheck logs a detailed error and then panics with the same msg
func pcheck(err error) {
	if err != nil {
		msg := fmt.Sprintf("Fatal Error: %v\n", err)
		log.Fatal(msg)
		panic(msg)
	}
}
