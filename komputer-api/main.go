package main

import (
	"log"
	"os"
)

func main() {
	log.Println("komputer-api starting...")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("listening on :%s", port)
}
