package main

import (
	"log"
	"os"

	"github.com/go-git/go-git/v5"
)

func main() {
	// gitRepo := "http://localhost:5050/test"
	gitRepo := "http://localhost:5050/simongottschlag/test/_git/test"
	localFolder := "/tmp/test"
	log.Printf("git clone %s", gitRepo)

	_, err := git.PlainClone(localFolder, false, &git.CloneOptions{
		URL:      gitRepo,
		Progress: os.Stdout,
	})
	if err != nil {
		log.Printf("ERROR: %s", err)
	}
}
