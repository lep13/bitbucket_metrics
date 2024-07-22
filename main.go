package main

import (
	"log"

	"github.com/lep13/bitbucket_metrics/config"
	"github.com/lep13/bitbucket_metrics/internal/bitbucket"
	db "github.com/lep13/bitbucket_metrics/internal/database"
)

func main() {
	// Initialize configuration
	config, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Initialize MongoDB
	err = db.InitializeMongoDB(config.MongoDBURI)
	if err != nil {
		log.Fatalf("Error initializing MongoDB: %v", err)
	}

	// Fetch commit data from Bitbucket and save to MongoDB
	err = bitbucket.FetchAndSaveCommits(config.BitbucketAccessToken)
	if err != nil {
		log.Fatalf("Error fetching and saving commits: %v", err)
	}

	log.Println("Successfully fetched and saved commit data")
}
