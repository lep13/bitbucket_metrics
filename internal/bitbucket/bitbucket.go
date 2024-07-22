package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/lep13/bitbucket_metrics/internal/bitbucket/models"
	db "github.com/lep13/bitbucket_metrics/internal/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func FetchAndSaveCommits(accessToken string) error {
	repos, err := fetchRepositories(accessToken)
	if err != nil {
		return fmt.Errorf("failed to fetch repositories: %v", err)
	}

	for _, repo := range repos {
		log.Printf("Processing repository: %s", repo.Name)
		commits, err := fetchCommits(accessToken, repo.Slug)
		if err != nil {
			log.Printf("Failed to fetch commits for repository %s: %v", repo.Slug, err)
			continue
		}

		for _, commit := range commits {
			log.Printf("Processing commit: %s", commit.Hash)
			reviewedBy := ""
			if commit.ReviewedBy.User.DisplayName != "" {
				reviewedBy = commit.ReviewedBy.User.DisplayName
			}
			pullRequestID := ""
			if commit.PullRequest.ID != "" {
				pullRequestID = commit.PullRequest.ID
			}
			newCommit := models.Commit{
				ProjectName:   repo.Project.Name,
				RepoName:      repo.Name,
				CommitMessage: commit.Message,
				CommitID:      commit.Hash,
				CommittedBy:   commit.Author.User.DisplayName,
				LinesAdded:    commit.Summary.LinesAdded,
				LinesDeleted:  commit.Summary.LinesDeleted,
				CommitDate:    commit.Date,
				ReviewedBy:    reviewedBy,
				PullRequestID: pullRequestID,
			}

			log.Printf("Upserting commit: %+v", newCommit)

			collection := db.GetCollection()
			log.Printf("Using collection: %v", collection)
			updateResult, err := collection.UpdateOne(
				context.Background(),
				bson.M{"commit_id": newCommit.CommitID},
				bson.M{"$set": newCommit},
				options.Update().SetUpsert(true),
			)
			if err != nil {
				log.Printf("Failed to upsert commit %s: %v", newCommit.CommitID, err)
			} else {
				log.Printf("Successfully upserted commit: %s, MatchedCount: %d, ModifiedCount: %d, UpsertedCount: %d, UpsertedID: %v",
					newCommit.CommitID, updateResult.MatchedCount, updateResult.ModifiedCount, updateResult.UpsertedCount, updateResult.UpsertedID)
			}
		}
	}

	return nil
}

func fetchRepositories(accessToken string) ([]models.Repository, error) {
	url := "https://api.bitbucket.org/2.0/repositories/lep13"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch repositories: %s", string(body))
	}

	var result struct {
		Values []models.Repository `json:"values"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	log.Printf("Fetched %d repositories", len(result.Values))
	return result.Values, nil
}

func fetchCommits(accessToken, repoSlug string) ([]models.CommitDetails, error) {
	url := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/lep13/%s/commits", repoSlug)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch commits: %s", string(body))
	}

	var result struct {
		Values []models.CommitDetails `json:"values"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	log.Printf("Fetched %d commits for repository %s", len(result.Values), repoSlug)
	return result.Values, nil
}
