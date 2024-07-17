package bitbucket

import (
    "github.com/lep13/bitbucket_metrics/internal/database"
    "context"
    "encoding/base64"
    "log"
    "net/http"
    "time"

    "github.com/shurcooL/graphql"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo/options"
)

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

func initClient(username, appPassword string) {
    basicAuth := base64.StdEncoding.EncodeToString([]byte(username + ":" + appPassword))
    httpClient := &http.Client{
        Transport: &transport{underlyingTransport: http.DefaultTransport, basicAuth: basicAuth},
    }
    client = graphql.NewClient("https://api.bitbucket.org/graphql", httpClient)
}

type transport struct {
    underlyingTransport http.RoundTripper
    basicAuth           string
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
    req.Header.Add("Authorization", "Basic "+t.basicAuth)
    return t.underlyingTransport.RoundTrip(req)
}

func FetchAndSaveCommits(username, appPassword string) error {
    initClient(username, appPassword)

    var query struct {
        Projects struct {
            Values []struct {
                Name  graphql.String
                Repos struct {
                    Values []struct {
                        Name    graphql.String
                        Commits struct {
                            Values []struct {
                                Message   graphql.String
                                Author    struct {
                                    Name graphql.String
                                }
                                Date        graphql.DateTime
                                Hash        graphql.String
                                LinesAdded  graphql.Int
                                LinesRemoved graphql.Int
                            }
                        }
                    }
                }
            }
        } `graphql:"projects"`
    }

    err := client.Query(context.Background(), &query, nil)
    if err != nil {
        return err
    }

    for _, project := range query.Projects.Values {
        for _, repo := range project.Repos.Values {
            for _, commit := range repo.Commits.Values {
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

                collection := database.GetCollection()
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

    return nil
}
