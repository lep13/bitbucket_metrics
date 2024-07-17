package main

import (
	"github.com/lep13/bitbucket_metrics/config"
	"github.com/lep13/bitbucket_metrics/internal/database"
	"github.com/lep13/bitbucket_metrics/internal/bitbucket"
	"log"
)

func main() {
	// Initialize configuration
	config, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Initialize MongoDB
	err = database.InitializeMongoDB(config.MongoDBURI)
	if err != nil {
		log.Fatalf("Error initializing MongoDB: %v", err)
	}

	// Fetch commit data from Bitbucket and save to MongoDB
	err = bitbucket.FetchAndSaveCommits(config.BitbucketUsername, config.BitbucketAppPassword)
	if err != nil {
		log.Fatalf("Error fetching and saving commits: %v", err)
	}

	log.Println("Successfully fetched and saved commit data")
}
