package main

import (
	"bitbucket_metrics/config"
	"bitbucket_metrics/internal/db"
	"bitbucket_metrics/internal/gitmetrics"
	"log"
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
	err = gitmetrics.FetchAndSaveCommits(config.BitbucketUsername, config.BitbucketAppPassword)
	if err != nil {
		log.Fatalf("Error fetching and saving commits: %v", err)
	}

	log.Println("Successfully fetched and saved commit data")
}
