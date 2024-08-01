package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/lep13/bitbucket_metrics/config"
	db "github.com/lep13/bitbucket_metrics/internal/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var cfg *config.Config

// HTTPClient defines the methods that our client should implement
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

var httpClient HTTPClient = &http.Client{}

// var loadConfigFunc = config.LoadConfig

func init() {
	var err error
	cfg, err = config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
}

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
			detailedCommit, err := fetchCommitDetails(accessToken, repo.Slug, commit.Hash)
			if err != nil {
				log.Printf("Failed to fetch detailed commit info for %s: %v", commit.Hash, err)
				continue
			}

			filesAdded, filesDeleted, filesUpdated := 0, 0, 0

			for _, file := range detailedCommit.Files {
				switch file.Type {
				case "added":
					filesAdded++
				case "removed":
					filesDeleted++
				case "modified":
					filesUpdated++
				}
			}

			newCommit := Commit{
				ProjectName:   repo.Project.Name,
				RepoName:      repo.Name,
				CommitMessage: detailedCommit.Message,
				CommitID:      detailedCommit.Hash,
				CommittedBy:   detailedCommit.Author.User.DisplayName,
				LinesAdded:    detailedCommit.Summary.LinesAdded,
				LinesDeleted:  detailedCommit.Summary.LinesDeleted,
				CommitDate:    detailedCommit.Date,
				FilesAdded:    filesAdded,
				FilesDeleted:  filesDeleted,
				FilesUpdated:  filesUpdated,
				ReviewedBy:    detailedCommit.ReviewedBy.User.DisplayName,
				PullRequestID: detailedCommit.PullRequest.ID,
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

func fetchRepositories(accessToken string) ([]Repository, error) {
	url := fmt.Sprintf(cfg.RepoURLTemplate, "lep13")
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch repositories: %s", string(body))
	}

	var result struct {
		Values []Repository `json:"values"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	log.Printf("Fetched %d repositories", len(result.Values))
	return result.Values, nil
}

func fetchCommits(accessToken, repoSlug string) ([]CommitDetails, error) {
	url := fmt.Sprintf(cfg.CommitsURLTemplate, "lep13", repoSlug)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch commits: %s", string(body))
	}

	var result struct {
		Values []CommitDetails `json:"values"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	log.Printf("Fetched %d commits for repository %s", len(result.Values), repoSlug)
	return result.Values, nil
}

func fetchCommitDetails(accessToken, repoSlug, commitHash string) (CommitDetails, error) {
	commitURL := fmt.Sprintf(cfg.CommitURLTemplate, "lep13", repoSlug, commitHash)
	commitReq, _ := http.NewRequest("GET", commitURL, nil)
	commitReq.Header.Set("Authorization", "Bearer "+accessToken)

	commitResp, err := httpClient.Do(commitReq)
	if err != nil {
		return CommitDetails{}, err
	}
	defer commitResp.Body.Close()

	if commitResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(commitResp.Body)
		return CommitDetails{}, fmt.Errorf("failed to fetch commit details: %s", string(body))
	}

	var commitDetails CommitDetails
	if err := json.NewDecoder(commitResp.Body).Decode(&commitDetails); err != nil {
		return CommitDetails{}, err
	}

	diffstatURL := fmt.Sprintf(cfg.DiffstatURLTemplate, "lep13", repoSlug, commitHash)
	diffstatReq, _ := http.NewRequest("GET", diffstatURL, nil)
	diffstatReq.Header.Set("Authorization", "Bearer "+accessToken)

	diffstatResp, err := httpClient.Do(diffstatReq)
	if err != nil {
		return CommitDetails{}, err
	}
	defer diffstatResp.Body.Close()

	if diffstatResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(diffstatResp.Body)
		return CommitDetails{}, fmt.Errorf("failed to fetch diffstat: %s", string(body))
	}

	var diffstatResult struct {
		Values []struct {
			Type string `json:"type"`
			Path struct {
				To string `json:"to"`
			} `json:"path"`
		} `json:"values"`
	}
	if err := json.NewDecoder(diffstatResp.Body).Decode(&diffstatResult); err != nil {
		return CommitDetails{}, err
	}

	commitDetails.Files = make([]struct {
		Type string `json:"type"`
		Path string `json:"path"`
	}, len(diffstatResult.Values))
	for i, file := range diffstatResult.Values {
		commitDetails.Files[i].Type = file.Type
		commitDetails.Files[i].Path = file.Path.To
	}

	log.Printf("Fetched detailed commit information and diffstat for commit %s", commitHash)
	return commitDetails, nil
}