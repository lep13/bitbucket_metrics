package bitbucket

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	db "github.com/lep13/bitbucket_metrics/internal/database"
	"github.com/shurcooL/graphql"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DateTime struct {
	time.Time
}

func (dt *DateTime) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}

	dt.Time = t
	return nil
}

type Commit struct {
	CommitMessage string    `bson:"commit_message"`
	LinesDeleted  int       `bson:"lines_deleted"`
	CommitID      string    `bson:"commit_id"`
	CommittedBy   string    `bson:"committed_by"`
	LinesAdded    int       `bson:"lines_added"`
	RepoName      string    `bson:"repo_name"`
	CommitDate    time.Time `bson:"commit_date"`
	FilesAdded    int       `bson:"files_added"`
	FilesDeleted  int       `bson:"files_deleted"`
	FilesUpdated  int       `bson:"files_updated"`
	ProjectName   string    `bson:"project_name"`
}

var client *graphql.Client

func initClient(accessToken string) {
	httpClient := &http.Client{
		Transport: &transport{underlyingTransport: http.DefaultTransport, accessToken: accessToken},
	}
	client = graphql.NewClient("https://api.atlassian.com/graphql", httpClient)
}

type transport struct {
	underlyingTransport http.RoundTripper
	accessToken         string
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.accessToken)
	log.Printf("Making request to: %s with Authorization: Bearer %s", req.URL, t.accessToken) // Log the request details for debugging
	resp, err := t.underlyingTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	log.Printf("Response status: %s, body: %s", resp.Status, string(bodyBytes)) // Log the response status and body for debugging
	return resp, nil
}

func FetchAndSaveCommits(accessToken string) error {
	initClient(accessToken)

	var query struct {
		Viewer struct {
			Repositories struct {
				Edges []struct {
					Node struct {
						Name     graphql.String
						Projects struct {
							Edges []struct {
								Node struct {
									Name  graphql.String
									Repos struct {
										Edges []struct {
											Node struct {
												Name    graphql.String
												Commits struct {
													Edges []struct {
														Node struct {
															Message graphql.String
															Author  struct {
																Name graphql.String
															}
															Date         DateTime
															Hash         graphql.String
															LinesAdded   graphql.Int
															LinesRemoved graphql.Int
														}
													}
												} `graphql:"commits(last: 10)"`
											}
										}
									} `graphql:"repositories(last: 10)"`
								}
							}
						} `graphql:"projects(last: 10)"`
					}
				}
			} `graphql:"repositories(last: 10)"`
		}
	}

	err := client.Query(context.Background(), &query, nil)
	if err != nil {
		log.Printf("Error querying Bitbucket API: %v", err) // Log the error for debugging
		return err
	}

	for _, repoEdge := range query.Viewer.Repositories.Edges {
		repo := repoEdge.Node
		for _, projectEdge := range repo.Projects.Edges {
			project := projectEdge.Node
			for _, repoEdge := range project.Repos.Edges {
				repo := repoEdge.Node
				for _, commitEdge := range repo.Commits.Edges {
					commit := commitEdge.Node
					newCommit := Commit{
						CommitMessage: string(commit.Message),
						LinesDeleted:  int(commit.LinesRemoved),
						CommitID:      string(commit.Hash),
						CommittedBy:   string(commit.Author.Name),
						LinesAdded:    int(commit.LinesAdded),
						RepoName:      string(repo.Name),
						CommitDate:    commit.Date.Time,
						ProjectName:   string(project.Name),
					}

					collection := db.GetCollection()
					_, err := collection.UpdateOne(
						context.Background(),
						bson.M{"commit_id": newCommit.CommitID},
						bson.M{"$set": newCommit},
						options.Update().SetUpsert(true),
					)
					if err != nil {
						log.Printf("Failed to upsert commit %s: %v", newCommit.CommitID, err)
					}
				}
			}
		}
	}

	return nil
}
