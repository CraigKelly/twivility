package main

import "fmt"

func main() {
	defer func() {
		fmt.Println("Twivility.com server shutting down")
	}()

	fmt.Println("Twivility.com server starting up")
}
