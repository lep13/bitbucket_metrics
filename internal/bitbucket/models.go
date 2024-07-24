package bitbucket

import (
	"time"
)

// Repository struct
type Repository struct {
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	Project struct {
		Name string `json:"name"`
	} `json:"project"`
}

// CommitDetails struct
type CommitDetails struct {
	Message string    `json:"message"`
	Hash    string    `json:"hash"`
	Date    time.Time `json:"date"`
	Summary struct {
		LinesAdded   int `json:"lines_added"`
		LinesDeleted int `json:"lines_deleted"`
	} `json:"summary"`
	Author struct {
		User struct {
			DisplayName string `json:"display_name"`
		} `json:"user"`
	} `json:"author"`
	Files []struct {
		Type string `json:"type"`
		Path string `json:"path"`
	} `json:"values"`
	ReviewedBy struct {
		User struct {
			DisplayName string `json:"display_name"`
		} `json:"user"`
	} `json:"reviewed_by,omitempty"`
	PullRequest struct {
		ID string `json:"id"`
	} `json:"pullrequest,omitempty"`
}

// Commit struct
type Commit struct {
	ProjectName   string    `bson:"project_name"`
	RepoName      string    `bson:"repo_name"`
	CommitMessage string    `bson:"commit_message"`
	LinesDeleted  int       `bson:"lines_deleted"`
	CommitID      string    `bson:"commit_id"`
	CommittedBy   string    `bson:"committed_by"`
	LinesAdded    int       `bson:"lines_added"`
	CommitDate    time.Time `bson:"commit_date"`
	FilesAdded    int       `bson:"files_added"`
	FilesDeleted  int       `bson:"files_deleted"`
	FilesUpdated  int       `bson:"files_updated"`
	ReviewedBy    string    `bson:"reviewed_by,omitempty"`
	PullRequestID string    `bson:"pull_request_id,omitempty"`
}